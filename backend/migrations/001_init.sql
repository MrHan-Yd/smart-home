-- smart-home P0 schema
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    sub           TEXT NOT NULL UNIQUE,
    email         TEXT,
    name          TEXT,
    avatar_url    TEXT,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS homes (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL DEFAULT '我的家',
    is_default BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_homes_user ON homes (user_id);

CREATE TABLE IF NOT EXISTS rooms (
    id         UUID PRIMARY KEY,
    home_id    UUID NOT NULL REFERENCES homes (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    ha_area_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (home_id, name)
);

CREATE INDEX IF NOT EXISTS idx_rooms_home ON rooms (home_id, sort_order);

CREATE TABLE IF NOT EXISTS devices (
    id         UUID PRIMARY KEY,
    home_id    UUID NOT NULL REFERENCES homes (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    entity_id  TEXT NOT NULL,
    domain     TEXT NOT NULL,
    name       TEXT,
    room_id    UUID REFERENCES rooms (id) ON DELETE SET NULL,
    favorite   BOOLEAN NOT NULL DEFAULT false,
    hidden     BOOLEAN NOT NULL DEFAULT false,
    sort_order INT NOT NULL DEFAULT 0,
    icon       TEXT,
    meta       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (home_id, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_devices_user ON devices (user_id, favorite DESC, sort_order, name);
CREATE INDEX IF NOT EXISTS idx_devices_home_domain ON devices (home_id, domain);
