package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Batter struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	AtBat      int    `json:"at_bat"`
	Hit        int    `json:"hit"`
	Double     int    `json:"double"`
	Triple     int    `json:"triple"`
	HomeRun    int    `json:"home_run"`
	BallOnBase int    `json:"ball_on_base"`
	HitByPitch int    `json:"hit_by_pitch"`
}

type LineupInput struct {
	Name      string  `json:"name"`
	Team      string  `json:"team"`
	Year      int     `json:"year"`
	BatterIDs []int64 `json:"batter_ids"`
	Revision  int64   `json:"revision,omitempty"`
}

type SavedLineup struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Team      string    `json:"team"`
	Year      int       `json:"year"`
	BatterIDs []int64   `json:"batter_ids"`
	Batters   []Batter  `json:"batters"`
	Revision  int64     `json:"revision"`
	UpdatedAt time.Time `json:"updated_at"`
}

var errConflict = errors.New("lineup changed; reload before saving")
var errNameConflict = errors.New("lineup name already exists for this team-season")

type LineupStore interface {
	List(context.Context, int64) ([]SavedLineup, error)
	Get(context.Context, int64, int64) (SavedLineup, error)
	Save(context.Context, int64, int64, LineupInput, []Batter) (SavedLineup, error)
	Delete(context.Context, int64, int64) error
}

type RosterLoader func(context.Context, string, int) ([]Batter, error)

type Server struct {
	SimulationTimeout time.Duration
	History           HistoryStore
	Simulate          Simulator
	Store             LineupStore
	Roster            RosterLoader
	Ready             func(context.Context) error
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if s.Ready != nil && s.Ready(ctx) != nil {
			apiError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /simulations", s.listSimulations)
	mux.HandleFunc("POST /simulations", s.runCurrentSimulation)
	mux.HandleFunc("POST /lineups/{id}/simulations", s.runSimulation)
	mux.HandleFunc("GET /lineups", s.list)
	mux.HandleFunc("POST /lineups", s.save)
	mux.HandleFunc("GET /lineups/{id}", s.get)
	mux.HandleFunc("PUT /lineups/{id}", s.save)
	mux.HandleFunc("DELETE /lineups/{id}", s.remove)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" && r.URL.Path != "/readyz" {
			value := r.Header.Get("X-Authenticated-User-ID")
			id, err := strconv.ParseInt(value, 10, 64)
			if err != nil || id <= 0 || strconv.FormatInt(id, 10) != value {
				apiError(w, http.StatusUnauthorized, "authentication required")
				return
			}
		}
		budget := crudRequestTimeout
		if r.Method == http.MethodPost && (r.URL.Path == "/simulations" || (strings.HasPrefix(r.URL.Path, "/lineups/") && strings.HasSuffix(r.URL.Path, "/simulations"))) {
			budget = s.simulationBudget()
		}
		ctx, cancel := context.WithTimeout(r.Context(), budget)
		defer cancel()
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.Header.Get("X-Authenticated-User-ID"), 10, 64)
	return id
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusNotFound, "lineup not found")
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}

func apiError(w http.ResponseWriter, code int, detail string) {
	writeJSON(w, code, map[string]string{"detail": detail})
}

func storeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		apiError(w, http.StatusNotFound, "lineup not found")
	case errors.Is(err, errConflict):
		apiError(w, http.StatusConflict, errConflict.Error())
	case errors.Is(err, errNameConflict):
		apiError(w, http.StatusConflict, errNameConflict.Error())
	default:
		log.Printf("lineups store: %v", err)
		apiError(w, http.StatusServiceUnavailable, "lineups unavailable")
	}
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.List(r.Context(), userID(r))
	if err != nil {
		storeError(w, err)
		return
	}
	if items == nil {
		items = []SavedLineup{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	item, err := s.Store.Get(r.Context(), userID(r), id)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) save(w http.ResponseWriter, r *http.Request) {
	id := int64(0)
	if r.Method == http.MethodPut {
		var ok bool
		id, ok = pathID(w, r)
		if !ok {
			return
		}
		if _, err := s.Store.Get(r.Context(), userID(r), id); err != nil {
			storeError(w, err)
			return
		}
	}
	var input LineupInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		apiError(w, http.StatusBadRequest, "invalid lineup request")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		apiError(w, http.StatusBadRequest, "request must contain one JSON value")
		return
	}
	input.Name, input.Team = strings.TrimSpace(input.Name), strings.TrimSpace(input.Team)
	if input.Name == "" || len([]rune(input.Name)) > 100 || input.Team == "" || len([]rune(input.Team)) > 100 || input.Year < 1900 || input.Year > 2100 || len(input.BatterIDs) != 9 || (id != 0 && input.Revision < 1) {
		apiError(w, http.StatusBadRequest, "name, team, year and nine unique batter IDs are required; updates require revision")
		return
	}
	batters, ok := s.resolveBatters(w, r, input.Team, input.Year, input.BatterIDs)
	if !ok {
		return
	}
	item, err := s.Store.Save(r.Context(), userID(r), id, input, batters)
	if err != nil {
		storeError(w, err)
		return
	}
	code := http.StatusOK
	if id == 0 {
		code = http.StatusCreated
		w.Header().Set("Location", "/api/lineups/"+strconv.FormatInt(item.ID, 10))
	}
	writeJSON(w, code, item)
}

