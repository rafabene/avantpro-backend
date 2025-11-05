-- Migration: create_users_table

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(500) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    avatar_url VARCHAR(500),
    created_at BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from now()),
    deleted_at BIGINT
);

-- Índices
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- Comentários
COMMENT ON TABLE users IS 'User accounts';
COMMENT ON COLUMN users.email IS 'User email (unique)';
COMMENT ON COLUMN users.role IS 'User role: admin, user, guest';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password';
COMMENT ON COLUMN users.deleted_at IS 'Soft delete timestamp (NULL = not deleted)';
