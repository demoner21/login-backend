-- Migration v0.01 - Sistema de Autenticação PostgreSQL com Snowflake ID
-- Created by: Anderson Demoner
-- Date: 2025-11-20

-- Tabela de roles (simplificada)
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    role_name VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de usuários (core) com Snowflake ID como BIGINT
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    last_password_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    refresh_token TEXT,
    is_email_verified BOOLEAN DEFAULT false,
    totp_secret VARCHAR(255),
    profile_image_url TEXT,
    role_id INTEGER NOT NULL DEFAULT 2,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,

    FOREIGN KEY (role_id) REFERENCES roles(id)
);

-- Inserir roles básicos
INSERT INTO roles (id, role_name, description) VALUES
    (1, 'SUPER_ADMIN', 'Acesso total ao sistema'),
    (2, 'USER', 'Usuário padrão do sistema')
ON CONFLICT (id) DO NOTHING;

-- Criar índices
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role_id ON users(role_id);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);