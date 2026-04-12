//go:build cgo

package store

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLStatStoreGetTeams(t *testing.T) {
	db := newTestDB(t)
	seedBattingTable(t, db)

	store := NewSQLStatStore(db)
	teams, err := store.GetTeams()
	if err != nil {
		t.Fatalf("GetTeams() error = %v", err)
	}

	if len(teams) != 2 {
		t.Fatalf("GetTeams() len = %d, want 2", len(teams))
	}

	gotTeams := map[string]int{}
	for _, team := range teams {
		gotTeams[team.Name] = team.Year
	}

	if gotTeams["Mets"] != 2023 {
		t.Fatalf("GetTeams() missing Mets 2023: %v", teams)
	}

	if gotTeams["Yankees"] != 2024 {
		t.Fatalf("GetTeams() missing Yankees 2024: %v", teams)
	}
}

func TestSQLStatStoreGetBattingStat(t *testing.T) {
	db := newTestDB(t)
	seedBattingTable(t, db)

	store := NewSQLStatStore(db)
	stats, err := store.GetBattingStat("Yankees", 2024)
	if err != nil {
		t.Fatalf("GetBattingStat() error = %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("GetBattingStat() len = %d, want 2", len(stats))
	}

	gotPlayers := map[string]bool{}
	for _, stat := range stats {
		name, ok := stat["name"].(string)
		if !ok {
			t.Fatalf("GetBattingStat() name type = %T, want string", stat["name"])
		}
		if team, ok := stat["team"].(string); !ok || team != "Yankees" {
			t.Fatalf("GetBattingStat() team = %v, want Yankees", stat["team"])
		}
		gotPlayers[name] = true
	}

	if !gotPlayers["Aaron Judge"] || !gotPlayers["Juan Soto"] {
		t.Fatalf("GetBattingStat() players = %v, want Aaron Judge and Juan Soto", gotPlayers)
	}
}

func TestRowsToMap(t *testing.T) {
	db := newTestDB(t)
	seedBattingTable(t, db)

	rows, err := db.Query("SELECT name, team, year FROM batting WHERE team = ? ORDER BY name", "Yankees")
	if err != nil {
		t.Fatalf("db.Query() error = %v", err)
	}
	defer rows.Close()

	result, err := rowsToMap(rows)
	if err != nil {
		t.Fatalf("rowsToMap() error = %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("rowsToMap() len = %d, want 2", len(result))
	}

	if got := result[0]["name"]; got != "Aaron Judge" {
		t.Fatalf("rowsToMap()[0][name] = %v, want Aaron Judge", got)
	}

	if got := result[0]["year"]; got != int64(2024) {
		t.Fatalf("rowsToMap()[0][year] = %v, want 2024", got)
	}
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func seedBattingTable(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		CREATE TABLE batting (
			name TEXT,
			team TEXT,
			year INTEGER,
			at_bat INTEGER,
			hit INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("CREATE TABLE error = %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO batting (name, team, year, at_bat, hit) VALUES
			('Aaron Judge', 'Yankees', 2024, 100, 30),
			('Juan Soto', 'Yankees', 2024, 100, 28),
			('Pete Alonso', 'Mets', 2023, 100, 24)
	`)
	if err != nil {
		t.Fatalf("INSERT batting error = %v", err)
	}
}
