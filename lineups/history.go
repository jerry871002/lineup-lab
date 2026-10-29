package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type SimulationResult struct {
	AverageScore     float64 `json:"average_score"`
	AverageHits      float64 `json:"average_hits"`
	SampleCount      int     `json:"sample_count"`
	SimulatorVersion string  `json:"simulator_version"`
}

type SimulationRun struct {
	ID             int64    `json:"id"`
	SavedLineupID  *int64   `json:"saved_lineup_id"`
	LineupName     string   `json:"lineup_name"`
	Team           string   `json:"team"`
	Year           int      `json:"year"`
	LineupRevision int64    `json:"lineup_revision"`
	BatterIDs      []int64  `json:"batter_ids"`
	Batters        []Batter `json:"batters"`
	SimulationResult
	CreatedAt time.Time `json:"created_at"`
}

type HistoryStore interface {
	AddRun(context.Context, int64, SavedLineup, SimulationResult) (SimulationRun, error)
	ListRuns(context.Context, int64) ([]SimulationRun, error)
}

type Simulator func(context.Context, []Batter) (SimulationResult, error)

func (s *Server) runSimulation(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	lineup, err := s.Store.Get(r.Context(), userID(r), id)
	if err != nil {
		storeError(w, err)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024))
	if err != nil || len(bytes.TrimSpace(body)) != 0 {
		apiError(w, http.StatusBadRequest, "simulation requests must not include a body")
		return
	}
	if s.Simulate == nil || s.History == nil {
		apiError(w, http.StatusServiceUnavailable, "simulation history unavailable")
		return
	}
	result, err := s.Simulate(r.Context(), lineup.Batters)
	if err != nil || !validSimulationResult(result) {
		log.Printf("simulate saved lineup: %v", err)
		apiError(w, http.StatusBadGateway, "simulation did not return a valid result")
		return
	}
	run, err := s.History.AddRun(r.Context(), userID(r), lineup, result)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) listSimulations(w http.ResponseWriter, r *http.Request) {
	if s.History == nil {
		apiError(w, http.StatusServiceUnavailable, "simulation history unavailable")
		return
	}
	items, err := s.History.ListRuns(r.Context(), userID(r))
	if err != nil {
		storeError(w, err)
		return
	}
	if items == nil {
		items = []SimulationRun{}
	}
	writeJSON(w, http.StatusOK, items)
}

func validSimulationResult(result SimulationResult) bool {
	return result.AverageScore >= 0 && !math.IsNaN(result.AverageScore) && !math.IsInf(result.AverageScore, 0) &&
		result.AverageHits >= 0 && !math.IsNaN(result.AverageHits) && !math.IsInf(result.AverageHits, 0) &&
		result.SampleCount > 0 && result.SampleCount <= 2147483647 &&
		strings.TrimSpace(result.SimulatorVersion) != "" && utf8.RuneCountInString(result.SimulatorVersion) <= 100
}

func simulateLineup(client *http.Client, base string) Simulator {
	return func(ctx context.Context, batters []Batter) (SimulationResult, error) {
		// The simulator accepts exactly these eight fields, without stored IDs.
		type simulationBatter struct {
			Name       string `json:"name"`
			AtBat      int    `json:"at_bat"`
			Hit        int    `json:"hit"`
			Double     int    `json:"double"`
			Triple     int    `json:"triple"`
			HomeRun    int    `json:"home_run"`
			BallOnBase int    `json:"ball_on_base"`
			HitByPitch int    `json:"hit_by_pitch"`
		}
		payload := make([]simulationBatter, len(batters))
		for i, batter := range batters {
			payload[i] = simulationBatter{batter.Name, batter.AtBat, batter.Hit, batter.Double, batter.Triple, batter.HomeRun, batter.BallOnBase, batter.HitByPitch}
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return SimulationResult{}, err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/simulate", bytes.NewReader(encoded))
		if err != nil {
			return SimulationResult{}, err
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			return SimulationResult{}, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return SimulationResult{}, errors.New("simulation service rejected request")
		}
		var wire struct {
			AverageScore     *float64 `json:"average_score"`
			AverageHits      *float64 `json:"average_hits"`
			SampleCount      *int     `json:"sample_count"`
			SimulatorVersion *string  `json:"simulator_version"`
		}
		decoder := json.NewDecoder(io.LimitReader(response.Body, 65536))
		if err := decoder.Decode(&wire); err != nil {
			return SimulationResult{}, err
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			return SimulationResult{}, errors.New("simulation response must contain one JSON value")
		}
		if wire.AverageScore == nil || wire.AverageHits == nil || wire.SampleCount == nil || wire.SimulatorVersion == nil {
			return SimulationResult{}, errors.New("simulation result metadata missing")
		}
		result := SimulationResult{*wire.AverageScore, *wire.AverageHits, *wire.SampleCount, *wire.SimulatorVersion}
		if !validSimulationResult(result) {
			return SimulationResult{}, errors.New("invalid simulation result metadata")
		}
		return result, nil
	}
}
