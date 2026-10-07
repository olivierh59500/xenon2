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

func TestPALThreeRefreshClockPreservesFractionalCadence(t *testing.T) {
	logic := NewRationalFrameClock(50, 3, 60)
	pal := NewFrameClock(50, 60)
	passes, ticks := 0, 0
	for update := 0; update < 60*3600; update++ {
		passes += logic.Advance()
		ticks += pal.Advance()
		if want := (update + 1) * 50 / (3 * 60); passes != want {
			t.Fatalf("update %d has %d logic passes, want %d", update, passes, want)
		}
		if want := (update + 1) * 50 / 60; ticks != want {
			t.Fatalf("logic profile changed PAL tick %d", update)
		}
		if fraction := logic.Fraction(); fraction < 0 || fraction >= 1 {
			t.Fatalf("interpolation fraction %f outside range", fraction)
		}
	}
	if passes != 60000 || ticks != 180000 {
		t.Fatalf("one-hour clocks drifted: logic %d PAL %d", passes, ticks)
	}
}
