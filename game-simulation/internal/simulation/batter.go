package simulation

type Batter struct {
	Name       string `json:"name"`
	AtBat      int    `json:"at_bat"`
	Hit        int    `json:"hit"`
	Double     int    `json:"double"`
	Triple     int    `json:"triple"`
	HomeRun    int    `json:"home_run"`
	BallOnBase int    `json:"ball_on_base"`
	HitByPitch int    `json:"hit_by_pitch"`
}

func (b *Batter) Single() int {
	return b.Hit - b.Double - b.Triple - b.HomeRun
}

func (b *Batter) PlateAppearance() int {
	return b.AtBat + b.BallOnBase + b.HitByPitch
}

func (b *Batter) OutProbability() float64 {
	return float64(b.AtBat-b.Hit) / float64(b.PlateAppearance())
}

func (b *Batter) BallOnBaseProbability() float64 {
	return float64(b.BallOnBase) / float64(b.PlateAppearance())
}

func (b *Batter) HitByPitchProbability() float64 {
	return float64(b.HitByPitch) / float64(b.PlateAppearance())
}

func (b *Batter) HitProbability() float64 {
	return float64(b.Hit) / float64(b.PlateAppearance())
}

func (b *Batter) HitAdvanceProbability() map[int]float64 {
	return map[int]float64{
		1: float64(b.Single()) / float64(b.PlateAppearance()),
		2: float64(b.Double) / float64(b.PlateAppearance()),
		3: float64(b.Triple) / float64(b.PlateAppearance()),
		4: float64(b.HomeRun) / float64(b.PlateAppearance()),
	}
}
