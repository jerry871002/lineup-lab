-- owner: lineups
-- Current-lineup experiments have no saved revision; saved-lineup runs retain theirs.
ALTER TABLE simulation_runs ALTER COLUMN lineup_revision DROP NOT NULL;
