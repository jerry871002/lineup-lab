package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSimulatorUsesOnlySavedBatterFieldsAndServerMetadata(t *testing.T) {
	_, roster := fixture()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/simulate" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("invalid simulation request")
		}
		if r.Header.Get("Cookie") != "" || r.Header.Get("X-Authenticated-User-ID") != "" {
			t.Error("identity must not reach simulator")
		}
		var payload []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
			return
		}
		if len(payload) != 9 {
			t.Error("need nine hitters")
			return
		}
		for i, batter := range payload {
			if len(batter) != 8 || batter["id"] != nil || batter["team"] != nil || batter["name"] != roster[i].Name {
				t.Errorf("wrong payload fields: %v", batter)
			}
		}
		writeJSON(w, 200, SimulationResult{4.5, 9.25, 10, "lineup-engine-v1"})
	}))
	defer upstream.Close()
	result, err := simulateLineup(upstream.Client(), upstream.URL)(context.Background(), roster[:9])
	if err != nil || result != (SimulationResult{4.5, 9.25, 10, "lineup-engine-v1"}) {
		t.Fatalf("simulation metadata: %+v %v", result, err)
	}
}

func TestPostgresHistorySnapshotsAndOwnership(t *testing.T) {
	for _, upgrade := range []bool{false, true} {
		t.Run(fmt.Sprintf("upgrade=%v", upgrade), func(t *testing.T) {
			db := setupTestDatabase(t, upgrade)
			store := &SQLStore{DB: db}
			ctx := context.Background()
			input, roster := fixture()
			lineup, err := store.Save(ctx, 7, 0, input, roster[:9])
			if err != nil {
				t.Fatal(err)
			}
			result := SimulationResult{3.25, 8.5, 10, "lineup-engine-v1"}
			run, err := store.AddRun(ctx, 7, lineup, result)
			if err != nil {
				t.Fatal(err)
			}
			if run.SavedLineupID == nil || *run.SavedLineupID != lineup.ID || run.LineupName != lineup.Name || run.LineupRevision != 1 || run.CreatedAt.IsZero() || run.SimulationResult != result || !reflect.DeepEqual(run.Batters, lineup.Batters) || !reflect.DeepEqual(run.BatterIDs, lineup.BatterIDs) {
				t.Fatalf("snapshot roundtrip: %+v", run)
			}
			foreign, err := store.ListRuns(ctx, 8)
			if err != nil || len(foreign) != 0 {
				t.Fatalf("foreign history: %v %v", foreign, err)
			}
			input.Name = "Changed after evaluation"
			input.Revision = lineup.Revision
			roster[0].Hit = 25
			if _, err := store.Save(ctx, 7, lineup.ID, input, roster[:9]); err != nil {
				t.Fatal(err)
			}
			if err := store.Delete(ctx, 7, lineup.ID); err != nil {
				t.Fatal(err)
			}
			items, err := store.ListRuns(ctx, 7)
			if err != nil || len(items) != 1 {
				t.Fatalf("history after deletion: %v %v", items, err)
			}
			if items[0].SavedLineupID != nil || items[0].LineupName != lineup.Name || items[0].LineupRevision != 1 || items[0].Batters[0].Hit != 30 || items[0].SimulationResult != result {
				t.Fatalf("historical snapshot changed: %+v", items[0])
			}
			late, err := store.AddRun(ctx, 7, lineup, result)
			if err != nil || late.SavedLineupID != nil {
				t.Fatalf("deleted during computation: %+v %v", late, err)
			}
			for _, invalid := range []SimulationResult{{-1, 1, 10, "v1"}, {1, math.Inf(1), 10, "v1"}, {1, math.NaN(), 10, "v1"}, {1, 1, 0, "v1"}} {
				if _, err := store.AddRun(ctx, 7, lineup, invalid); err == nil {
					t.Fatalf("database accepted invalid metadata %+v", invalid)
				}
			}
		})
	}
}

