-- NetLab Backend — Initial Schema
-- PostgreSQL 15+

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(64) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar VARCHAR(512),
    roles JSONB NOT NULL DEFAULT '["viewer"]',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    failed_login_attempts INT DEFAULT 0,
    locked_until TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_username ON users(username);
CREATE UNIQUE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- Passkey credentials table
CREATE TABLE passkey_credentials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id TEXT NOT NULL,
    public_key TEXT NOT NULL,
    attestation_type VARCHAR(64) NOT NULL,
    transports JSONB,
    flags JSONB,
    authenticator JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_passkey_credentials_cred_id ON passkey_credentials(credential_id);
CREATE INDEX idx_passkey_credentials_user ON passkey_credentials(user_id);

-- System configurations table
CREATE TABLE system_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key VARCHAR(128) NOT NULL,
    value JSONB NOT NULL,
    description VARCHAR(512),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_system_configs_key ON system_configs(key);

-- Seed default configurations
INSERT INTO system_configs (key, value, description) VALUES
    ('registration_enabled', 'true', 'Allow new user registration'),
    ('captcha_enabled', 'false', 'Require image captcha on login and registration'),
    ('passkey_enabled', 'true', 'Enable WebAuthn/Passkey authentication'),
    ('oauth_providers', '[]', 'Configured OAuth providers')
ON CONFLICT (key) DO NOTHING;

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to users table
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to system_configs table
CREATE TRIGGER trg_system_configs_updated_at
    BEFORE UPDATE ON system_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
