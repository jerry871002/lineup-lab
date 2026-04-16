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