func TestPostgresHistoryEndpointRejectsForgedScoresAndFailedSimulation(t *testing.T) {
	db := testDatabase(t)
	store := &SQLStore{DB: db}
	input, roster := fixture()
	lineup, err := store.Save(context.Background(), 7, 0, input, roster[:9])
	if err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/lineups/%d/simulations", lineup.ID)
	calls := 0
	server := &Server{Store: store, History: store, Simulate: func(ctx context.Context, batters []Batter) (SimulationResult, error) {
		calls++
		return SimulationResult{4, 9, 10, "lineup-engine-v1"}, nil
	}}
	handler := server.Handler()
	for _, body := range []string{`{"average_score":999}`, `{}`, `null`, `[]`, strings.Repeat(" ", 1025)} {
		if w := request(handler, "POST", path, "7", body); w.Code != 400 {
			t.Fatalf("client body accepted: %d %s", w.Code, w.Body)
		}
	}
	for _, user := range []string{"8", ""} {
		want := 404
		if user == "" {
			want = 401
		}
		if w := request(handler, "POST", path, user, nil); w.Code != want {
			t.Fatalf("foreign/anonymous simulation: %d", w.Code)
		}
	}
	if calls != 0 {
		t.Fatal("rejected requests must not run simulation")
	}
	for _, invalid := range []SimulationResult{{-1, 1, 10, "v1"}, {1, math.NaN(), 10, "v1"}, {1, math.Inf(1), 10, "v1"}, {1, 1, 0, "v1"}, {1, 1, 10, " "}} {
		server.Simulate = func(context.Context, []Batter) (SimulationResult, error) { return invalid, nil }
		if w := request(handler, "POST", path, "7", nil); w.Code != 502 {
			t.Fatalf("unsafe result recorded: %d %s", w.Code, w.Body)
		}
	}
	server.Simulate = func(context.Context, []Batter) (SimulationResult, error) {
		return SimulationResult{}, errors.New("upstream unavailable")
	}
	if w := request(handler, "POST", path, "7", nil); w.Code != 502 {
		t.Fatalf("failure status %d", w.Code)
	}
	items, err := store.ListRuns(context.Background(), 7)
	if err != nil || len(items) != 0 {
		t.Fatalf("failed computations created history: %v %v", items, err)
	}
	server.Simulate = func(ctx context.Context, batters []Batter) (SimulationResult, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) < 80*time.Second {
			t.Error("missing longer simulation request budget")
		}
		if !reflect.DeepEqual(batters, lineup.Batters) {
			t.Error("simulation did not use saved snapshot")
		}
		return SimulationResult{4, 9, 10, "lineup-engine-v1"}, nil
	}
	w := request(handler, "POST", path, "7", nil)
	if w.Code != 201 {
		t.Fatalf("record result: %d %s", w.Code, w.Body)
	}
	var run SimulationRun
	if err := json.Unmarshal(w.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	if run.AverageScore != 4 || run.LineupName != input.Name || run.SampleCount != 10 {
		t.Fatalf("stored result %+v", run)
	}
	for _, tc := range []struct {
		user  string
		count int
	}{{"7", 1}, {"8", 0}} {
		w := request(handler, "GET", "/simulations", tc.user, nil)
		var got []SimulationRun
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || len(got) != tc.count {
			t.Fatalf("history endpoint owner isolation: %d %s", w.Code, w.Body)
		}
	}
	if w := request(handler, "GET", "/simulations", "", nil); w.Code != 401 {
		t.Fatal("anonymous history accepted")
	}
	server.SimulationTimeout = 120 * time.Second
	server.Simulate = func(ctx context.Context, _ []Batter) (SimulationResult, error) {
		deadline, ok := ctx.Deadline()
		remaining := time.Until(deadline)
		if !ok || remaining <= 120*time.Second || remaining > 125*time.Second {
			t.Errorf("configured run budget was not applied: %s", remaining)
		}
		return SimulationResult{4, 9, 10, "lineup-engine-v1"}, nil
	}
	if w := request(handler, "POST", path, "7", nil); w.Code != 201 {
		t.Fatalf("configured run failed: %d", w.Code)
	}
}

