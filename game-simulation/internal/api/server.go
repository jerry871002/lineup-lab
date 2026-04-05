package api

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"

	"github.com/jerry871002/lineup-lab/game-simulation/internal/simulation"
	"github.com/rs/cors"
)

type Server struct {
	debugMode bool
}

func NewHandler(debugMode bool, allowedOrigin string) http.Handler {
	server := &Server{debugMode: debugMode}

	mux := http.NewServeMux()
	mux.HandleFunc("/simulate", server.simulateHandler)
	mux.HandleFunc("/optimize", server.optimizeHandler)

	return cors.New(cors.Options{
		AllowedOrigins: []string{allowedOrigin},
	}).Handler(mux)
}

func (s *Server) simulateHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("simulateHandler is called")

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		log.Println("Invalid request method:", r.Method)
		return
	}

	lineup, err := decodeAndValidateLineup(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}

	numGames, numBatches := s.simulationConfig()
	result := simulation.SimulateGamesInParallel(lineup, numGames, numBatches)

	w.Header().Set("Content-Type", "application/json")
	_ = writeJSON(w, result)
}

func (s *Server) optimizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		log.Println("Invalid request method:", r.Method)
		return
	}

	var roster []simulation.Batter
	if err := json.NewDecoder(r.Body).Decode(&roster); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Println(err)
		return
	}

	optimizer := simulation.NewGeneticOptimizer(50, 50, 0.2)
	lineup := optimizer.Optimize(roster)

	w.Header().Set("Content-Type", "application/json")
	_ = writeJSON(w, lineup)
}

func (s *Server) simulationConfig() (int, int) {
	if s.debugMode {
		return 10, 1
	}

	return 100000, runtime.NumCPU()
}
