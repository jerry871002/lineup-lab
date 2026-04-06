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

func TestNewBaseballGameResetAndSimulate(t *testing.T) {
	ConfigureLoggers(log.New(io.Discard, "", 0), log.New(io.Discard, "", 0))

	game := NewBaseballGame()
	game.Reset()

	if game.Inning != 1 || game.Outs != 0 || game.Score != 0 || game.Hits != 0 || game.EndOfGame {
		t.Fatalf("Reset() produced unexpected game state: %+v", game)
	}

	game.SimulateGame(alwaysOutLineup())

	if !game.EndOfGame {
		t.Fatal("SimulateGame() EndOfGame = false, want true")
	}

	if game.Inning != 9 {
		t.Fatalf("SimulateGame() inning = %d, want 9", game.Inning)
	}

	if game.Score != 0 {
		t.Fatalf("SimulateGame() score = %d, want 0", game.Score)
	}
}

func TestSimulateOneBatterOutAndHit(t *testing.T) {
	ConfigureLoggers(log.New(io.Discard, "", 0), log.New(io.Discard, "", 0))

	game := NewBaseballGame()
	game.Reset()

	game.SimulateOneBatter(&Batter{
		Name:  "Out Batter",
		AtBat: 1,
		Hit:   0,
	})

	if game.Outs != 1 {
		t.Fatalf("SimulateOneBatter() outs = %d, want 1", game.Outs)
	}

	game.Reset()
	game.SimulateOneBatter(&Batter{
		Name:    "Home Run Batter",
		AtBat:   1,
		Hit:     1,
		HomeRun: 1,
	})

	if game.Score != 1 {
		t.Fatalf("SimulateOneBatter() score = %d, want 1", game.Score)
	}

	if game.Hits != 1 {
		t.Fatalf("SimulateOneBatter() hits = %d, want 1", game.Hits)
	}
}

func TestHandleOutEndsInningAndGame(t *testing.T) {
	ConfigureLoggers(log.New(io.Discard, "", 0), log.New(io.Discard, "", 0))

	game := &BaseballGame{
		Inning:  1,
		Outs:    2,
		Runners: []int{1, 1, 0},
	}

	game.HandleOut(&Batter{Name: "Test Batter"})

	if game.Inning != 2 || game.Outs != 0 || !reflect.DeepEqual(game.Runners, []int{0, 0, 0}) {
		t.Fatalf("HandleOut() did not advance inning correctly: %+v", game)
	}

	game = &BaseballGame{
		Inning: 9,
		Outs:   2,
	}

	game.HandleOut(&Batter{Name: "Test Batter"})

	if !game.EndOfGame {
		t.Fatal("HandleOut() EndOfGame = false, want true")
	}
}

func TestHandleHitAndGetHitAdvanceBases(t *testing.T) {
	ConfigureLoggers(log.New(io.Discard, "", 0), log.New(io.Discard, "", 0))

	game := &BaseballGame{
		Runners: []int{1, 0, 0},
	}

	batter := &Batter{
		Name:   "Double Batter",
		AtBat:  1,
		Hit:    1,
		Double: 1,
	}

	if got := game.GetHitAdvanceBases(batter); got != 2 {
		t.Fatalf("GetHitAdvanceBases() = %d, want 2", got)
	}

	game.HandleHit(batter)

	if game.Hits != 1 {
		t.Fatalf("HandleHit() hits = %d, want 1", game.Hits)
	}

	if game.Score != 0 {
		t.Fatalf("HandleHit() score = %d, want 0", game.Score)
	}

	if !reflect.DeepEqual(game.Runners, []int{0, 1, 1}) {
		t.Fatalf("HandleHit() runners = %v, want %v", game.Runners, []int{0, 1, 1})
	}
}
