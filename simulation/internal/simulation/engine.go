package simulation

import (
	"math/rand"
	"time"
)

func weightedChoice[T any](keys []T, weights []float64) T {
	totalWeight := 0.0
	for _, weight := range weights {
		totalWeight += weight
	}
	randVal := rand.Float64() * totalWeight
	for i, weight := range weights {
		if randVal < weight {
			return keys[i]
		}
		randVal -= weight
	}
	return keys[len(keys)-1]
}

func sum(arr []int) int {
	total := 0
	for _, v := range arr {
		total += v
	}
	return total
}

func simulateBatchWorker(lineup []Batter, numGames int, scoreChan chan<- int, hitChan chan<- int) {
	startTime := time.Now()

	game := NewBaseballGame()
	scores := 0
	hits := 0
	for i := 0; i < numGames; i++ {
		game.SimulateGame(lineup)
		scores += game.Score
		hits += game.Hits
	}
	scoreChan <- scores
	hitChan <- hits

	elapsedTime := time.Since(startTime)
	debugLogger.Printf("simulateBatchWorker took %s to simulate %d games", elapsedTime, numGames)
}

func SimulateGamesInParallel(lineup []Batter, numGames, numBatches int) map[string]float64 {
	gamePerBatch := numGames / numBatches

	scoreChan := make(chan int)
	hitChan := make(chan int)
	for i := 0; i < numBatches; i++ {
		batchGames := gamePerBatch
		if i == numBatches-1 {
			batchGames = numGames - (gamePerBatch * (numBatches - 1))
		}
		go simulateBatchWorker(lineup, batchGames, scoreChan, hitChan)
	}

	totalScore := 0
	totalHits := 0
	for i := 0; i < numBatches; i++ {
		totalScore += <-scoreChan
		totalHits += <-hitChan
	}

	averageScore := float64(totalScore) / float64(numGames)
	averageHits := float64(totalHits) / float64(numGames)
	return map[string]float64{
		"average_score": averageScore,
		"average_hits":  averageHits,
	}
}
