package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type memoryStore struct {
	items    map[int64]SavedLineup
	owners   map[int64]int64
	lastUser int64
}

func (m *memoryStore) List(ctx context.Context, user int64) ([]SavedLineup, error) {
	m.lastUser = user
	items := []SavedLineup{}
	for id, item := range m.items {
		if m.owners[id] == user {
			items = append(items, item)
		}
	}
	return items, nil
}
func (m *memoryStore) Get(ctx context.Context, user, id int64) (SavedLineup, error) {
	m.lastUser = user
	if item, ok := m.items[id]; ok && m.owners[id] == user {
		return item, nil
	}
	return SavedLineup{}, sql.ErrNoRows
}
func (m *memoryStore) Save(ctx context.Context, user, id int64, input LineupInput, batters []Batter) (SavedLineup, error) {
	m.lastUser = user
	if id == 0 {
		id = int64(len(m.items) + 1)
	} else {
		previous, err := m.Get(ctx, user, id)
		if err != nil {
			return SavedLineup{}, err
		}
		if previous.Revision != input.Revision {
			return SavedLineup{}, errConflict
		}
	}
	item := SavedLineup{ID: id, Name: input.Name, Team: input.Team, Year: input.Year, BatterIDs: input.BatterIDs, Batters: batters, Revision: input.Revision + 1}
	m.items[id], m.owners[id] = item, user
	return item, nil
}
func (m *memoryStore) Delete(ctx context.Context, user, id int64) error {
	if _, err := m.Get(ctx, user, id); err != nil {
		return err
	}
	delete(m.items, id)
	delete(m.owners, id)
	return nil
}

func fixture() (LineupInput, []Batter) {
	input := LineupInput{Name: "My lineup", Team: "Demo & Friends", Year: 2024}
	batters := make([]Batter, 10)
	for i := range batters {
		batters[i] = Batter{ID: int64(i + 1), Name: fmt.Sprintf("Batter %d", i+1), AtBat: 100, Hit: 30, Double: 5, Triple: 1, HomeRun: 4, BallOnBase: 10, HitByPitch: 2}
	}
	for i := 9; i > 0; i-- {
		input.BatterIDs = append(input.BatterIDs, int64(i))
	}
	return input, batters
}

