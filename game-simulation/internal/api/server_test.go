package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/jerry871002/lineup-lab/game-simulation/internal/simulation"
)

func TestSimulateHandlerAcceptsValidPayload(t *testing.T) {
	handler := NewHandler(true, "http://localhost:3000")
	request := httptest.NewRequest(http.MethodPost, "/simulate", strings.NewReader(validLineupJSON))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("simulateHandler status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHealthAndReadinessHandlers(t *testing.T) {
	handler := NewHandler(true, "http://localhost:3000")

	testCases := []struct {
		name   string
		path   string
		method string
		want   int
	}{
		{name: "health ok", path: "/healthz", method: http.MethodGet, want: http.StatusOK},
		{name: "ready ok", path: "/readyz", method: http.MethodGet, want: http.StatusOK},
		{name: "health slash not found", path: "/healthz/", method: http.MethodGet, want: http.StatusNotFound},
		{name: "ready slash not found", path: "/readyz/", method: http.MethodGet, want: http.StatusNotFound},
		{name: "health rejects post", path: "/healthz", method: http.MethodPost, want: http.StatusMethodNotAllowed},
		{name: "ready rejects post", path: "/readyz", method: http.MethodPost, want: http.StatusMethodNotAllowed},
		{name: "simulate slash not found", path: "/simulate/", method: http.MethodPost, want: http.StatusNotFound},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, testCase.path, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != testCase.want {
				t.Fatalf("%s status = %d, want %d", testCase.path, response.Code, testCase.want)
			}
		})
	}
}

func TestSimulateHandlerRejectsInvalidPayloads(t *testing.T) {
	testCases := []struct {
		name           string
		body           string
		wantStatusCode int
		wantBody       string
	}{
		{
			name:           "malformed json",
			body:           `[{"name":"Mike Trout"`,
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "invalid request body",
		},
		{
			name:           "unknown field",
			body:           strings.Replace(validLineupJSON, `"hit_by_pitch":2`, `"hit_by_pitch":2,"unexpected":true`, 1),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "unknown field",
		},
		{
			name:           "trailing json",
			body:           validLineupJSON + `{"extra":true}`,
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "single JSON value",
		},
		{
			name:           "wrong lineup size",
			body:           `[` + strings.Join(validBatters[:8], ",") + `]`,
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "lineup must have 9 batters",
		},
		{
			name:           "empty name",
			body:           strings.Replace(validLineupJSON, `"Mike Trout"`, `"   "`, 1),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "batter 0 name must not be empty",
		},
		{
			name:           "negative stat",
			body:           strings.Replace(validLineupJSON, `"at_bat":100`, `"at_bat":-1`, 1),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "batter 0 at_bat must be non-negative",
		},
		{
			name:           "zero at bat",
			body:           strings.Replace(validLineupJSON, `"at_bat":100`, `"at_bat":0`, 1),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "batter 0 at_bat must be greater than 0",
		},
		{
			name:           "hits exceed at bats",
			body:           strings.Replace(validLineupJSON, `"hit":30`, `"hit":101`, 1),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "batter 0 hit must not exceed at_bat",
		},
		{
			name:           "extra base hits exceed hits",
			body:           strings.Replace(validLineupJSON, `"home_run":4`, `"home_run":40`, 1),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "batter 0 doubles, triples, and home runs must not exceed hits",
		},
		{
			name:           "duplicate names",
			body:           strings.Replace(validLineupJSON, `"Shohei Ohtani"`, `"Mike Trout"`, 1),
			wantStatusCode: http.StatusBadRequest,
			wantBody:       `duplicates batter 0 name "Mike Trout"`,
		},
		{
			name:           "lineup cannot record an out",
			body:           impossibleOutLineupJSON,
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "lineup must contain at least one batter with non-zero out probability",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := NewHandler(true, "http://localhost:3000")
			request := httptest.NewRequest(http.MethodPost, "/simulate", strings.NewReader(testCase.body))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != testCase.wantStatusCode {
				t.Fatalf("simulateHandler status = %d, want %d; body = %s", response.Code, testCase.wantStatusCode, response.Body.String())
			}

			if !strings.Contains(response.Body.String(), testCase.wantBody) {
				t.Fatalf("simulateHandler body = %q, want substring %q", response.Body.String(), testCase.wantBody)
			}
		})
	}
}

