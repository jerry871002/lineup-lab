package store

import (
	"database/sql"

	"github.com/jerry871002/lineup-lab/stat-api-server/internal/api"
)

type SQLStatStore struct {
	db *sql.DB
}

func NewSQLStatStore(db *sql.DB) *SQLStatStore {
	return &SQLStatStore{db: db}
}

func (s *SQLStatStore) GetTeams() ([]api.Team, error) {
	rows, err := s.db.Query("SELECT DISTINCT team, year FROM batting")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []api.Team{}
	for rows.Next() {
		var team api.Team
		if err := rows.Scan(&team.Name, &team.Year); err != nil {
			return nil, err
		}
		data = append(data, team)
	}

	return data, rows.Err()
}

func (s *SQLStatStore) GetBattingStat(team string, year int) ([]map[string]any, error) {
	rows, err := s.db.Query("SELECT * FROM batting WHERE team = $1 AND year = $2", team, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToMap(rows)
}

func rowsToMap(rows *sql.Rows) ([]map[string]any, error) {
	var data []map[string]any
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		columns := make([]any, len(cols))
		columnPointers := make([]any, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}
		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		entry := make(map[string]any, len(cols))
		for i, colName := range cols {
			value := columnPointers[i].(*any)
			entry[colName] = *value
		}
		data = append(data, entry)
	}

	return data, rows.Err()
}
