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
    role VARCHAR(32) NOT NULL DEFAULT 'viewer',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    force_password_change BOOLEAN NOT NULL DEFAULT FALSE,
    force_email_change BOOLEAN NOT NULL DEFAULT FALSE,
    two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    two_factor_secret VARCHAR(512),
    preferred_auth_method VARCHAR(16) NOT NULL DEFAULT 'totp',
    password_changed_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_username ON users(username);
CREATE UNIQUE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- Unified third-party authentication bindings table.
-- OAuth and Passkey share this table. Passkey rows use provider='passkey'
-- and store WebAuthn credential JSON in credential-related columns.
CREATE TABLE auth_bindings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    provider_user_id VARCHAR(191) NOT NULL,
    email VARCHAR(255),
    credential_id TEXT,
    credential JSONB,
    name VARCHAR(128),
    sign_count BIGINT NOT NULL DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_oauth_provider_uid ON auth_bindings(provider, provider_user_id);
CREATE UNIQUE INDEX idx_auth_bindings_credential_id ON auth_bindings(credential_id) WHERE credential_id IS NOT NULL;
CREATE INDEX idx_auth_bindings_user ON auth_bindings(user_id);

-- Two-factor recovery codes (one-time use, stored as SHA-256 hashes)
CREATE TABLE recovery_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(64) NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recovery_codes_user ON recovery_codes(user_id);
CREATE INDEX idx_recovery_codes_hash ON recovery_codes(code_hash);

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
    ('password_reset_enabled', 'true', 'Enable password reset via email'),
    ('password_max_age_days', '0', 'Password max age in days; 0 means never expires'),
    ('two_factor_required', 'false', 'Require two-factor authentication for backend access')
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
