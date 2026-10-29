CREATE TABLE batting (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    team VARCHAR(100) NOT NULL,
    year INT NOT NULL CHECK (year >= 1900 AND year <= 2100),
    at_bat INT NOT NULL CHECK (at_bat >= 0),
    hit INT NOT NULL CHECK (hit >= 0),
    double INT NOT NULL CHECK (double >= 0),
    triple INT NOT NULL CHECK (triple >= 0),
    home_run INT NOT NULL CHECK (home_run >= 0),
    ball_on_base INT NOT NULL CHECK (ball_on_base >= 0),
    hit_by_pitch INT NOT NULL CHECK (hit_by_pitch >= 0),
    CONSTRAINT unique_constraint UNIQUE (name, team, year)
);

CREATE TABLE leaderboard (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    team VARCHAR(100) NOT NULL,
    score FLOAT NOT NULL
);

-- Users are created by the auth service layer. Email and username stay unique
-- so they can both serve lookup and display use cases later.
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,
    CONSTRAINT users_username_unique UNIQUE (username),
    CONSTRAINT users_email_unique UNIQUE (email)
);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS users_set_updated_at ON users;
CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Sessions are stored server-side. The application generates the UUID session ID
-- and can revoke sessions without deleting the historical row immediately.
CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    user_agent TEXT,
    ip_address INET
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_sessions_active_user_id ON sessions(user_id, expires_at)
WHERE revoked_at IS NULL;
-- owner: lineups
CREATE TABLE IF NOT EXISTS saved_lineups (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL CHECK (LENGTH(TRIM(name)) > 0),
    team VARCHAR(100) NOT NULL CHECK (LENGTH(TRIM(team)) > 0),
    year INT NOT NULL CHECK (year BETWEEN 1900 AND 2100),
    batter_ids JSONB NOT NULL CHECK (jsonb_typeof(batter_ids) = 'array' AND jsonb_array_length(batter_ids) = 9),
    batters JSONB NOT NULL CHECK (jsonb_typeof(batters) = 'array' AND jsonb_array_length(batters) = 9),
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT saved_lineups_owner_name_team_season UNIQUE (user_id, name, team, year)
);
CREATE INDEX IF NOT EXISTS saved_lineups_owner_updated ON saved_lineups(user_id, updated_at DESC, id DESC);

-- owner: lineups
CREATE TABLE IF NOT EXISTS simulation_runs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    saved_lineup_id BIGINT REFERENCES saved_lineups(id) ON DELETE SET NULL,
    lineup_name VARCHAR(100) NOT NULL CHECK (LENGTH(TRIM(lineup_name)) > 0),
    team VARCHAR(100) NOT NULL CHECK (LENGTH(TRIM(team)) > 0),
    year INT NOT NULL CHECK (year BETWEEN 1900 AND 2100),
    lineup_revision BIGINT CHECK (lineup_revision > 0),
    batter_ids JSONB NOT NULL CHECK (jsonb_typeof(batter_ids) = 'array' AND jsonb_array_length(batter_ids) = 9),
    batters JSONB NOT NULL CHECK (jsonb_typeof(batters) = 'array' AND jsonb_array_length(batters) = 9),
    average_score DOUBLE PRECISION NOT NULL CHECK (average_score >= 0 AND average_score < 'Infinity'::double precision),
    average_hits DOUBLE PRECISION NOT NULL CHECK (average_hits >= 0 AND average_hits < 'Infinity'::double precision),
    sample_count INT NOT NULL CHECK (sample_count > 0),
    simulator_version VARCHAR(100) NOT NULL CHECK (LENGTH(TRIM(simulator_version)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS simulation_runs_owner_created ON simulation_runs(user_id, created_at DESC, id DESC);
