package engine

import "runtime"

// forecastThirdMiddleInput compares all nine ordinary moves through a complete
// arm extension and retreat, then applies only the first ordinary command.
func (p *PresentationPilot) forecastThirdMiddleInput(w *World, fallback Input) Input {
	if w.Level.Number != 3 || w.ThirdMiddle == nil || w.ThirdMiddle.Defeated || !w.PlayerAlive || w.Ready {
		return fallback
	}
	policy := &p.middleForecastPolicy
	if runtime.GOMAXPROCS(0) <= 1 || policy.Config.DisableBossAlignment || w.GameOver || w.thirdMiddleArt == nil || w.Coverage == nil || w.Level.PlayerStencil == nil {
		return p.forecastThirdMiddleSerial(w, fallback)
	}
	return p.forecastThirdMiddleParallel(w, fallback)
}

func prepareThirdMiddlePolicy(policy *DemoPilot, state *World, camera int) {
	if policy.navigation == nil {
		policy.navigation = &demoNavigation{}
	}
	policy.navigation.refresh(state)
	policy.navigation.cacheTouchWindow(camera)
	// Only the continuation proposal freezes terrain; every predicted
	// World.Step still owns exact mutable-map contact and source callbacks.
	policy.middleTerrainFrozen = true
}

func thirdMiddlePALRefreshes(pal int) int {
	if pal <= 0 {
		return 3
	}
	return pal
}

func (p *PresentationPilot) forecastThirdMiddleSerial(w *World, fallback Input) Input {
	best := fallback
	score := emptyThirdMiddleCandidate()
	for index, motion := range demoDirections {
		if err := p.forecast.Load(w); err != nil {
			return fallback
		}
		if index == 0 {
			prepareThirdMiddlePolicy(&p.middleForecastPolicy, p.forecast.State(), w.ScrollY)
		}
		candidate, err := evaluateThirdMiddleCandidate(&p.forecast, &p.middleForecastPolicy, motion, fallback, thirdMiddlePALRefreshes(p.PALRefreshes), false)
		if err != nil {
			return fallback
		}
		if thirdMiddleCandidateBetter(candidate, score) {
			best, score = fallback, candidate
			best.Motion = motion
		}
	}
	best.Fire = !w.blockedFireUntilRelease && w.Dive.Phase == 0 && presentationShotOpportunityForMotion(w, best.Motion)
	return best
}
