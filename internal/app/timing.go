package app

import "xenon2/internal/engine"

// newLogicClock retains the source two-refresh maximum by default. The
// measured A500 profile uses three PAL refreshes, independently of 60Hz drawing
// and 50Hz audio/fades. Presentation and shop clocks retain the source maximum
// until their scene-dependent processing cadence is measured more completely.
func (g *Game) newLogicClock() engine.FrameClock {
	refreshes := g.Config.LogicPALRefreshes
	if refreshes == 0 {
		refreshes = 2
	}
	return engine.NewRationalFrameClock(50, refreshes, 60)
}
