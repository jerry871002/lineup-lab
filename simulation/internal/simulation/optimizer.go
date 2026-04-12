package simulation

type Roster []Batter
type Lineup []Batter

type Optimizer interface {
	Optimize(Roster) Lineup
}
