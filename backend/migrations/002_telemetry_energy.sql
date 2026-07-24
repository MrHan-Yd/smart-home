-- 电耗 / 功率 / 传感器时序入库（统计分析用）
-- 依赖：001_init.sql（users / homes / devices）

-- 同步水位：按 entity 记录上次从 HA 拉 history 的进度
CREATE TABLE IF NOT EXISTS sync_cursors (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    home_id         UUID NOT NULL REFERENCES homes (id) ON DELETE CASCADE,
    device_id       UUID REFERENCES devices (id) ON DELETE SET NULL,
    entity_id       TEXT NOT NULL,
    metric          TEXT NOT NULL DEFAULT 'state',
    -- metric: power | energy | state | temperature | humidity | ...
    last_ts         TIMESTAMPTZ,
    last_success_at TIMESTAMPTZ,
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (home_id, entity_id, metric)
);

CREATE INDEX IF NOT EXISTS idx_sync_cursors_user ON sync_cursors (user_id);
CREATE INDEX IF NOT EXISTS idx_sync_cursors_due ON sync_cursors (last_success_at NULLS FIRST);

-- 时序采样点（功率 / 电量 / 其它数值 sensor）
CREATE TABLE IF NOT EXISTS telemetry_samples (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    home_id     UUID NOT NULL REFERENCES homes (id) ON DELETE CASCADE,
    device_id   UUID REFERENCES devices (id) ON DELETE SET NULL,
    entity_id   TEXT NOT NULL,
    metric      TEXT NOT NULL,
    -- power: 瞬时功率(W)  energy: 累计电量(kWh)  其它 metric 自定
    ts          TIMESTAMPTZ NOT NULL,
    num_value   DOUBLE PRECISION,
    state_raw   TEXT,
    unit        TEXT,
    source      TEXT NOT NULL DEFAULT 'ha_history',
    -- ha_history | ha_state | estimated
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, metric, ts)
);

CREATE INDEX IF NOT EXISTS idx_telemetry_entity_ts ON telemetry_samples (entity_id, metric, ts DESC);
CREATE INDEX IF NOT EXISTS idx_telemetry_home_ts ON telemetry_samples (home_id, metric, ts DESC);
CREATE INDEX IF NOT EXISTS idx_telemetry_device_ts ON telemetry_samples (device_id, ts DESC)
    WHERE device_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_telemetry_user_ts ON telemetry_samples (user_id, ts DESC);

-- 按日汇总（分析页主查表，避免扫全量时序）
CREATE TABLE IF NOT EXISTS daily_device_stats (
    id               UUID PRIMARY KEY,
    user_id          UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    home_id          UUID NOT NULL REFERENCES homes (id) ON DELETE CASCADE,
    device_id        UUID NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    entity_id        TEXT NOT NULL,
    stat_date        DATE NOT NULL,
    on_count         INT NOT NULL DEFAULT 0,
    on_duration_sec  BIGINT NOT NULL DEFAULT 0,
    energy_kwh       NUMERIC(14, 6),
    -- 当日用电量（优先 energy 差分；否则 power 积分估算）
    energy_source    TEXT,
    -- meter_diff | power_integral | mixed | none
    avg_power_w      NUMERIC(14, 4),
    max_power_w      NUMERIC(14, 4),
    sample_count     INT NOT NULL DEFAULT 0,
    extra            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (device_id, stat_date)
);

CREATE INDEX IF NOT EXISTS idx_daily_stats_user_date ON daily_device_stats (user_id, stat_date DESC);
CREATE INDEX IF NOT EXISTS idx_daily_stats_home_date ON daily_device_stats (home_id, stat_date DESC);
CREATE INDEX IF NOT EXISTS idx_daily_stats_energy ON daily_device_stats (user_id, stat_date)
    WHERE energy_kwh IS NOT NULL;

COMMENT ON TABLE telemetry_samples IS 'HA 同步的时序点；统计分析读此表/日汇总，不实时扫 HA';
COMMENT ON TABLE daily_device_stats IS '按日聚合：开启时长 + 用电 kWh 等';
COMMENT ON TABLE sync_cursors IS 'history 增量同步水位';
