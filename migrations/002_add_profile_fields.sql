-- Migration v0.02 -
-- Migration: Adicionar campos de perfil e endereço
-- Date: 2025-11-21

ALTER TABLE users 
ADD COLUMN IF NOT EXISTS phone VARCHAR(50),
ADD COLUMN IF NOT EXISTS job_title VARCHAR(100), -- "Role" no frontend (visual)
ADD COLUMN IF NOT EXISTS location VARCHAR(150),
ADD COLUMN IF NOT EXISTS avatar_url TEXT,

-- Endereço / Faturamento
ADD COLUMN IF NOT EXISTS country VARCHAR(100),
ADD COLUMN IF NOT EXISTS city VARCHAR(100),
ADD COLUMN IF NOT EXISTS state VARCHAR(100),
ADD COLUMN IF NOT EXISTS postal_code VARCHAR(20),
ADD COLUMN IF NOT EXISTS tax_id VARCHAR(50); -- CPF/CNPJ