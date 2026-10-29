package main

import (
	"testing"
	"time"
)

func TestSimulationTimeoutConfigurationAndBudgetConsistency(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		want        time.Duration
		invalid     bool
	}{
		{"default", "", 85 * time.Second, false},
		{"custom", "120s", 120 * time.Second, false},
		{"duration units", "2m", 2 * time.Minute, false},
		{"invalid unit", "90", 0, true},
		{"negative", "-1s", 0, true},
		{"zero", "0s", 0, true},
		{"overflowing budgets", "2562047h47m16s", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SIMULATION_TIMEOUT", tc.value)
			timeout, err := simulationTimeoutFromEnv()
			if (err != nil) != tc.invalid {
				t.Fatalf("configuration error=%v, invalid=%v", err, tc.invalid)
			}
			if tc.invalid {
				return
			}
			if timeout != tc.want {
				t.Fatalf("timeout=%s, want%s", timeout, tc.want)
			}
			requestBudget := (&Server{SimulationTimeout: timeout}).simulationBudget()
			writeBudget := requestBudget + responseWriteGrace
			if requestBudget <= timeout || writeBudget <= requestBudget {
				t.Fatalf("invalid budget order: %s/%s/%s", timeout, requestBudget, writeBudget)
			}
		})
	}
}
