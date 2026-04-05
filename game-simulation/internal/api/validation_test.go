package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jerry871002/lineup-lab/game-simulation/internal/simulation"
)

func TestValidateLineupAcceptsValidLineup(t *testing.T) {
	lineup := mustDecodeLineup(t, validLineupJSON)

	if err := validateLineup(lineup); err != nil {
		t.Fatalf("validateLineup() error = %v, want nil", err)
	}
}

func TestValidateLineupRejectsInvalidLineup(t *testing.T) {
	testCases := []struct {
		name    string
		lineup  []simulation.Batter
		wantErr string
	}{
		{
			name:    "wrong lineup size",
			lineup:  mustDecodeLineup(t, `[`+strings.Join(validBatters[:8], ",")+`]`),
			wantErr: "lineup must have 9 batters",
		},
		{
			name: "empty name",
			lineup: func() []simulation.Batter {
				lineup := mustDecodeLineup(t, validLineupJSON)
				lineup[0].Name = "   "
				return lineup
			}(),
			wantErr: "batter 0 name must not be empty",
		},
		{
			name: "negative stat",
			lineup: func() []simulation.Batter {
				lineup := mustDecodeLineup(t, validLineupJSON)
				lineup[0].AtBat = -1
				return lineup
			}(),
			wantErr: "batter 0 at_bat must be non-negative",
		},
		{
			name: "zero at bat",
			lineup: func() []simulation.Batter {
				lineup := mustDecodeLineup(t, validLineupJSON)
				lineup[0].AtBat = 0
				return lineup
			}(),
			wantErr: "batter 0 at_bat must be greater than 0",
		},
		{
			name: "hits exceed at bats",
			lineup: func() []simulation.Batter {
				lineup := mustDecodeLineup(t, validLineupJSON)
				lineup[0].Hit = lineup[0].AtBat + 1
				return lineup
			}(),
			wantErr: "batter 0 hit must not exceed at_bat",
		},
		{
			name: "extra base hits exceed hits",
			lineup: func() []simulation.Batter {
				lineup := mustDecodeLineup(t, validLineupJSON)
				lineup[0].Double = 20
				lineup[0].Triple = 20
				lineup[0].HomeRun = 20
				return lineup
			}(),
			wantErr: "batter 0 doubles, triples, and home runs must not exceed hits",
		},
		{
			name: "duplicate names",
			lineup: func() []simulation.Batter {
				lineup := mustDecodeLineup(t, validLineupJSON)
				lineup[8].Name = lineup[0].Name
				return lineup
			}(),
			wantErr: `duplicates batter 0 name "Mike Trout"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateLineup(testCase.lineup)
			if err == nil {
				t.Fatal("validateLineup() error = nil, want non-nil")
			}

			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("validateLineup() error = %q, want substring %q", err.Error(), testCase.wantErr)
			}
		})
	}
}

func TestDecodeStrictJSONRejectsUnexpectedPayloads(t *testing.T) {
	testCases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "unknown field",
			body:    strings.Replace(validLineupJSON, `"hit_by_pitch":2`, `"hit_by_pitch":2,"unexpected":true`, 1),
			wantErr: "unknown field",
		},
		{
			name:    "trailing json",
			body:    validLineupJSON + `{"extra":true}`,
			wantErr: "single JSON value",
		},
		{
			name:    "malformed json",
			body:    `[{"name":"Mike Trout"`,
			wantErr: "unexpected EOF",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var lineup []simulation.Batter
			err := decodeStrictJSON(strings.NewReader(testCase.body), &lineup)
			if err == nil {
				t.Fatal("decodeStrictJSON() error = nil, want non-nil")
			}

			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("decodeStrictJSON() error = %q, want substring %q", err.Error(), testCase.wantErr)
			}
		})
	}
}

func mustDecodeLineup(t *testing.T, payload string) []simulation.Batter {
	t.Helper()

	var lineup []simulation.Batter
	if err := json.Unmarshal([]byte(payload), &lineup); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return lineup
}
