-- Migration: create_activation_tokens_table
-- Create activation_tokens table for simplified registration flow
-- Activation tokens are used for email verification + auto-login
-- Replaces email_verification_tokens with enhanced functionality

CREATE TABLE activation_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at BIGINT NOT NULL,  -- Unix timestamp - 24 hours after creation
    used_at BIGINT,               -- Unix timestamp when token was used (null if not used)
    created_at BIGINT NOT NULL,
    CONSTRAINT fk_activation_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_activation_tokens_token ON activation_tokens(token);
CREATE INDEX idx_activation_tokens_user_id ON activation_tokens(user_id);
CREATE INDEX idx_activation_tokens_expires_at ON activation_tokens(expires_at);

-- Comments for documentation
COMMENT ON TABLE activation_tokens IS 'Stores activation tokens for email verification + auto-login (simplified flow)';
COMMENT ON COLUMN activation_tokens.token IS 'Unique 64-character hex token (32 random bytes)';
COMMENT ON COLUMN activation_tokens.expires_at IS 'Unix timestamp - token expires 24 hours after creation';
COMMENT ON COLUMN activation_tokens.used_at IS 'Unix timestamp when user clicked activation link (null if not used yet)';
