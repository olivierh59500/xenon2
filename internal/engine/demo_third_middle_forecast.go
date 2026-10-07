package engine

// forecastThirdMiddleInput compares source-calibrated moves through a complete
// arm extension and retreat, then applies only the first ordinary command.
func (p *PresentationPilot) forecastThirdMiddleInput(w *World, fallback Input) Input {
	if w.Level.Number != 3 || w.ThirdMiddle == nil || w.ThirdMiddle.Defeated || !w.PlayerAlive || w.Ready {
		return fallback
	}
	pal := p.PALRefreshes
	if pal <= 0 {
		pal = 3
	}
	forecast := &p.forecast
	best := fallback
	bestAlive, bestShield, bestHP, bestDistance := false, -1, 1000000, 1000000
	policy := &p.middleForecastPolicy
	for index, motion := range demoDirections {
		if err := forecast.Load(w); err != nil {
			return fallback
		}
		if index == 0 {
			if policy.navigation == nil {
				policy.navigation = &demoNavigation{}
			}
			policy.navigation.refresh(forecast.State())
			// This is only the continuation proposal's terrain cache. Every
			// candidate still executes exact mutable-map contact in World.Step.
			policy.middleTerrainFrozen = true
		}
		var result ForecastResult
		for pass := 0; pass < 36; pass++ {
			state := forecast.State()
			input := fallback
			if pass < 3 {
				input.Motion = motion
			} else {
				input = policy.NormalInput(state)
			}
			input.Fire = !state.blockedFireUntilRelease && state.Dive.Phase == 0 && presentationShotOpportunityForMotion(state, input.Motion)
			for range pal {
				forecast.AdvancePALTick()
			}
			var err error
			result, err = forecast.Advance(input)
			if err != nil {
				return fallback
			}
			if result.Boundary != ForecastRunning {
				break
			}
		}
		end := forecast.State()
		hp := max(0, int(int16(end.ThirdMiddle.EyeHealth[0]))) + max(0, int(int16(end.ThirdMiddle.EyeHealth[1])))
		target := 3
		if int16(end.ThirdMiddle.EyeHealth[0]) <= 0 {
			target = 4
		}
		x := end.ThirdMiddle.Parts[target].X
		distance := absDemo(end.Player.X-x) + absDemo(end.Player.Y-176)
		if result.Alive && !bestAlive || result.Alive == bestAlive && (result.Shield > bestShield || result.Shield == bestShield && (hp < bestHP || hp == bestHP && distance < bestDistance)) {
			best = fallback
			best.Motion = motion
			bestAlive, bestShield, bestHP, bestDistance = result.Alive, result.Shield, hp, distance
		}
	}
	best.Fire = !w.blockedFireUntilRelease && w.Dive.Phase == 0 && presentationShotOpportunityForMotion(w, best.Motion)
	return best
}
