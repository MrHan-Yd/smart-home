-- smart-home P2 schema: analytics + telemetry
-- Aligns with docs/backend/数据库设计.md §3.8 (daily_device_stats) / §3.9 (telemetry_samples)

-- 3.9 telemetry_samples: 逐条状态采样（WS state_changed 落库）
CREATE TABLE IF NOT EXISTS telemetry_samples (
    device_id  UUID NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    entity_id  TEXT NOT NULL,
    ts         TIMESTAMPTZ NOT NULL,
    state      TEXT,
    num_value  DOUBLE PRECISION,
    attributes JSONB
);

CREATE INDEX IF NOT EXISTS idx_telemetry_entity_ts ON telemetry_samples (entity_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_telemetry_device_ts ON telemetry_samples (device_id, ts DESC);

-- 3.8 daily_device_stats: 日聚合（懒聚合，analytics 接口触发）
CREATE TABLE IF NOT EXISTS daily_device_stats (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    device_id       UUID NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    stat_date       DATE NOT NULL,
    on_count        INT NOT NULL DEFAULT 0,
    on_duration_sec BIGINT NOT NULL DEFAULT 0,
    energy_kwh      NUMERIC(12,4),
    extra           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (device_id, stat_date)
);

CREATE INDEX IF NOT EXISTS idx_daily_stats_user_date ON daily_device_stats (user_id, stat_date);