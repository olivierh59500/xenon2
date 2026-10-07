package app

import "xenon2/internal/engine"

// newLogicClock preserves the measured A500 early-game cadence by default.
// An explicit two-refresh setting still selects the source maximum. Display
// updates and the audio/palette clocks remain independent.
func (g *Game) newLogicClock() engine.FrameClock {
	refreshes := g.Config.LogicPALRefreshes
	if refreshes == 0 {
		refreshes = 3
	}
	return engine.NewRationalFrameClock(50, refreshes, 60)
}
