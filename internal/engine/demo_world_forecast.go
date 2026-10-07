package engine

// forecastOpeningGuard guards the third opening and post-merchant corridor
// through isolated World.Step callbacks. It keeps the practiced action unless
// a short exact lookahead predicts damage from a turn, birth or weapon callback.
func (p *PresentationPilot) forecastOpeningGuard(w *World, planned Input) Input {
	opening := w.Checkpoint.ScrollY > 3408 && w.ThirdMiddle == nil
	corridor := w.ThirdMiddle != nil && w.ThirdMiddle.Defeated && w.PendingExitDrops == 0 && !w.ShopReady && w.ScrollY > 208
	if w.Level.Number != 3 || (!opening && !corridor) || w.Ready || !w.PlayerAlive {
		return planned
	}
	forecast := &p.forecast
	pal := p.PALRefreshes
	if pal <= 0 {
		pal = 3
	}
	// Keep the practiced action unless the real callback lookahead predicts damage.
	if err := forecast.Load(w); err != nil {
		return planned
	}
	for pass := 0; pass < 6; pass++ {
		for range pal {
			forecast.AdvancePALTick()
		}
		r, err := forecast.Advance(planned)
		if err != nil {
			return planned
		}
		if r.Shield < w.Equipment.Shield || !r.Alive {
			break
		}
		if r.Boundary != ForecastRunning || pass == 5 {
			return planned
		}
	}
	best := planned
	bestAlive, bestShield, bestProgress, bestDistance := false, -1, 1000000, 1000000
	for _, motion := range demoDirections {
		if err := forecast.Load(w); err != nil {
			return planned
		}
		input := planned
		input.Motion = motion
		var r ForecastResult
		for pass := 0; pass < 6; pass++ {
			for range pal {
				forecast.AdvancePALTick()
			}
			var err error
			r, err = forecast.Advance(input)
			if err != nil {
				return planned
			}
			if r.Boundary != ForecastRunning {
				break
			}
		}
		end := forecast.State()
		distance := absDemo(end.Player.X-w.Player.X) + absDemo(end.Player.Y-120)
		progress := end.ScrollY
		if r.Alive && !bestAlive || r.Alive == bestAlive && (r.Shield > bestShield || r.Shield == bestShield && (distance < bestDistance || distance == bestDistance && progress < bestProgress)) {
			best, bestAlive, bestShield, bestProgress, bestDistance = input, r.Alive, r.Shield, progress, distance
		}
	}
	return best
}
