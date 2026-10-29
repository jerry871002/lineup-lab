package main

import (
	"fmt"
	"os"
	"time"
)

const (
	defaultSimulationTimeout = 85 * time.Second
	crudRequestTimeout       = 10 * time.Second
	historyPersistenceGrace  = 5 * time.Second
	responseWriteGrace       = 10 * time.Second
)

func simulationTimeoutFromEnv() (time.Duration, error) {
	value := os.Getenv("SIMULATION_TIMEOUT")
	if value == "" {
		return defaultSimulationTimeout, nil
	}
	timeout, err := time.ParseDuration(value)
	// The request and write budgets add their grace periods to this timeout.
	maximum := time.Duration(1<<63-1) - historyPersistenceGrace - responseWriteGrace
	if err != nil || timeout <= 0 || timeout > maximum {
		return 0, fmt.Errorf("SIMULATION_TIMEOUT must be a positive duration with room for request and write grace periods")
	}
	return timeout, nil
}

func (s *Server) simulationBudget() time.Duration {
	timeout := s.SimulationTimeout
	if timeout == 0 {
		timeout = defaultSimulationTimeout
	}
	return timeout + historyPersistenceGrace
}
