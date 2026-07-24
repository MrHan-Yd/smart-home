-- smart-home P3 schema: composite devices, multi-step scenarios, multi-HA switching
-- Aligns with docs/backend/设备类型与接入设计.md (组合设备) / 功能设计.md §7 (本系统多步场景)

-- 组合设备成员：一个 device 行承载，device_members 显式记录多 entity 绑定
CREATE TABLE IF NOT EXISTS device_members (
    device_id  UUID NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    entity_id  TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'member',  -- primary | member
    sort_order INT  NOT NULL DEFAULT 0,
    PRIMARY KEY (device_id, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_device_members_entity ON device_members (entity_id);

-- 多步场景：本系统存步骤，一键串行执行
CREATE TABLE IF NOT EXISTS scenarios (
    id          UUID PRIMARY KEY,
    home_id     UUID NOT NULL REFERENCES homes (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    icon        TEXT,
    room_id     UUID REFERENCES rooms (id) ON DELETE SET NULL,
    sort_order  INT  NOT NULL DEFAULT 0,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    last_run_at TIMESTAMPTZ,
    run_count   INT  NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_scenarios_home ON scenarios (home_id, sort_order, name);

CREATE TABLE IF NOT EXISTS scenario_steps (
    id          UUID PRIMARY KEY,
    scenario_id UUID NOT NULL REFERENCES scenarios (id) ON DELETE CASCADE,
    sort_order  INT  NOT NULL DEFAULT 0,
    device_id   UUID NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    action      TEXT NOT NULL,
    params      JSONB NOT NULL DEFAULT '{}'::jsonb,
    delay_ms    INT  NOT NULL DEFAULT 0,   -- 前置等待
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scenario_id, sort_order)
);

CREATE INDEX IF NOT EXISTS idx_scenario_steps ON scenario_steps (scenario_id, sort_order);

-- ha_instances 切换：is_active 已存在；加唯一部分索引保证每 home 至多一个 active
CREATE UNIQUE INDEX IF NOT EXISTS idx_ha_instances_active
    ON ha_instances (home_id) WHERE is_active = true;