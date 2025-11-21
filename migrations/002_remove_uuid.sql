-- Migration para remover UUID e usar ID sequencial
-- Created by: Anderson Demoner
-- Date: 2025-11-21

-- Criar tabela temporária
CREATE TABLE users_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    last_password_update TIMESTAMP DEFAULT current_timestamp,
    refresh_token TEXT,
    is_email_verified BOOLEAN DEFAULT false,
    totp_secret VARCHAR(255),
    profile_image_url TEXT,
    role_id INTEGER NOT NULL DEFAULT 2,
    created_at TIMESTAMP DEFAULT current_timestamp,
    updated_at TIMESTAMP DEFAULT current_timestamp,
    last_login_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    
    FOREIGN KEY (role_id) REFERENCES roles(id)
);

-- Copiar dados se existirem
INSERT INTO users_new (
    email, name, password_hash, last_password_update, refresh_token,
    is_email_verified, totp_secret, profile_image_url, role_id,
    created_at, updated_at, last_login_at, is_active
)
SELECT 
    email, name, password_hash, last_password_update, refresh_token,
    is_email_verified, totp_secret, profile_image_url, role_id,
    created_at, updated_at, last_login_at, is_active
FROM users;

-- Dropar tabela antiga
DROP TABLE users;

-- Renomear nova tabela
ALTER TABLE users_new RENAME TO users;

-- Recriar índices
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role_id ON users(role_id);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);