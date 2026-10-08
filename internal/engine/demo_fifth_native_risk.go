package engine

// demoFifthNativeRisk owns reusable simulation storage for the destructible
// guided missiles. Their eighth-pass turns depend on the candidate ship motion;
// extrapolating the last displacement cannot predict their contact callbacks.
type demoFifthNativeRisk struct {
	forecast WorldForecast
}

func fifthNativeRiskEligible(w *World) bool {
	if w == nil || w.Level.Number != 5 || forecastBoundary(w) != ForecastRunning || w.Dive.Phase != 0 || w.ScreenClearFrames != 0 || w.stepContinuation.active || w.Level.FixedSprites == nil || w.Level.FixedSprites.Projectile == nil || len(w.Level.FixedSprites.Projectile.HeadingAnimations) != 8 {
		return false
	}
	for _, actor := range w.Actors {
		if actor.Active && actor.fixedAiming != nil {
			return true
		}
	}
	return false
}

// losses retains each actual shield debit, including native collision ordering,
// animation bounds, missile expiry, scroll changes and intervening callbacks.
// Existing shots still advance; hypothetical new player fire is excluded.
func (p *demoFifthNativeRisk) losses(w *World, motion MotionInput, horizon, pal int) (damage [8]int, supported bool) {
	if p == nil || horizon < 1 || horizon > len(damage) || !fifthNativeRiskEligible(w) || p.forecast.Load(w) != nil {
		return damage, false
	}
	pal = thirdMiddlePALRefreshes(pal)
	shield := w.Equipment.Shield
	for pass := range horizon {
		for range pal {
			p.forecast.AdvancePALTick()
		}
		result, err := p.forecast.Advance(Input{Motion: motion})
		if err != nil {
			return [8]int{}, false
		}
		damage[pass] = max(0, shield-result.Shield)
		shield = result.Shield
		if result.Boundary != ForecastRunning {
			break
		}
	}
	return damage, true
}

func (p *DemoPilot) prepareFifthNativeRisk(w *World) {
	if !fifthNativeRiskEligible(w) {
		return
	}
	if p.fifthRisk == nil {
		p.fifthRisk = &demoFifthNativeRisk{}
	}
}