func TestSimulatorRejectsUnusableResponses(t *testing.T) {
	_, roster := fixture()
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"upstream error", `{}`, 503},
		{"missing metadata", `{"average_score":1,"average_hits":2}`, 200},
		{"null averages", `{"average_score":null,"average_hits":2,"sample_count":10,"simulator_version":"v1"}`, 200},
		{"negative average", `{"average_score":-1,"average_hits":2,"sample_count":10,"simulator_version":"v1"}`, 200},
		{"nonfinite average", `{"average_score":1e309,"average_hits":2,"sample_count":10,"simulator_version":"v1"}`, 200},
		{"zero samples", `{"average_score":1,"average_hits":2,"sample_count":0,"simulator_version":"v1"}`, 200},
		{"empty version", `{"average_score":1,"average_hits":2,"sample_count":10,"simulator_version":" "}`, 200},
		{"malformed", `{`, 200},
		{"trailing values", `{} {}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer upstream.Close()
			if _, err := simulateLineup(upstream.Client(), upstream.URL)(context.Background(), roster[:9]); err == nil {
				t.Fatal("unsafe upstream response accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := simulateLineup(http.DefaultClient, "http://127.0.0.1:1")(ctx, roster[:9]); err == nil {
		t.Fatal("canceled simulation accepted")
	}
}

func TestPostgresHistoryReturnsOnlyLatest100PerOwner(t *testing.T) {
	db := testDatabase(t)
	store := &SQLStore{DB: db}
	ctx := context.Background()
	input, roster := fixture()
	lineup, err := store.Save(ctx, 7, 0, input, roster[:9])
	if err != nil {
		t.Fatal(err)
	}
	var newest SimulationRun
	for i := 0; i < 103; i++ {
		newest, err = store.AddRun(ctx, 7, lineup, SimulationResult{float64(i), 8, 10, "v1"})
		if err != nil {
			t.Fatal(err)
		}
	}
	foreignLineup, err := store.Save(ctx, 8, 0, input, roster[:9])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddRun(ctx, 8, foreignLineup, SimulationResult{999, 8, 10, "v1"}); err != nil {
		t.Fatal(err)
	}
	// Force timestamp ties to exercise the stable descending ID ordering.
	if _, err := db.Exec("UPDATE simulation_runs SET created_at='2024-01-01T00:00:00Z'"); err != nil {
		t.Fatal(err)
	}
	runs, err := store.ListRuns(ctx, 7)
	if err != nil || len(runs) != 100 || runs[0].ID != newest.ID || runs[99].AverageScore != 3 {
		t.Fatalf("latest100 ordering/limit: count=%d err=%v", len(runs), err)
	}
	for i, run := range runs {
		if run.AverageScore == 999 || (i > 0 && runs[i-1].ID <= run.ID) {
			t.Fatalf("history order/isolation broken at %+v", run)
		}
	}
}

type deadlineLineupStore struct {
	*memoryStore
	check func(context.Context)
}

func (s deadlineLineupStore) List(ctx context.Context, user int64) ([]SavedLineup, error) {
	s.check(ctx)
	return s.memoryStore.List(ctx, user)
}
func TestHistoryDoesNotExtendCRUDRequestBudget(t *testing.T) {
	store := deadlineLineupStore{memoryStore: &memoryStore{items: map[int64]SavedLineup{}, owners: map[int64]int64{}}, check: func(ctx context.Context) {
		deadline, ok := ctx.Deadline()
		remaining := time.Until(deadline)
		if !ok || remaining <= 0 || remaining > 10*time.Second {
			t.Errorf("CRUD budget changed: %v", remaining)
		}
	}}
	if w := request((&Server{Store: store}).Handler(), "GET", "/lineups", "7", nil); w.Code != 200 {
		t.Fatal("list failed")
	}
}

func TestPostgresRunSurvivesConcurrentLineupDeletion(t *testing.T) {
	db := testDatabase(t)
	store := &SQLStore{DB: db}
	input, roster := fixture()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lineup, err := store.Save(ctx, 7, 0, input, roster[:9])
	if err != nil {
		t.Fatal(err)
	}
	deleting, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer deleting.Rollback()
	var deletingPID int
	if err := deleting.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&deletingPID); err != nil {
		t.Fatal(err)
	}
	if _, err := deleting.ExecContext(ctx, "DELETE FROM saved_lineups WHERE id=$1 AND user_id=$2", lineup.ID, 7); err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		run SimulationRun
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		run, err := store.AddRun(ctx, 7, lineup, SimulationResult{4, 8, 10, "v1"})
		done <- outcome{run, err}
	}()
	// Wait until the INSERT is competing with this exact deletion transaction.
	deadline := time.Now().Add(2 * time.Second)
	blocked := false
	for time.Now().Before(deadline) {
		if err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE $1 = ANY(pg_blocking_pids(pid)))", deletingPID).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("run insert did not reach the concurrent deletion lock")
	}
	if err := deleting.Commit(); err != nil {
		t.Fatal(err)
	}
	result := <-done
	if result.err != nil || result.run.SavedLineupID != nil || result.run.LineupName != lineup.Name || result.run.LineupRevision != lineup.Revision {
		t.Fatalf("successful computation lost during concurrent deletion: %+v %v", result.run, result.err)
	}
	runs, err := store.ListRuns(ctx, 7)
	if err != nil || len(runs) != 1 || runs[0].SavedLineupID != nil {
		t.Fatalf("run not retained after racing delete: %+v %v", runs, err)
	}
}
