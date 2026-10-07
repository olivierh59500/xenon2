package engine

import "testing"

func TestFirstStagePilotPreferencesRespectExplicitConfiguration(t *testing.T) {
	w := testWorld(t)
	w.FirstMiddle = &FirstMiddleState{}
	w.ScrollY = 3300
	player, equipment, random := w.Player, w.Equipment, w.RandomState()
	selected := firstStageConfig(w, DemoPilotConfig{})
	if selected.TargetY != 100 || selected.SafetyMargin != 12 || selected.FireReleasePeriod != 2 || !selected.DisableBonuses {
		t.Fatalf("defense-stream preferences differ: %+v", selected)
	}
	w.FirstMiddle.Crossed = true
	if selected = firstStageConfig(w, DemoPilotConfig{}); selected.TargetY != 70 || !selected.DisableBonuses {
		t.Fatal("second-section preferences did not follow the real crossing")
	}
	explicit := DemoPilotConfig{TargetY: 40, DisableBonuses: true}
	if selected = firstStageConfig(w, explicit); selected != explicit {
		t.Fatal("first-stage defaults replaced an explicit pilot configuration")
	}
	if w.Player != player || w.Equipment != equipment || w.RandomState() != random {
		t.Fatal("pilot preferences modified live gameplay state")
	}
	w.Level.Number = 2
	if selected = firstStageConfig(w, DemoPilotConfig{}); selected != (DemoPilotConfig{}) {
		t.Fatal("first-stage defaults leaked into another stage")
	}
}
