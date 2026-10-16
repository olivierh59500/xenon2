package engine

// thirdOpeningConfig selects the verified first-checkpoint controller window.
// It returns a temporary configuration; the pilot, world, gear and shared
// random state are unchanged, and bonus collection resumes after the marker.
func thirdOpeningConfig(w *World, config DemoPilotConfig) DemoPilotConfig {
	if w == nil || w.Level.Number != 3 || w.ScrollY < 4032 || w.Checkpoint.ScrollY <= 4032 || config.DisableOpeningRoute {
		return config
	}
	config.TargetX, config.TargetY = 250, 120
	config.Lookahead, config.SafetyMargin, config.FireReleasePeriod = 5, 12, 2
	config.DisableBonuses = true
	return config
}
