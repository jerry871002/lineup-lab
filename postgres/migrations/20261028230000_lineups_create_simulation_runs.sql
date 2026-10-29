-- owner: lineups
CREATE TABLE IF NOT EXISTS simulation_runs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    saved_lineup_id BIGINT REFERENCES saved_lineups(id) ON DELETE SET NULL,
    lineup_name VARCHAR(100) NOT NULL CHECK (LENGTH(TRIM(lineup_name)) > 0),
    team VARCHAR(100) NOT NULL CHECK (LENGTH(TRIM(team)) > 0),
    year INT NOT NULL CHECK (year BETWEEN 1900 AND 2100),
    lineup_revision BIGINT NOT NULL CHECK (lineup_revision > 0),
    batter_ids JSONB NOT NULL CHECK (jsonb_typeof(batter_ids) = 'array' AND jsonb_array_length(batter_ids) = 9),
    batters JSONB NOT NULL CHECK (jsonb_typeof(batters) = 'array' AND jsonb_array_length(batters) = 9),
    average_score DOUBLE PRECISION NOT NULL CHECK (average_score >= 0 AND average_score < 'Infinity'::double precision),
    average_hits DOUBLE PRECISION NOT NULL CHECK (average_hits >= 0 AND average_hits < 'Infinity'::double precision),
    sample_count INT NOT NULL CHECK (sample_count > 0),
    simulator_version VARCHAR(100) NOT NULL CHECK (LENGTH(TRIM(simulator_version)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS simulation_runs_owner_created ON simulation_runs(user_id, created_at DESC, id DESC);
