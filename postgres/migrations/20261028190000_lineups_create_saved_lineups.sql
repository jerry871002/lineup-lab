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
