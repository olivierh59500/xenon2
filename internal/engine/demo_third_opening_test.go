package engine

import "testing"

func TestDemoThirdOpeningConfigurationIsTemporary(t *testing.T) {
	w := testWorld(t)
	w.Level.Number, w.ScrollY, w.Checkpoint.ScrollY = 3, 4608, 4608
	original := DemoPilotConfig{TargetX: 80, TargetY: 166, SafetyMargin: 24, FireReleasePeriod: 4}
	player, equipment, random := w.Player, w.Equipment, w.RandomState()
	selected := thirdOpeningConfig(w, original)
	if selected.TargetX != 250 || selected.TargetY != 120 || selected.Lookahead != 5 || selected.SafetyMargin != 12 || selected.FireReleasePeriod != 2 || !selected.DisableBonuses {
		t.Fatalf("opening selection differs: %+v", selected)
	}
	if original.TargetX != 80 || original.TargetY != 166 || original.DisableBonuses || w.Player != player || w.Equipment != equipment || w.RandomState() != random {
		t.Fatal("opening selection modified caller configuration or game state")
	}
	w.Checkpoint.ScrollY = 4032
	if next := thirdOpeningConfig(w, original); next != original {
		t.Fatal("temporary opening configuration survived its checkpoint")
	}
	w.Checkpoint.ScrollY = 4608
	original.DisableOpeningRoute = true
	if next := thirdOpeningConfig(w, original); next != original {
		t.Fatal("explicit opening-route disable was ignored")
	}
}

// This starts an unmodified real-resource session. Every checkpoint and shield
// change comes from normal public commands under three PAL refreshes per pass.
func TestDemoThirdOpeningFirstCheckpointWithoutShipLossOptional(t *testing.T) {
	session, err := NewSession(playableOriginalWorldData(t, 3), 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	pilot := DemoPilot{}
	for pass := 0; pass < 700; pass++ {
		w := session.ActiveWorld()
		if w.GameOver || w.Equipment.Lives != 3 || w.ContinueCredits != 2 {
			t.Fatal("opening consumed a ship or continue")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		input := pilot.NormalInput(w)
		if _, err = session.Advance(input); err != nil {
			t.Fatal(err)
		}
		w = session.ActiveWorld()
		if w.Checkpoint.ScrollY <= 4032 {
			if pass+1 != 578 || w.Checkpoint.ScrollY != 4032 || w.ScrollY != 4031 || w.Equipment.Lives != 3 || w.Equipment.Shield != 3 || w.ContinueCredits != 2 || w.LevelFinished {
				t.Fatalf("opening differs: pass%d checkpoint%d camera%d lives%d shield%d credits%d", pass+1, w.Checkpoint.ScrollY, w.ScrollY, w.Equipment.Lives, w.Equipment.Shield, w.ContinueCredits)
			}
			t.Log("Reached checkpoint 4032 in 578 ordinary commands with all 3 ships and both continues")
			return
		}
	}
	t.Fatal("bounded ordinary opening did not reach the first checkpoint")
}
