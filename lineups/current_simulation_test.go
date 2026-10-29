package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func currentBody(input LineupInput) map[string]any {
	return map[string]any{"team": input.Team, "year": input.Year, "batter_ids": input.BatterIDs}
}

func TestPostgresCurrentSimulationRecordsExperimentsWithoutSavedLineups(t *testing.T) {
	for _, upgrade := range []bool{false, true} {
		t.Run(fmt.Sprintf("upgrade=%v", upgrade), func(t *testing.T) {
			db := setupTestDatabase(t, upgrade)
			store := &SQLStore{DB: db}
			input, roster := fixture()
			server := &Server{Store: store, History: store, Roster: func(_ context.Context, team string, year int) ([]Batter, error) {
				if team != input.Team || year != input.Year {
					t.Error("wrong canonical roster requested")
				}
				return roster, nil
			}}
			calls := 0
			server.Simulate = func(ctx context.Context, batters []Batter) (SimulationResult, error) {
				calls++
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) < 80*time.Second {
					t.Error("quick simulation must receive the run budget")
				}
				for i, batter := range batters {
					if batter.ID != input.BatterIDs[i] {
						t.Error("simulation order differs from requested IDs")
					}
				}
				return SimulationResult{4, 8, 10, "lineup-engine-v1"}, nil
			}
			handler := server.Handler()
			var runs []SimulationRun
			for attempt := 0; attempt < 2; attempt++ {
				w := request(handler, "POST", "/simulations", "7", currentBody(input))
				if w.Code != 201 {
					t.Fatalf("current run: %d %s", w.Code, w.Body)
				}
				var run SimulationRun
				if err := json.Unmarshal(w.Body.Bytes(), &run); err != nil {
					t.Fatal(err)
				}
				var wire map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &wire); err != nil {
					t.Fatal(err)
				}
				if _, exists := wire["lineup_revision"]; exists {
					t.Fatal("current run must not fabricate a saved revision")
				}
				if run.SavedLineupID != nil || run.LineupName != "Current lineup" || run.LineupRevision != 0 || !reflect.DeepEqual(run.BatterIDs, input.BatterIDs) || len(run.Batters) != 9 {
					t.Fatalf("current snapshot: %+v", run)
				}
				for i, batter := range run.Batters {
					if batter.ID != input.BatterIDs[i] {
						t.Fatal("recorded order changed")
					}
				}
				var revision, link sql.NullInt64
				if err := db.QueryRow("SELECT lineup_revision,saved_lineup_id FROM simulation_runs WHERE id=$1", run.ID).Scan(&revision, &link); err != nil {
					t.Fatal(err)
				}
				if revision.Valid || link.Valid {
					t.Fatal("current run must store NULL revision and saved link")
				}
				var savedCount int
				if err := db.QueryRow("SELECT count(*) FROM saved_lineups").Scan(&savedCount); err != nil || savedCount != 0 {
					t.Fatalf("experiment created saved lineups: %d %v", savedCount, err)
				}
				runs = append(runs, run)
				for i, j := 0, len(input.BatterIDs)-1; i < j; i, j = i+1, j-1 {
					input.BatterIDs[i], input.BatterIDs[j] = input.BatterIDs[j], input.BatterIDs[i]
				}
				roster[0].Hit = 25
			}
			history, err := store.ListRuns(context.Background(), 7)
			if err != nil || len(history) != 2 || history[0].ID != runs[1].ID || !reflect.DeepEqual(history[1].Batters, runs[0].Batters) {
				t.Fatalf("history snapshot/order: %+v %v", history, err)
			}
			foreign, err := store.ListRuns(context.Background(), 8)
			if err != nil || len(foreign) != 0 {
				t.Fatalf("history leaked: %+v %v", foreign, err)
			}
			if calls != 2 {
				t.Fatalf("compute calls=%d", calls)
			}
			// Existing saved-lineup runs still expose their real revision.
			saved, err := store.Save(context.Background(), 7, 0, input, roster[:9])
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec("UPDATE saved_lineups SET revision=2147483648 WHERE id=$1", saved.ID); err != nil {
				t.Fatal(err)
			}
			server.Simulate = func(context.Context, []Batter) (SimulationResult, error) {
				return SimulationResult{4, 8, 10, "lineup-engine-v1"}, nil
			}
			w := request(handler, "POST", fmt.Sprintf("/lineups/%d/simulations", saved.ID), "7", nil)
			var savedRun SimulationRun
			if err := json.Unmarshal(w.Body.Bytes(), &savedRun); err != nil {
				t.Fatal(err)
			}
			if w.Code != 201 || savedRun.SavedLineupID == nil || *savedRun.SavedLineupID != saved.ID || savedRun.LineupRevision != 2147483648 {
				t.Fatalf("saved revision compatibility: %d %+v", w.Code, savedRun)
			}
		})
	}
}

