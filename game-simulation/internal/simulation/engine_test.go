package simulation

import (
	"io"
	"log"
	"testing"
)

func TestWeightedChoiceReturnsOnlyNonZeroWeight(t *testing.T) {
	got := weightedChoice([]string{"first", "second", "third"}, []float64{0, 1, 0})
	if got != "second" {
		t.Fatalf("weightedChoice() = %q, want %q", got, "second")
	}
}

func TestSimulateBatchWorker(t *testing.T) {
	ConfigureLoggers(log.New(io.Discard, "", 0), log.New(io.Discard, "", 0))

	scoreChan := make(chan int, 1)
	hitChan := make(chan int, 1)

	simulateBatchWorker(alwaysOutLineup(), 3, scoreChan, hitChan)

	if got := <-scoreChan; got != 0 {
		t.Fatalf("simulateBatchWorker score = %d, want 0", got)
	}

	if got := <-hitChan; got != 0 {
		t.Fatalf("simulateBatchWorker hits = %d, want 0", got)
	}
}

func TestSimulateGamesInParallel(t *testing.T) {
	ConfigureLoggers(log.New(io.Discard, "", 0), log.New(io.Discard, "", 0))

	result := SimulateGamesInParallel(alwaysOutLineup(), 4, 2)

	if got := result["average_score"]; got != 0 {
		t.Fatalf("average_score = %v, want 0", got)
	}

	if got := result["average_hits"]; got != 0 {
		t.Fatalf("average_hits = %v, want 0", got)
	}
}

func alwaysOutLineup() []Batter {
	lineup := make([]Batter, 9)
	for i := range lineup {
		lineup[i] = Batter{
			Name:  "Batter",
			AtBat: 1,
			Hit:   0,
		}
	}
	return lineup
}
