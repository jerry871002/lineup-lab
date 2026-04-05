package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jerry871002/lineup-lab/game-simulation/internal/simulation"
)

const lineupSize = 9

func writeJSON(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func decodeAndValidateLineup(r *http.Request) ([]simulation.Batter, error) {
	var lineup []simulation.Batter
	if err := decodeStrictJSON(r.Body, &lineup); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	if err := validateLineup(lineup); err != nil {
		return nil, err
	}

	return lineup, nil
}

func decodeStrictJSON(body io.Reader, dst any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain a single JSON value")
		}

		return fmt.Errorf("request body must contain a single JSON value: %w", err)
	}

	return nil
}

func validateLineup(lineup []simulation.Batter) error {
	if len(lineup) != lineupSize {
		return fmt.Errorf("lineup must have %d batters", lineupSize)
	}

	seenNames := make(map[string]int, len(lineup))
	lineupCanRecordOut := false
	for i, batter := range lineup {
		if err := validateBatter(batter, i); err != nil {
			return err
		}

		normalizedName := strings.ToLower(strings.TrimSpace(batter.Name))
		if previousIndex, exists := seenNames[normalizedName]; exists {
			return fmt.Errorf("batter %d duplicates batter %d name %q", i, previousIndex, strings.TrimSpace(batter.Name))
		}

		seenNames[normalizedName] = i
		if batter.CanRecordOut() {
			lineupCanRecordOut = true
		}
	}

	if !lineupCanRecordOut {
		return errors.New("lineup must contain at least one batter with non-zero out probability")
	}

	return nil
}

func validateBatter(batter simulation.Batter, index int) error {
	name := strings.TrimSpace(batter.Name)
	if name == "" {
		return fmt.Errorf("batter %d name must not be empty", index)
	}

	stats := []struct {
		label string
		value int
	}{
		{label: "at_bat", value: batter.AtBat},
		{label: "hit", value: batter.Hit},
		{label: "double", value: batter.Double},
		{label: "triple", value: batter.Triple},
		{label: "home_run", value: batter.HomeRun},
		{label: "ball_on_base", value: batter.BallOnBase},
		{label: "hit_by_pitch", value: batter.HitByPitch},
	}

	for _, stat := range stats {
		if stat.value < 0 {
			return fmt.Errorf("batter %d %s must be non-negative", index, stat.label)
		}
	}

	if batter.AtBat <= 0 {
		return fmt.Errorf("batter %d at_bat must be greater than 0", index)
	}

	if batter.Hit > batter.AtBat {
		return fmt.Errorf("batter %d hit must not exceed at_bat", index)
	}

	extraBaseHits := batter.Double + batter.Triple + batter.HomeRun
	if extraBaseHits > batter.Hit {
		return fmt.Errorf("batter %d doubles, triples, and home runs must not exceed hits", index)
	}

	if batter.PlateAppearance() <= 0 {
		return fmt.Errorf("batter %d plate appearances must be greater than 0", index)
	}

	return nil
}
