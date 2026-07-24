-- smart-home P1 schema: HA instances + action audit logs
-- Aligns with docs/backend/数据库设计.md §3.3 / §3.6

-- 3.3 ha_instances：在线录入 HA 连接（P0 可用 env 代替；DB 优先于 env）
CREATE TABLE IF NOT EXISTS ha_instances (
    id              UUID PRIMARY KEY,
    home_id         UUID NOT NULL REFERENCES homes (id) ON DELETE CASCADE,
    name            TEXT NOT NULL DEFAULT 'default',
    base_url        TEXT NOT NULL,
    token_encrypted TEXT NOT NULL,                       -- AES-GCM 密文，禁止明文
    is_active        BOOLEAN NOT NULL DEFAULT true,
    last_ok_at       TIMESTAMPTZ,
    last_error       TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (home_id, name)
);

CREATE INDEX IF NOT EXISTS idx_ha_instances_home ON ha_instances (home_id, is_active);

-- 3.6 device_action_logs：控制审计（P1）
CREATE TABLE IF NOT EXISTS device_action_logs (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL,
    home_id       UUID NOT NULL,
    device_id     UUID REFERENCES devices (id) ON DELETE SET NULL,
    entity_id     TEXT NOT NULL,
    action        TEXT NOT NULL,
    params        JSONB NOT NULL DEFAULT '{}'::jsonb,
    success       BOOLEAN NOT NULL,
    error_message TEXT,
    ha_domain     TEXT,
    ha_service    TEXT,
    duration_ms   INT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_action_logs_user_time ON device_action_logs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_action_logs_entity_time ON device_action_logs (entity_id, created_at DESC);