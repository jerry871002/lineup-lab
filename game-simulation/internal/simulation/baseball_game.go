package simulation

import (
	"log"
	"reflect"
)

type BaseballGame struct {
	Inning    int
	Outs      int
	Score     int
	Hits      int
	Runners   []int
	EndOfGame bool
}

func NewBaseballGame() *BaseballGame {
	return &BaseballGame{}
}

func (g *BaseballGame) Reset() {
	g.Inning = 1
	g.Outs = 0
	g.Score = 0
	g.Hits = 0
	g.Runners = []int{0, 0, 0}
	g.EndOfGame = false
}

func (g *BaseballGame) SimulateGame(lineup []Batter) {
	if len(lineup) != 9 {
		log.Fatal("Lineup must have 9 batters")
	}

	g.Reset()
	battingOrder := lineup
	for !g.EndOfGame {
		batter := battingOrder[0]
		g.SimulateOneBatter(&batter)
		battingOrder = append(battingOrder[1:], batter)
	}
}

func (g *BaseballGame) SimulateOneBatter(batter *Batter) {
	outcome := weightedChoice(
		[]string{"out", "ball_on_base", "hit_by_pitch", "hit"},
		[]float64{batter.OutProbability(), batter.BallOnBaseProbability(), batter.HitByPitchProbability(), batter.HitProbability()},
	)

	switch outcome {
	case "out":
		g.HandleOut(batter)
	case "ball_on_base", "hit_by_pitch":
		g.HandleAwardBase(batter)
	case "hit":
		g.HandleHit(batter)
	}
}

func (g *BaseballGame) HandleOut(batter *Batter) {
	debugLogger.Printf("Batter %s is out", batter.Name)
	g.Outs++
	if g.Outs == 3 {
		g.EndOfInning()
	}
}

func (g *BaseballGame) EndOfInning() {
	if g.Inning == 9 {
		g.EndOfGame = true
		debugLogger.Printf("End of game. Final score: %d", g.Score)
	} else {
		debugLogger.Printf("End of inning %d. Score: %d", g.Inning, g.Score)
		g.Inning++
		g.Outs = 0
		g.Runners = []int{0, 0, 0}
	}
}

func (g *BaseballGame) HandleAwardBase(batter *Batter) {
	debugLogger.Printf("Batter %s is awarded to first base (BB or HBP)", batter.Name)
	if g.Runners[0] == 0 {
		g.Runners[0] = 1
	} else if reflect.DeepEqual(g.Runners, []int{1, 1, 1}) {
		debugLogger.Printf("Batter %s got 1 RBI", batter.Name)
		g.Score++
	} else if sum(g.Runners) == 2 {
		g.Runners = []int{1, 1, 1}
	} else {
		g.Runners = append([]int{1}, g.Runners[:len(g.Runners)-1]...)
	}
}

func (g *BaseballGame) HandleHit(batter *Batter) {
	g.Hits++
	advanceBases := g.GetHitAdvanceBases(batter)
	if advanceBases == 4 {
		g.HandleHomeRun(batter)
	} else {
		g.HandleHitAdvance(batter, advanceBases)
	}
}

func (g *BaseballGame) HandleHomeRun(batter *Batter) {
	score := sum(g.Runners) + 1
	debugLogger.Printf("Batter %s hits a home run with %d RBIs", batter.Name, score)
	g.Score += score
	g.Runners = []int{0, 0, 0}
}

func (g *BaseballGame) HandleHitAdvance(batter *Batter, advanceBases int) {
	score := sum(g.Runners[len(g.Runners)-advanceBases:])
	var hitType string
	switch advanceBases {
	case 1:
		hitType = "single"
	case 2:
		hitType = "double"
	case 3:
		hitType = "triple"
	}
	debugLogger.Printf("Batter %s hits a %s with %d RBIs", batter.Name, hitType, score)
	g.Score += score
	newRunners := make([]int, advanceBases-1)
	newRunners = append(newRunners, 1)
	g.Runners = append(newRunners, g.Runners[:len(g.Runners)-advanceBases]...)
}

func (g *BaseballGame) GetHitAdvanceBases(batter *Batter) int {
	advanceProbability := batter.HitAdvanceProbability()
	keys := []int{1, 2, 3, 4}
	weights := []float64{
		advanceProbability[1],
		advanceProbability[2],
		advanceProbability[3],
		advanceProbability[4],
	}
	return weightedChoice(keys, weights)
}
