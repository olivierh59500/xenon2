package engine

// FrameClock separates the original simulation cadence from display updates.
// Integer accumulation avoids drift and preserves the 25/60 ratio exactly.
type FrameClock struct {
	SimulationRate int
	DisplayRate    int
	remainder      int
}

func NewFrameClock(simulationRate, displayRate int) FrameClock {
	if simulationRate <= 0 || displayRate <= 0 {
		panic("frame rates must be positive")
	}
	return FrameClock{SimulationRate: simulationRate, DisplayRate: displayRate}
}

// Advance returns the number of simulation passes due on this display update.
func (c *FrameClock) Advance() int {
	c.remainder += c.SimulationRate
	passes := c.remainder / c.DisplayRate
	c.remainder %= c.DisplayRate
	return passes
}

// Fraction interpolates the last two completed simulation states. Rendering
// retains one simulation pass of latency rather than predicting collisions.
func (c FrameClock) Fraction() float64 {
	return float64(c.remainder) / float64(c.DisplayRate)
}
