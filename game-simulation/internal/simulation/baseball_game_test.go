package simulation

import (
	"io"
	"log"
	"reflect"
	"testing"
)

func TestHandleAwardBase(t *testing.T) {
	ConfigureLoggers(log.New(io.Discard, "", 0), log.New(io.Discard, "", 0))

	g := &BaseballGame{
		Runners: []int{0, 0, 0},
	}

	b := &Batter{Name: "John Doe"}

	g.HandleAwardBase(b)

	expectedRunners := []int{1, 0, 0}
	if !reflect.DeepEqual(g.Runners, expectedRunners) {
		t.Errorf("HandleAwardBase() runners = %v, want %v", g.Runners, expectedRunners)
	}

	g = &BaseballGame{
		Runners: []int{1, 0, 0},
	}

	g.HandleAwardBase(b)

	expectedRunners = []int{1, 1, 0}
	if !reflect.DeepEqual(g.Runners, expectedRunners) {
		t.Errorf("HandleAwardBase() runners = %v, want %v", g.Runners, expectedRunners)
	}

	g = &BaseballGame{
		Runners: []int{1, 1, 1},
		Score:   0,
	}

	g.HandleAwardBase(b)

	expectedRunners = []int{1, 1, 1}
	if !reflect.DeepEqual(g.Runners, expectedRunners) {
		t.Errorf("HandleAwardBase() runners = %v, want %v", g.Runners, expectedRunners)
	}

	expectedScore := 1
	if g.Score != expectedScore {
		t.Errorf("HandleAwardBase() score = %d, want %d", g.Score, expectedScore)
	}
}

func TestHandleHitAdvance(t *testing.T) {
	ConfigureLoggers(log.New(io.Discard, "", 0), log.New(io.Discard, "", 0))

	g := &BaseballGame{
		Runners: []int{0, 1, 1},
		Score:   2,
	}

	b := &Batter{Name: "John Doe"}

	g.HandleHitAdvance(b, 2)

	expectedScore := 4
	if g.Score != expectedScore {
		t.Errorf("HandleHitAdvance() score = %d, want %d", g.Score, expectedScore)
	}

	expectedRunners := []int{0, 1, 0}
	if !reflect.DeepEqual(g.Runners, expectedRunners) {
		t.Errorf("HandleHitAdvance() runners = %v, want %v", g.Runners, expectedRunners)
	}
}

func TestHandleHomeRun(t *testing.T) {
	ConfigureLoggers(log.New(io.Discard, "", 0), log.New(io.Discard, "", 0))

	g := &BaseballGame{
		Runners: []int{0, 1, 1},
		Score:   2,
	}

	b := &Batter{Name: "John Doe"}

	g.HandleHomeRun(b)

	expectedScore := 5
	if g.Score != expectedScore {
		t.Errorf("HandleHitAdvance() score = %d, want %d", g.Score, expectedScore)
	}

	expectedRunners := []int{0, 0, 0}
	if !reflect.DeepEqual(g.Runners, expectedRunners) {
		t.Errorf("HandleHitAdvance() runners = %v, want %v", g.Runners, expectedRunners)
	}

	g = &BaseballGame{
		Runners: []int{0, 0, 0},
		Score:   0,
	}

	g.HandleHomeRun(b)

	expectedScore = 1
	if g.Score != expectedScore {
		t.Errorf("HandleHitAdvance() score = %d, want %d", g.Score, expectedScore)
	}

	expectedRunners = []int{0, 0, 0}
	if !reflect.DeepEqual(g.Runners, expectedRunners) {
		t.Errorf("HandleHitAdvance() runners = %v, want %v", g.Runners, expectedRunners)
	}
}
