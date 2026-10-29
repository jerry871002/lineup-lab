package main

import (
	"context"
	"database/sql"
	"encoding/json"
)

const runColumns = "id,saved_lineup_id,lineup_name,team,year,lineup_revision,batter_ids,batters,average_score,average_hits,sample_count,simulator_version,created_at"

func scanRun(row scanner) (SimulationRun, error) {
	var run SimulationRun
	var ids, batters []byte
	var revision sql.NullInt64
	if err := row.Scan(&run.ID, &run.SavedLineupID, &run.LineupName, &run.Team, &run.Year, &revision, &ids, &batters, &run.AverageScore, &run.AverageHits, &run.SampleCount, &run.SimulatorVersion, &run.CreatedAt); err != nil {
		return run, err
	}
	run.LineupRevision = revision.Int64
	if err := json.Unmarshal(ids, &run.BatterIDs); err != nil {
		return run, err
	}
	err := json.Unmarshal(batters, &run.Batters)
	return run, err
}

func (s *SQLStore) AddRun(ctx context.Context, user int64, lineup SavedLineup, result SimulationResult) (SimulationRun, error) {
	ids, err := json.Marshal(lineup.BatterIDs)
	if err != nil {
		return SimulationRun{}, err
	}
	batters, err := json.Marshal(lineup.Batters)
	if err != nil {
		return SimulationRun{}, err
	}
	// Lock any remaining link until insertion so a concurrent delete cannot race the FK check.
	// If deletion wins first, the evaluated snapshot is still stored with a null link.
	return scanRun(s.DB.QueryRowContext(ctx, `INSERT INTO simulation_runs
	(user_id,saved_lineup_id,lineup_name,team,year,lineup_revision,batter_ids,batters,average_score,average_hits,sample_count,simulator_version)
	VALUES($1,(SELECT id FROM saved_lineups WHERE id=$2 AND user_id=$1 FOR KEY SHARE),$3,$4,$5,NULLIF($6::bigint,0),$7,$8,$9,$10,$11,$12)
	RETURNING `+runColumns, user, lineup.ID, lineup.Name, lineup.Team, lineup.Year, lineup.Revision, string(ids), string(batters), result.AverageScore, result.AverageHits, result.SampleCount, result.SimulatorVersion))
}

func (s *SQLStore) ListRuns(ctx context.Context, user int64) ([]SimulationRun, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+runColumns+" FROM simulation_runs WHERE user_id=$1 ORDER BY created_at DESC,id DESC LIMIT 100", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := []SimulationRun{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}
