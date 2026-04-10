CREATE TABLE IF NOT EXISTS push_devices (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id VARCHAR(255) NOT NULL UNIQUE,
    platform VARCHAR(16) NOT NULL CHECK (platform IN ('android', 'ios')),
    fcm_token TEXT NOT NULL UNIQUE,
    app_version VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS push_devices_user_id_idx ON push_devices(user_id);
CREATE INDEX IF NOT EXISTS push_devices_updated_at_idx ON push_devices(updated_at DESC);