func TestCurrentSimulationRejectsUnsafeInputBeforeComputation(t *testing.T) {
	input, roster := fixture()
	encoded, _ := json.Marshal(currentBody(input))
	encodedTeam, _ := json.Marshal(input.Team)
	valid := string(encoded)
	cases := []struct{ name, body string }{
		{"malformed", "{"}, {"null", "null"}, {"empty", "{}"},
		{"score", strings.TrimSuffix(valid, "}") + `,"average_score":999}`},
		{"owner", strings.TrimSuffix(valid, "}") + `,"user_id":8}`},
		{"name", strings.TrimSuffix(valid, "}") + `,"name":"Fake saved lineup"}`},
		{"revision", strings.TrimSuffix(valid, "}") + `,"revision":12}`},
		{"extra field", strings.TrimSuffix(valid, "}") + `,"extra":true}`},
		{"trailing JSON", valid + `{}`}, {"trailing garbage", valid + `!`},
		{"oversized", strings.Repeat(" ", 16385) + valid},
		{"invalid year", strings.Replace(valid, `"year":2024`, `"year":1899`, 1)},
		{"empty team", strings.Replace(valid, `"team":`+string(encodedTeam), `"team":" "`, 1)},
		{"short lineup", strings.Replace(valid, "[9,8,7,6,5,4,3,2,1]", "[8,7,6,5,4,3,2,1]", 1)},
		{"duplicate ID", strings.Replace(valid, "[9,8,7,6,5,4,3,2,1]", "[8,8,7,6,5,4,3,2,1]", 1)},
		{"unknown ID", strings.Replace(valid, "[9,8,7,6,5,4,3,2,1]", "[99,8,7,6,5,4,3,2,1]", 1)},
		{"negative ID", strings.Replace(valid, "[9,8,7,6,5,4,3,2,1]", "[-1,8,7,6,5,4,3,2,1]", 1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := &Server{Roster: func(context.Context, string, int) ([]Batter, error) { return roster, nil }, Simulate: func(context.Context, []Batter) (SimulationResult, error) {
				t.Fatal("unsafe input reached compute")
				return SimulationResult{}, nil
			}}
			// There is intentionally no saved-lineup store: quick runs must not use one.
			w := request(server.Handler(), "POST", "/simulations", "7", tc.body)
			if w.Code != 400 {
				t.Fatalf("unsafe input accepted: %d %s", w.Code, w.Body)
			}
		})
	}
	for _, scenario := range []string{"invalid stats", "duplicate names", "no outs", "unavailable roster"} {
		t.Run(scenario, func(t *testing.T) {
			_, canonical := fixture()
			want := 400
			switch scenario {
			case "invalid stats":
				canonical[0].Hit = 101
			case "duplicate names":
				canonical[0].Name = canonical[1].Name
			case "no outs":
				for i := range canonical {
					canonical[i].Hit = canonical[i].AtBat
				}
			case "unavailable roster":
				want = 503
			}
			server := &Server{Roster: func(context.Context, string, int) ([]Batter, error) {
				if scenario == "unavailable roster" {
					return nil, errors.New("stats unavailable")
				}
				return canonical, nil
			}, Simulate: func(context.Context, []Batter) (SimulationResult, error) {
				t.Fatal("unsafe canonical roster reached compute")
				return SimulationResult{}, nil
			}}
			if w := request(server.Handler(), "POST", "/simulations", "7", valid); w.Code != want {
				t.Fatalf("unsafe roster accepted: %d %s", w.Code, w.Body)
			}
		})
	}
	if w := request((&Server{}).Handler(), "POST", "/simulations", "", valid); w.Code != http.StatusUnauthorized {
		t.Fatal("anonymous quick run accepted")
	}
}

func TestPostgresCurrentSimulationFailureCreatesNoRows(t *testing.T) {
	db := testDatabase(t)
	store := &SQLStore{DB: db}
	input, roster := fixture()
	server := &Server{History: store, Roster: func(context.Context, string, int) ([]Batter, error) { return roster, nil }, Simulate: func(context.Context, []Batter) (SimulationResult, error) {
		return SimulationResult{}, errors.New("simulation unavailable")
	}}
	if w := request(server.Handler(), "POST", "/simulations", "7", currentBody(input)); w.Code != 502 {
		t.Fatalf("failed quick run: %d %s", w.Code, w.Body)
	}
	for _, table := range []string{"saved_lineups", "simulation_runs"} {
		var count int
		if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("failed compute wrote %s: %d %v", table, count, err)
		}
	}
}
