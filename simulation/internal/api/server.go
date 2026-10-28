package api

import (
	"log"
	"net/http"
	"runtime"

	"github.com/jerry871002/lineup-lab/simulation/internal/simulation"
	"github.com/rs/cors"
)

const simulatorVersion = "lineup-engine-v1"

type simulationResult struct {
	AverageScore     float64 `json:"average_score"`
	AverageHits      float64 `json:"average_hits"`
	SampleCount      int     `json:"sample_count"`
	SimulatorVersion string  `json:"simulator_version"`
}

func newSimulationResult(averages map[string]float64, numGames int) simulationResult {
	return simulationResult{
		AverageScore:     averages["average_score"],
		AverageHits:      averages["average_hits"],
		SampleCount:      numGames,
		SimulatorVersion: simulatorVersion,
	}
}

type Server struct {
	debugMode bool
}

var newOptimizer = func() simulation.Optimizer {
	return simulation.NewGeneticOptimizer(50, 50, 0.2)
}

func NewHandler(debugMode bool, allowedOrigin string) http.Handler {
	server := &Server{debugMode: debugMode}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", server.healthHandler)
	mux.HandleFunc("/readyz", server.readyHandler)
	mux.HandleFunc("/simulate", server.simulateHandler)
	mux.HandleFunc("/optimize", server.optimizeHandler)

	return cors.New(cors.Options{
		AllowedOrigins: []string{allowedOrigin},
	}).Handler(mux)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
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
	_ = writeJSON(w, newSimulationResult(result, numGames))
}

func (s *Server) optimizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		log.Println("Invalid request method:", r.Method)
		return
	}

	roster, err := decodeAndValidateRoster(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}

	optimizer := newOptimizer()
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
