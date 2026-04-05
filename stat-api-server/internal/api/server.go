package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Team struct {
	Name string `json:"name"`
	Year int    `json:"year"`
}

type StatStore interface {
	GetTeams() ([]Team, error)
	GetBattingStat(team string, year int) ([]map[string]any, error)
}

type Server struct {
	store StatStore
}

func NewServer(store StatStore) *Server {
	return &Server{store: store}
}

func NewHandler(server *Server, allowedOrigin string) http.Handler {
	router := mux.NewRouter()
	router.StrictSlash(true)
	router.HandleFunc("/teams", server.GetTeamsHandler).Methods(http.MethodGet)
	router.HandleFunc("/batting", server.GetBattingStatHandler).Methods(http.MethodGet)

	return cors.New(cors.Options{
		AllowedOrigins: []string{allowedOrigin},
	}).Handler(router)
}

func (s *Server) GetTeamsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("GetTeamsHandler is called")

	data, err := s.store.GetTeams()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) GetBattingStatHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("GetBattingStatHandler is called")

	query := r.URL.Query()
	team := query.Get("team")
	year, err := strconv.Atoi(query.Get("year"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("Team:", team, "year:", year)

	data, err := s.store.GetBattingStat(team, year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}
