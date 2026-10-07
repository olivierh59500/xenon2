package engine

import "testing"

func TestSixtyDrawsPreserveTwentyFiveSimulationPasses(t *testing.T) {
	c := NewFrameClock(25, 60)
	passes := 0
	for second := range 120 {
		for range 60 {
			passes += c.Advance()
			if f := c.Fraction(); f < 0 || f >= 1 {
				t.Fatalf("interpolation fraction %f outside [0,1)", f)
			}
		}
		if passes != (second+1)*25 {
			t.Fatalf("second %d accumulated %d passes", second+1, passes)
		}
	}
}