func TestOptimizeHandler(t *testing.T) {
	originalOptimizer := newOptimizer
	t.Cleanup(func() {
		newOptimizer = originalOptimizer
	})

	handler := NewHandler(true, "http://localhost:3000")

	t.Run("rejects invalid method", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/optimize", nil)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("optimizeHandler status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("rejects invalid body", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/optimize", strings.NewReader(`{`))
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("optimizeHandler status = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("returns optimized lineup", func(t *testing.T) {
		expectedLineup := []simulation.Batter{
			{Name: "Lead Off", AtBat: 10, Hit: 3},
			{Name: "Slugger", AtBat: 10, Hit: 4},
		}
		newOptimizer = func() simulation.Optimizer {
			return stubOptimizer{lineup: expectedLineup}
		}

		request := httptest.NewRequest(http.MethodPost, "/optimize", strings.NewReader(validLineupJSON))
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("optimizeHandler status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
		}

		var lineup []simulation.Batter
		if err := json.NewDecoder(response.Body).Decode(&lineup); err != nil {
			t.Fatalf("json.Decode() error = %v", err)
		}

		if !reflect.DeepEqual(lineup, expectedLineup) {
			t.Fatalf("optimized lineup = %+v, want %+v", lineup, expectedLineup)
		}
	})
}

type stubOptimizer struct {
	lineup simulation.Lineup
}

func (s stubOptimizer) Optimize(simulation.Roster) simulation.Lineup {
	return s.lineup
}

var validBatters = []string{
	`{"name":"Mike Trout","at_bat":100,"hit":30,"double":5,"triple":1,"home_run":4,"ball_on_base":10,"hit_by_pitch":2}`,
	`{"name":"Mookie Betts","at_bat":100,"hit":25,"double":6,"triple":0,"home_run":3,"ball_on_base":12,"hit_by_pitch":1}`,
	`{"name":"Aaron Judge","at_bat":100,"hit":28,"double":7,"triple":2,"home_run":5,"ball_on_base":8,"hit_by_pitch":3}`,
	`{"name":"Freddie Freeman","at_bat":100,"hit":32,"double":8,"triple":1,"home_run":6,"ball_on_base":9,"hit_by_pitch":2}`,
	`{"name":"Juan Soto","at_bat":100,"hit":27,"double":4,"triple":2,"home_run":3,"ball_on_base":11,"hit_by_pitch":1}`,
	`{"name":"Fernando Tatis Jr.","at_bat":100,"hit":29,"double":5,"triple":1,"home_run":4,"ball_on_base":10,"hit_by_pitch":2}`,
	`{"name":"Bryce Harper","at_bat":100,"hit":26,"double":6,"triple":0,"home_run":3,"ball_on_base":12,"hit_by_pitch":1}`,
	`{"name":"Ronald Acuna Jr.","at_bat":100,"hit":31,"double":7,"triple":2,"home_run":5,"ball_on_base":8,"hit_by_pitch":3}`,
	`{"name":"Shohei Ohtani","at_bat":100,"hit":33,"double":8,"triple":1,"home_run":6,"ball_on_base":9,"hit_by_pitch":2}`,
}

var validLineupJSON = `[` + strings.Join(validBatters, ",") + `]`

var impossibleOutLineupJSON = `[` + strings.Join([]string{
	`{"name":"Mike Trout","at_bat":100,"hit":100,"double":5,"triple":1,"home_run":4,"ball_on_base":10,"hit_by_pitch":2}`,
	`{"name":"Mookie Betts","at_bat":100,"hit":100,"double":6,"triple":0,"home_run":3,"ball_on_base":12,"hit_by_pitch":1}`,
	`{"name":"Aaron Judge","at_bat":100,"hit":100,"double":7,"triple":2,"home_run":5,"ball_on_base":8,"hit_by_pitch":3}`,
	`{"name":"Freddie Freeman","at_bat":100,"hit":100,"double":8,"triple":1,"home_run":6,"ball_on_base":9,"hit_by_pitch":2}`,
	`{"name":"Juan Soto","at_bat":100,"hit":100,"double":4,"triple":2,"home_run":3,"ball_on_base":11,"hit_by_pitch":1}`,
	`{"name":"Fernando Tatis Jr.","at_bat":100,"hit":100,"double":5,"triple":1,"home_run":4,"ball_on_base":10,"hit_by_pitch":2}`,
	`{"name":"Bryce Harper","at_bat":100,"hit":100,"double":6,"triple":0,"home_run":3,"ball_on_base":12,"hit_by_pitch":1}`,
	`{"name":"Ronald Acuna Jr.","at_bat":100,"hit":100,"double":7,"triple":2,"home_run":5,"ball_on_base":8,"hit_by_pitch":3}`,
	`{"name":"Shohei Ohtani","at_bat":100,"hit":100,"double":8,"triple":1,"home_run":6,"ball_on_base":9,"hit_by_pitch":2}`,
}, ",") + `]`
