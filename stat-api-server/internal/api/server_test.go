package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/jerry871002/lineup-lab/stat-api-server/internal/api"
	"github.com/jerry871002/lineup-lab/stat-api-server/internal/store"
)

func TestGetTeams(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/teams/", nil)
	if err != nil {
		t.Fatal(err)
	}

	server := api.NewServer(store.NewMockStatStore())

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
	req, err := http.NewRequest(http.MethodGet, "/batting/?team=test&year=2024", nil)
	if err != nil {
		t.Fatal(err)
	}

	server := api.NewServer(store.NewMockStatStore())

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
