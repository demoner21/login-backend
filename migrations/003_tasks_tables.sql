-- Migration v0.03 - Tabelas de Tarefas e Teams
-- Reordenado para satisfazer dependências de Foreign Key

-- ============================================
-- 1. TEAMS TABLE (Precisa vir antes de tasks e team_members)
-- ============================================
CREATE TABLE IF NOT EXISTS teams (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    parent_team_id BIGINT REFERENCES teams(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_teams_parent ON teams(parent_team_id);

-- ============================================
-- 2. FARM AREAS (Placeholder para evitar erro de FK em tasks)
-- ============================================
-- Se você ainda não tem essa tabela definida em outra migration, 
-- crie ela aqui ou remova a referência "REFERENCES farm_areas(id)" da tabela tasks abaixo.
CREATE TABLE IF NOT EXISTS farm_areas (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255),
    geojson TEXT
);

-- ============================================
-- 3. TASKS TABLE (Source of Truth)
-- ============================================
CREATE TABLE IF NOT EXISTS tasks (
    id BIGINT PRIMARY KEY, -- Snowflake ID gerado pelo Go
    title VARCHAR(255) NOT NULL,
    description TEXT,
    due_date TIMESTAMP,
    priority VARCHAR(20) CHECK (priority IN ('Low', 'Medium', 'High')),
    status VARCHAR(20) CHECK (status IN ('Pending', 'InProgress', 'Done', 'Canceled')),
    
    -- Ownership & Permissions
    owner_id BIGINT NOT NULL REFERENCES users(id),
    team_id BIGINT REFERENCES teams(id), -- Agora funciona pois teams já existe
    
    -- Geolocation (Agtech specific)
    location_lat DECIMAL(10, 8),
    location_lng DECIMAL(11, 8),
    farm_area_id BIGINT REFERENCES farm_areas(id), -- Referência corrigida
    
    -- Versioning & CRDT
    version BIGINT NOT NULL DEFAULT 1,
    vector_clock JSONB, 
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    
    -- Full-text search column
    tsv tsvector GENERATED ALWAYS AS (
        to_tsvector('portuguese', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED
);

CREATE INDEX IF NOT EXISTS idx_tasks_owner ON tasks(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_team ON tasks(team_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_tsv ON tasks USING GIN(tsv);

-- ============================================
-- 4. EVENTS TABLE (Event Sourcing)
-- ============================================
CREATE TABLE IF NOT EXISTS task_events (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL, -- Removido FK estrita para permitir deletar task mantendo histórico ou adicione ON DELETE CASCADE
    event_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    version BIGINT NOT NULL,
    sequence_number BIGINT NOT NULL,
    vector_clock JSONB,
    user_id BIGINT NOT NULL REFERENCES users(id),
    client_timestamp TIMESTAMP,
    server_timestamp TIMESTAMP DEFAULT NOW(),
    synced_at TIMESTAMP,
    idempotency_key UUID UNIQUE,
    
    CONSTRAINT unique_task_version UNIQUE(task_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_task ON task_events(task_id, version DESC);

-- ============================================
-- 5. SNAPSHOTS TABLE (Performance)
-- ============================================
CREATE TABLE IF NOT EXISTS task_snapshots (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    version BIGINT NOT NULL,
    snapshot_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT unique_task_snapshot UNIQUE(task_id, version)
);

-- ============================================
-- 6. TASK COLLABORATORS (Permissions)
-- ============================================
CREATE TABLE IF NOT EXISTS task_collaborators (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('Owner', 'Editor', 'Viewer')),
    created_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT unique_task_collaborator UNIQUE(task_id, user_id)
);

-- ============================================
-- 7. TEAM MEMBERS (Many-to-Many)
-- ============================================
CREATE TABLE IF NOT EXISTS team_members (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('Admin', 'Member', 'Viewer')),
    joined_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT unique_team_member UNIQUE(team_id, user_id)
);

-- ============================================
-- 8. SYNC QUEUE (Offline Handling)
-- ============================================
CREATE TABLE IF NOT EXISTS sync_queue (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    operation_type VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id BIGINT,
    payload JSONB NOT NULL,
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 5,
    last_error TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_sync_queue_user ON sync_queue(user_id, status);