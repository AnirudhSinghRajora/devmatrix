-- 001_initial_schema.sql

CREATE TABLE IF NOT EXISTS items (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    category        TEXT NOT NULL,
    stats           JSONB NOT NULL DEFAULT '{}',
    price           INTEGER NOT NULL CHECK (price >= 0),
    tier_required   INTEGER NOT NULL DEFAULT 1,
    description     TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        TEXT NOT NULL UNIQUE,
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

CREATE TABLE IF NOT EXISTS profiles (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    coins       INTEGER NOT NULL DEFAULT 0 CHECK (coins >= 0),
    kills       INTEGER NOT NULL DEFAULT 0,
    deaths      INTEGER NOT NULL DEFAULT 0,
    ai_tier     INTEGER NOT NULL DEFAULT 1 CHECK (ai_tier BETWEEN 1 AND 5),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS loadouts (
    user_id          UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    hull_id          TEXT NOT NULL DEFAULT 'hull_basic' REFERENCES items(id),
    primary_weapon   TEXT NOT NULL DEFAULT 'wpn_laser_1' REFERENCES items(id),
    secondary_weapon TEXT REFERENCES items(id),
    shield_id        TEXT NOT NULL DEFAULT 'shld_basic' REFERENCES items(id),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS inventory (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id     TEXT NOT NULL REFERENCES items(id),
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, item_id)
);

CREATE TABLE IF NOT EXISTS kill_log (
    id          BIGSERIAL PRIMARY KEY,
    killer_id   UUID NOT NULL REFERENCES users(id),
    victim_id   UUID NOT NULL REFERENCES users(id),
    killed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_kill_log_killer ON kill_log (killer_id, killed_at DESC);
CREATE INDEX IF NOT EXISTS idx_kill_log_time ON kill_log (killed_at DESC);