func (s *Server) resolveBatters(w http.ResponseWriter, r *http.Request, team string, year int, batterIDs []int64) ([]Batter, bool) {
	if team == "" || len([]rune(team)) > 100 || year < 1900 || year > 2100 || len(batterIDs) != 9 {
		apiError(w, http.StatusBadRequest, "team, year and nine unique batter IDs are required")
		return nil, false
	}
	seen := make(map[int64]bool, 9)
	for _, batterID := range batterIDs {
		if batterID <= 0 || seen[batterID] {
			apiError(w, http.StatusBadRequest, "nine unique positive batter IDs are required")
			return nil, false
		}
		seen[batterID] = true
	}
	roster, err := s.Roster(r.Context(), team, year)
	if err != nil {
		log.Printf("load roster: %v", err)
		apiError(w, http.StatusServiceUnavailable, "roster unavailable")
		return nil, false
	}
	byID := make(map[int64]Batter, len(roster))
	for _, batter := range roster {
		byID[batter.ID] = batter
	}
	batters := make([]Batter, 9)
	names := make(map[string]bool, 9)
	canRecordOut := false
	for i, batterID := range batterIDs {
		batter, exists := byID[batterID]
		name := strings.ToLower(strings.TrimSpace(batter.Name))
		if !exists || name == "" || names[name] || batter.AtBat <= 0 || batter.Hit < 0 || batter.Hit > batter.AtBat || batter.Double < 0 || batter.Triple < 0 || batter.HomeRun < 0 || batter.BallOnBase < 0 || batter.HitByPitch < 0 || batter.Double+batter.Triple+batter.HomeRun > batter.Hit {
			apiError(w, http.StatusBadRequest, "batters must be valid members of the selected team-season")
			return nil, false
		}
		names[name] = true
		canRecordOut = canRecordOut || batter.AtBat > batter.Hit
		batters[i] = batter
	}
	if !canRecordOut {
		apiError(w, http.StatusBadRequest, "lineup must be able to record an out")
		return nil, false
	}
	return batters, true
}

func (s *Server) remove(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.Store.Delete(r.Context(), userID(r), id); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func loadRoster(client *http.Client, base string) RosterLoader {
	return func(ctx context.Context, team string, year int) ([]Batter, error) {
		endpoint := strings.TrimRight(base, "/") + "/batting?" + url.Values{"team": {team}, "year": {strconv.Itoa(year)}}.Encode()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		response, err := client.Do(request)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, errors.New("stats service rejected roster request")
		}
		var batters []Batter
		err = json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&batters)
		return batters, err
	}
}

func main() {
	databaseURL, statsURL := os.Getenv("DATABASE_URL"), os.Getenv("STATS_BASE_URL")
	simulationURL := os.Getenv("SIMULATION_BASE_URL")
	if databaseURL == "" || statsURL == "" || simulationURL == "" {
		log.Fatal("DATABASE_URL, STATS_BASE_URL and SIMULATION_BASE_URL must be set")
	}
	simulationTimeout, err := simulationTimeoutFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	store := &SQLStore{DB: db}
	server := &Server{
		Store:             store,
		History:           store,
		Simulate:          simulateLineup(&http.Client{Timeout: simulationTimeout}, simulationURL),
		SimulationTimeout: simulationTimeout,
		Roster:            loadRoster(&http.Client{Timeout: 8 * time.Second}, statsURL),
		Ready:             db.PingContext,
	}
	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      server.simulationBudget() + responseWriteGrace,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(httpServer.ListenAndServe())
}
