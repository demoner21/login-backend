-- Migration v0.04 - ACL System com Bitmask (CORRIGIDO)
-- Suporta compartilhamento de recursos via Snowflake ID

-- ============================================
-- 1. TABELA PRINCIPAL DE ACLs
-- ============================================
CREATE TABLE IF NOT EXISTS acls (
    id BIGSERIAL PRIMARY KEY,
    
    -- Recurso sendo controlado
    resource_id BIGINT NOT NULL,
    resource_type VARCHAR(20) NOT NULL CHECK (
        resource_type IN ('TASK', 'FARM_AREA', 'TEAM', 'DOCUMENT')
    ),
    
    -- Entidade que recebeu acesso
    grantee_type VARCHAR(10) NOT NULL CHECK (
        grantee_type IN ('USER', 'TEAM', 'PUBLIC')
    ),
    grantee_id BIGINT, -- NULL se PUBLIC
    
    -- Permissões (bitmask)
    permissions INTEGER NOT NULL DEFAULT 0,
    
    -- Auditoria
    granted_by BIGINT NOT NULL REFERENCES users(id),
    granted_at TIMESTAMP DEFAULT NOW(),
    
    -- Links temporários
    expires_at TIMESTAMP,
    
    -- Metadados flexíveis
    metadata JSONB DEFAULT '{}',
    
    -- Constraints
    CONSTRAINT chk_grantee_id CHECK (
        (grantee_type = 'PUBLIC' AND grantee_id IS NULL) OR
        (grantee_type != 'PUBLIC' AND grantee_id IS NOT NULL)
    ),
    CONSTRAINT unique_acl UNIQUE (resource_id, resource_type, grantee_type, grantee_id)
);

-- CORREÇÃO AQUI: Removemos o "OR expires_at > NOW()" dos índices.
-- O filtro de tempo deve ser feito apenas na query (SELECT), não na estrutura do índice.
CREATE INDEX idx_acls_resource ON acls(resource_id, resource_type);

CREATE INDEX idx_acls_grantee ON acls(grantee_id, grantee_type);

CREATE INDEX idx_acls_expires ON acls(expires_at) 
    WHERE expires_at IS NOT NULL;

-- ============================================
-- 2. CACHE DE PERMISSÕES (Performance)
-- ============================================
CREATE TABLE IF NOT EXISTS resource_permissions_cache (
    user_id BIGINT NOT NULL,
    resource_id BIGINT NOT NULL,
    resource_type VARCHAR(20) NOT NULL,
    
    -- Permissão efetiva (agregada de todas as ACLs)
    effective_permissions INTEGER NOT NULL,
    
    -- TTL do cache
    cached_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP DEFAULT NOW() + INTERVAL '1 hour',
    
    PRIMARY KEY (user_id, resource_id, resource_type)
);

-- CORREÇÃO AQUI: Removemos o filtro de tempo.
CREATE INDEX idx_cache_user ON resource_permissions_cache(user_id);

-- ============================================
-- 3. FUNÇÃO PARA CALCULAR PERMISSÕES EFETIVAS
-- ============================================
CREATE OR REPLACE FUNCTION calculate_effective_permissions(
    p_user_id BIGINT,
    p_resource_id BIGINT,
    p_resource_type VARCHAR(20)
) RETURNS INTEGER AS $$
DECLARE
    v_permissions INTEGER := 0;
    v_user_teams BIGINT[];
BEGIN
    -- 1. Buscar times do usuário
    SELECT ARRAY_AGG(team_id) INTO v_user_teams
    FROM team_members
    WHERE user_id = p_user_id;
    
    -- 2. Agregar permissões diretas (USER)
    SELECT COALESCE(BIT_OR(permissions), 0) INTO v_permissions
    FROM acls
    WHERE resource_id = p_resource_id
      AND resource_type = p_resource_type
      AND grantee_type = 'USER'
      AND grantee_id = p_user_id
      AND (expires_at IS NULL OR expires_at > NOW());
    
    -- 3. Agregar permissões via TEAM
    IF v_user_teams IS NOT NULL THEN
        SELECT v_permissions | COALESCE(BIT_OR(permissions), 0) INTO v_permissions
        FROM acls
        WHERE resource_id = p_resource_id
          AND resource_type = p_resource_type
          AND grantee_type = 'TEAM'
          AND grantee_id = ANY(v_user_teams)
          AND (expires_at IS NULL OR expires_at > NOW());
    END IF;
    
    -- 4. Agregar permissões PUBLIC
    SELECT v_permissions | COALESCE(BIT_OR(permissions), 0) INTO v_permissions
    FROM acls
    WHERE resource_id = p_resource_id
      AND resource_type = p_resource_type
      AND grantee_type = 'PUBLIC'
      AND (expires_at IS NULL OR expires_at > NOW());
    
    RETURN v_permissions;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 4. TRIGGER PARA INVALIDAR CACHE
-- ============================================
CREATE OR REPLACE FUNCTION invalidate_permissions_cache()
RETURNS TRIGGER AS $$
BEGIN
    -- Invalida cache para todos os usuários afetados
    DELETE FROM resource_permissions_cache
    WHERE resource_id = COALESCE(NEW.resource_id, OLD.resource_id)
      AND resource_type = COALESCE(NEW.resource_type, OLD.resource_type);
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_acls_cache_invalidate
AFTER INSERT OR UPDATE OR DELETE ON acls
FOR EACH ROW EXECUTE FUNCTION invalidate_permissions_cache();

-- ============================================
-- 5. VIEW PARA CONSULTAS SIMPLIFICADAS
-- ============================================
CREATE OR REPLACE VIEW v_resource_access AS
SELECT 
    a.resource_id,
    a.resource_type,
    a.grantee_type,
    a.grantee_id,
    u.name AS grantee_name,
    a.permissions,
    (a.permissions & 1)  > 0 AS can_read,
    (a.permissions & 2)  > 0 AS can_write,
    (a.permissions & 4)  > 0 AS can_delete,
    (a.permissions & 8)  > 0 AS can_share,
    (a.permissions & 16) > 0 AS is_admin,
    a.granted_by,
    grantor.name AS granted_by_name,
    a.granted_at,
    a.expires_at
FROM acls a
LEFT JOIN users u ON a.grantee_id = u.id
LEFT JOIN users grantor ON a.granted_by = grantor.id
WHERE a.expires_at IS NULL OR a.expires_at > NOW();

-- ============================================
-- 6. ÍNDICE PARA OWNER_ID (Para queries de "recursos que eu criei")
-- ============================================
-- Nota: "deleted_at IS NULL" funciona porque não é uma função de tempo, é um estado estático.
CREATE INDEX IF NOT EXISTS idx_tasks_owner_active ON tasks(owner_id) 
    WHERE deleted_at IS NULL;

-- ============================================
-- 7. JOB DE LIMPEZA DE EXPIRADOS (Cronjob externo ou pg_cron)
-- ============================================
CREATE OR REPLACE FUNCTION cleanup_expired_acls()
RETURNS void AS $$
BEGIN
    DELETE FROM acls WHERE expires_at < NOW();
    DELETE FROM resource_permissions_cache WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;