func request(handler http.Handler, method, path, user string, body any) *httptest.ResponseRecorder {
	var payload string
	if raw, ok := body.(string); ok {
		payload = raw
	} else if body != nil {
		encoded, _ := json.Marshal(body)
		payload = string(encoded)
	}
	r := httptest.NewRequest(method, path, strings.NewReader(payload))
	if user != "" {
		r.Header.Set("X-Authenticated-User-ID", user)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestLineupOwnershipAndRoundTrip(t *testing.T) {
	input, roster := fixture()
	store := &memoryStore{items: map[int64]SavedLineup{}, owners: map[int64]int64{}}
	server := &Server{Store: store, Roster: func(ctx context.Context, team string, year int) ([]Batter, error) {
		if team != input.Team || year != input.Year {
			t.Fatalf("wrong team-season %s/%d", team, year)
		}
		return roster, nil
	}}
	handler := server.Handler()
	created := request(handler, "POST", "/lineups", "7", input)
	if created.Code != 201 {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	var item SavedLineup
	if err := json.Unmarshal(created.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.ID != 1 || item.Revision != 1 || !reflect.DeepEqual(item.BatterIDs, input.BatterIDs) || item.Batters[0].ID != 9 || store.lastUser != 7 {
		t.Fatalf("bad saved order or owner: %+v", item)
	}
	for _, user := range []string{"7", "8"} {
		list := request(handler, "GET", "/lineups", user, nil)
		var items []SavedLineup
		if err := json.Unmarshal(list.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		if (user == "7" && len(items) != 1) || (user == "8" && len(items) != 0) {
			t.Fatalf("list leaked/missing items: %s", list.Body)
		}
	}
	for _, method := range []string{"GET", "PUT", "DELETE"} {
		foreign := request(handler, method, "/lineups/1", "8", input)
		missing := request(handler, method, "/lineups/999", "8", input)
		if foreign.Code != 404 || missing.Code != 404 || foreign.Body.String() != missing.Body.String() {
			t.Fatalf("cross-user %s: %d %s vs %d %s", method, foreign.Code, foreign.Body, missing.Code, missing.Body)
		}
	}
	input.Revision = 1
	input.Name = "Changed"
	if w := request(handler, "PUT", "/lineups/1", "7", input); w.Code != 200 {
		t.Fatalf("update: %d %s", w.Code, w.Body)
	}
	if w := request(handler, "PUT", "/lineups/1", "7", input); w.Code != 409 {
		t.Fatalf("stale revision: %d", w.Code)
	}
	if w := request(handler, "DELETE", "/lineups/1", "7", nil); w.Code != 204 {
		t.Fatalf("delete: %d", w.Code)
	}
	if w := request(handler, "GET", "/lineups/1", "7", nil); w.Code != 404 {
		t.Fatalf("deleted item: %d", w.Code)
	}
}

func TestRejectsInvalidLineupBeforeSave(t *testing.T) {
	valid, roster := fixture()
	encoded, _ := json.Marshal(valid)
	cases := []struct{ name, body string }{
		{"malformed", "{"},
		{"owner spoofing", strings.TrimSuffix(string(encoded), "}") + `,"user_id":99}`},
		{"extra JSON", string(encoded) + "{}"},
		{"empty object", "{}"},
		{"missing member", strings.Replace(string(encoded), "[9,8,7,6,5,4,3,2,1]", "[99,8,7,6,5,4,3,2,1]", 1)},
		{"duplicate member", strings.Replace(string(encoded), "[9,8,7,6,5,4,3,2,1]", "[8,8,7,6,5,4,3,2,1]", 1)},
		{"too few", strings.Replace(string(encoded), "[9,8,7,6,5,4,3,2,1]", "[8,7,6,5,4,3,2,1]", 1)},
		{"oversized", strings.Repeat(" ", 16385) + string(encoded)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &memoryStore{items: map[int64]SavedLineup{}, owners: map[int64]int64{}}
			handler := (&Server{Store: store, Roster: func(context.Context, string, int) ([]Batter, error) { return roster, nil }}).Handler()
			w := request(handler, "POST", "/lineups", "7", tc.body)
			if w.Code != 400 || len(store.items) != 0 {
				t.Fatalf("invalid payload saved: %d %s", w.Code, w.Body)
			}
		})
	}
}

func TestUntrustedIdentityAndUpstreamFailures(t *testing.T) {
	input, _ := fixture()
	store := &memoryStore{items: map[int64]SavedLineup{}, owners: map[int64]int64{}}
	handler := (&Server{Store: store, Roster: func(context.Context, string, int) ([]Batter, error) { return nil, errors.New("upstream down") }, Ready: func(context.Context) error { return errors.New("db down") }}).Handler()
	for _, user := range []string{"", "0", "-1", "abc", "01", "7,8"} {
		if w := request(handler, "GET", "/lineups", user, nil); w.Code != 401 {
			t.Fatalf("bad identity %q accepted", user)
		}
	}
	if w := request(handler, "POST", "/lineups", "7", input); w.Code != 503 {
		t.Fatalf("upstream error: %d", w.Code)
	}
	if w := request(handler, "GET", "/healthz", "", nil); w.Code != 200 {
		t.Fatal("liveness failed")
	}
	if w := request(handler, "GET", "/readyz", "", nil); w.Code != 503 {
		t.Fatal("readiness ignores DB failure")
	}
}

func TestRosterLoaderEncodesTeamSeason(t *testing.T) {
	input, roster := fixture()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/batting" || r.URL.Query().Get("team") != input.Team || r.URL.Query().Get("year") != "2024" {
			t.Errorf("bad upstream query %s", r.URL)
		}
		if r.Header.Get("Cookie") != "" {
			t.Error("cookies forwarded to stats")
		}
		writeJSON(w, 200, roster)
	}))
	defer upstream.Close()
	got, err := loadRoster(upstream.Client(), upstream.URL)(context.Background(), input.Team, input.Year)
	if err != nil || !reflect.DeepEqual(got, roster) {
		t.Fatalf("roster: %v %v", got, err)
	}
}

func TestRejectsInvalidCanonicalRoster(t *testing.T) {
	for _, scenario := range []string{"invalid stats", "duplicate names", "no outs"} {
		t.Run(scenario, func(t *testing.T) {
			input, roster := fixture()
			switch scenario {
			case "invalid stats":
				roster[0].Hit = 101
			case "duplicate names":
				roster[0].Name = roster[1].Name
			case "no outs":
				for i := range roster {
					roster[i].Hit = roster[i].AtBat
				}
			}
			store := &memoryStore{items: map[int64]SavedLineup{}, owners: map[int64]int64{}}
			handler := (&Server{Store: store, Roster: func(context.Context, string, int) ([]Batter, error) { return roster, nil }}).Handler()
			w := request(handler, "POST", "/lineups", "7", input)
			if w.Code != 400 || len(store.items) != 0 {
				t.Fatalf("unsafe roster saved: %d %s", w.Code, w.Body)
			}
		})
	}
}
