package engine

// FrameClock separates the original simulation cadence from display updates.
// Integer accumulation avoids drift, including fractional PAL-derived rates.
type FrameClock struct {
	SimulationRate int
	DisplayRate    int
	remainder      int
	denominator    int
}

func NewFrameClock(simulationRate, displayRate int) FrameClock {
	if simulationRate <= 0 || displayRate <= 0 {
		panic("frame rates must be positive")
	}
	return NewRationalFrameClock(simulationRate, 1, displayRate)
}

// NewRationalFrameClock schedules numerator/denominator simulation passes per
// second without rounding rates such as fifty PAL refreshes divided by three.
func NewRationalFrameClock(numerator, denominator, displayRate int) FrameClock {
	if numerator <= 0 || denominator <= 0 || displayRate <= 0 {
		panic("frame rates must be positive")
	}
	return FrameClock{SimulationRate: numerator, DisplayRate: displayRate, denominator: denominator}
}

// Advance returns the number of simulation passes due on this display update.
func (c *FrameClock) Advance() int {
	c.remainder += c.SimulationRate
	divisor := c.DisplayRate * max(1, c.denominator)
	passes := c.remainder / divisor
	c.remainder %= divisor
	return passes
}

// Fraction interpolates the last two completed simulation states. Rendering
// retains one simulation pass of latency rather than predicting collisions.
func (c FrameClock) Fraction() float64 {
	return float64(c.remainder) / float64(c.DisplayRate*max(1, c.denominator))
}
