package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/jerry871002/lineup-lab/stats/internal/api"
	"github.com/jerry871002/lineup-lab/stats/internal/store"
)

func TestGetTeams(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/teams", nil)
	if err != nil {
		t.Fatal(err)
	}

	server := api.NewServer(store.NewMockStatStore(), nil)

	response := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetTeamsHandler)
	handler.ServeHTTP(response, req)

	if status := response.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := `[{"name":"Team1","year":2024},{"name":"Team2","year":2024},{"name":"Team3","year":2024}]`
	if !isJSONEqual(response.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", response.Body.String(), expected)
	}
}

func TestGetBatting(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/batting?team=test&year=2024", nil)
	if err != nil {
		t.Fatal(err)
	}

	server := api.NewServer(store.NewMockStatStore(), nil)

	response := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetBattingStatHandler)
	handler.ServeHTTP(response, req)

	if status := response.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		t.Errorf("error: %v", response.Body.String())
	}

	expected := `[{"name":"Player1","at_bat":"50","hit:":"10"},{"name":"Player2","at_bat":"100","hit:":"20"},{"name":"Player3","at_bat":"150","hit:":"30"}]`
	if !isJSONEqual(response.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", response.Body.String(), expected)
	}
}

func isJSONEqual(obj1, obj2 string) bool {
	var o1 any
	var o2 any
	if err := json.Unmarshal([]byte(obj1), &o1); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(obj2), &o2); err != nil {
		return false
	}
	return reflect.DeepEqual(o1, o2)
}

func TestHealthz(t *testing.T) {
	server := api.NewServer(store.NewMockStatStore(), nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	handler := api.NewHandler(server, "http://localhost:3000")
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestReadyz(t *testing.T) {
	testCases := []struct {
		name       string
		readyCheck func(context.Context) error
		wantStatus int
	}{
		{
			name:       "ready without check",
			readyCheck: nil,
			wantStatus: http.StatusOK,
		},
		{
			name: "ready with healthy check",
			readyCheck: func(context.Context) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "ready with failing check",
			readyCheck: func(context.Context) error {
				return errors.New("db down")
			},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := api.NewServer(store.NewMockStatStore(), testCase.readyCheck)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

			handler := api.NewHandler(server, "http://localhost:3000")
			handler.ServeHTTP(response, request)

			if response.Code != testCase.wantStatus {
				t.Fatalf("readyz status = %d, want %d", response.Code, testCase.wantStatus)
			}
		})
	}
}

func TestRoutesRejectTrailingSlashVariants(t *testing.T) {
	server := api.NewServer(store.NewMockStatStore(), nil)
	handler := api.NewHandler(server, "http://localhost:3000")

	testCases := []struct {
		name string
		path string
		want int
	}{
		{name: "teams slash", path: "/teams/", want: http.StatusNotFound},
		{name: "batting slash", path: "/batting/?team=test&year=2024", want: http.StatusNotFound},
		{name: "ready slash", path: "/readyz/", want: http.StatusNotFound},
		{name: "health slash", path: "/healthz/", want: http.StatusNotFound},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, testCase.path, nil)

			handler.ServeHTTP(response, request)

			if response.Code != testCase.want {
				t.Fatalf("%s status = %d, want %d", testCase.path, response.Code, testCase.want)
			}
		})
	}
}
