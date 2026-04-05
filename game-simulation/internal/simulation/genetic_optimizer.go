package simulation

import (
	"math/rand"
	"sync"
	"time"
)

type GeneticOptimizer struct {
	Optimizer
	MaxGeneration  int
	PopulationSize int
	MutationRate   float64
}

func NewGeneticOptimizer(maxGeneration, populationSize int, mutationRate float64) *GeneticOptimizer {
	return &GeneticOptimizer{
		MaxGeneration:  maxGeneration,
		PopulationSize: populationSize,
		MutationRate:   mutationRate,
	}
}

func (g *GeneticOptimizer) Optimize(roster Roster) Lineup {
	population := make([]Lineup, g.PopulationSize)
	for i := 0; i < g.PopulationSize; i++ {
		population[i] = g.GenerateRandomLineup(roster)
	}

	maxFitnessScore := 0.0
	bestLineup := Lineup{}
	for gen := 0; gen < g.MaxGeneration; gen++ {
		infoLogger.Println("Generation:", gen)

		startTime := time.Now()
		fitnessScores, genMaxFitnessScore, genBestLineup := g.EvaluatePopulation(population)
		if genMaxFitnessScore > maxFitnessScore {
			maxFitnessScore = genMaxFitnessScore
			bestLineup = genBestLineup
		}

		elapsedTime := time.Since(startTime)
		infoLogger.Printf("Evaluating population took %s", elapsedTime)
		infoLogger.Println("Best fitness score so far:", maxFitnessScore)

		parents := g.SelectParents(population, fitnessScores)
		population = g.CrossoverAndMutate(parents)
	}

	for _, lineup := range population {
		fitnessScore := g.ComputeFitness(lineup)
		if fitnessScore > maxFitnessScore {
			maxFitnessScore = fitnessScore
			bestLineup = lineup
		}
	}

	infoLogger.Println("Best lineup:", bestLineup, "with fitness score:", maxFitnessScore)

	return bestLineup
}

func (g *GeneticOptimizer) GenerateRandomLineup(roster Roster) Lineup {
	lineup := make([]Batter, 9)
	inLineup := map[string]bool{}
	for i := 0; i < 9; i++ {
		var candidate Batter
		for {
			candidate = roster[rand.Intn(len(roster))]
			if !inLineup[candidate.Name] {
				break
			}
		}
		lineup[i] = candidate
		inLineup[candidate.Name] = true
	}
	return lineup
}

func (g *GeneticOptimizer) EvaluatePopulation(population []Lineup) ([]float64, float64, Lineup) {
	fitnessScores := make([]float64, g.PopulationSize)
	var (
		wg              sync.WaitGroup
		mu              sync.Mutex
		maxFitnessScore float64
		bestLineup      Lineup
	)

	for i, lineup := range population {
		wg.Add(1)
		go func(i int, lineup Lineup) {
			defer wg.Done()
			fitness := g.ComputeFitness(lineup)
			fitnessScores[i] = fitness

			mu.Lock()
			if fitness > maxFitnessScore {
				maxFitnessScore = fitness
				bestLineup = lineup
			}
			mu.Unlock()
		}(i, lineup)
	}

	wg.Wait()
	return fitnessScores, maxFitnessScore, bestLineup
}

func (g *GeneticOptimizer) ComputeFitness(lineup Lineup) float64 {
	result := SimulateGamesInParallel([]Batter(lineup), 1000, 1)
	return result["average_score"]*100 + result["average_hits"]
}

func (g *GeneticOptimizer) SelectParents(population []Lineup, fitnessScores []float64) []Lineup {
	parents := make([]Lineup, g.PopulationSize)
	for i := 0; i < g.PopulationSize; i++ {
		a, b := rand.Intn(g.PopulationSize), rand.Intn(g.PopulationSize)
		if fitnessScores[a] > fitnessScores[b] {
			parents[i] = population[a]
		} else {
			parents[i] = population[b]
		}
	}
	return parents
}

func (g *GeneticOptimizer) CrossoverAndMutate(parents []Lineup) []Lineup {
	children := make([]Lineup, g.PopulationSize)
	for i := 0; i < len(parents)/2; i++ {
		p1, p2 := parents[2*i], parents[2*i+1]
		c1, c2 := g.Crossover(p1, p2)
		children[2*i], children[2*i+1] = g.Mutate(c1, c2)
	}
	return children
}

func (g *GeneticOptimizer) Crossover(p1, p2 Lineup) (Lineup, Lineup) {
	child1, child2 := make([]Batter, 9), make([]Batter, 9)

	for _, child := range []Lineup{child1, child2} {
		standbyCandidates := make([]Batter, 0, 9)
		selectedBatters := make(map[string]bool, 9)

		for i := 0; i < 9; i++ {
			candidate1, candidate2 := p1[i], p2[i]
			child[i] = g.selectBatter(candidate1, candidate2, selectedBatters, standbyCandidates)
			selectedBatters[child[i].Name] = true
			if child[i] == candidate1 {
				standbyCandidates = append(standbyCandidates, candidate2)
			} else {
				standbyCandidates = append(standbyCandidates, candidate1)
			}
		}
	}

	return child1, child2
}

func (g *GeneticOptimizer) selectBatter(candidate1, candidate2 Batter, selectedBatters map[string]bool, standbyCandidates []Batter) Batter {
	if selectedBatters[candidate1.Name] && selectedBatters[candidate2.Name] {
		return g.selectFromStandby(standbyCandidates, selectedBatters)
	} else if selectedBatters[candidate1.Name] {
		return candidate2
	} else if selectedBatters[candidate2.Name] {
		return candidate1
	}
	if rand.Float64() < 0.5 {
		return candidate1
	}
	return candidate2
}

func (g *GeneticOptimizer) selectFromStandby(standbyCandidates []Batter, selectedBatters map[string]bool) Batter {
	for {
		candidate := standbyCandidates[rand.Intn(len(standbyCandidates))]
		if !selectedBatters[candidate.Name] {
			return candidate
		}
	}
}

func (g *GeneticOptimizer) Mutate(c1, c2 Lineup) (Lineup, Lineup) {
	for _, child := range []Lineup{c1, c2} {
		if rand.Float64() < 0.1 {
			i, j := rand.Intn(9), rand.Intn(9)
			child[i], child[j] = child[j], child[i]
		}
	}
	return c1, c2
}
