package engine

// firstStageConfig keeps the ordinary ship below incoming formations and gives
// terrain navigation priority over loose bonuses inside the defense streams.
// Explicit controller configurations retain their own targets and preferences.
func firstStageConfig(w *World, config DemoPilotConfig) DemoPilotConfig {
	if w == nil || w.Level.Number != 1 || w.FirstMiddle == nil || config != (DemoPilotConfig{}) {
		return config
	}
	config.TargetY, config.SafetyMargin, config.FireReleasePeriod = 100, 12, 2
	if w.ScrollY < 3456 {
		config.DisableBonuses = true
	}
	if w.FirstMiddle.Crossed {
		config.TargetY, config.DisableBonuses = 70, true
	}
	return config
}
