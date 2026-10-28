package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/lib/pq"
)

type SQLStore struct{ DB *sql.DB }

const columns = "id, name, team, year, batter_ids, batters, revision, updated_at"

type scanner interface{ Scan(...any) error }

func scanLineup(row scanner) (SavedLineup, error) {
	var item SavedLineup
	var ids, batters []byte
	if err := row.Scan(&item.ID, &item.Name, &item.Team, &item.Year, &ids, &batters, &item.Revision, &item.UpdatedAt); err != nil {
		return item, err
	}
	if err := json.Unmarshal(ids, &item.BatterIDs); err != nil {
		return item, err
	}
	err := json.Unmarshal(batters, &item.Batters)
	return item, err
}

func (s *SQLStore) List(ctx context.Context, user int64) ([]SavedLineup, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+columns+" FROM saved_lineups WHERE user_id=$1 ORDER BY updated_at DESC, id DESC LIMIT 100", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SavedLineup{}
	for rows.Next() {
		item, err := scanLineup(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLStore) Get(ctx context.Context, user, id int64) (SavedLineup, error) {
	return scanLineup(s.DB.QueryRowContext(ctx, "SELECT "+columns+" FROM saved_lineups WHERE id=$1 AND user_id=$2", id, user))
}

func (s *SQLStore) Save(ctx context.Context, user, id int64, input LineupInput, batters []Batter) (SavedLineup, error) {
	idsJSON, err := json.Marshal(input.BatterIDs)
	if err != nil {
		return SavedLineup{}, err
	}
	battersJSON, err := json.Marshal(batters)
	if err != nil {
		return SavedLineup{}, err
	}
	var row *sql.Row
	if id == 0 {
		row = s.DB.QueryRowContext(ctx, "INSERT INTO saved_lineups (user_id,name,team,year,batter_ids,batters) VALUES($1,$2,$3,$4,$5,$6) RETURNING "+columns, user, input.Name, input.Team, input.Year, string(idsJSON), string(battersJSON))
	} else {
		row = s.DB.QueryRowContext(ctx, "UPDATE saved_lineups SET name=$3,team=$4,year=$5,batter_ids=$6,batters=$7,revision=revision+1,updated_at=NOW() WHERE id=$1 AND user_id=$2 AND revision=$8 RETURNING "+columns, id, user, input.Name, input.Team, input.Year, string(idsJSON), string(battersJSON), input.Revision)
	}
	item, err := scanLineup(row)
	var pqError *pq.Error
	if errors.As(err, &pqError) && pqError.Code == "23505" {
		return item, errNameConflict
	}
	if id != 0 && errors.Is(err, sql.ErrNoRows) {
		if _, getErr := s.Get(ctx, user, id); getErr != nil {
			return item, getErr
		}
		return item, errConflict
	}
	return item, err
}

func (s *SQLStore) Delete(ctx context.Context, user, id int64) error {
	result, err := s.DB.ExecContext(ctx, "DELETE FROM saved_lineups WHERE id=$1 AND user_id=$2", id, user)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}
