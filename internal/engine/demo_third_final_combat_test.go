package engine

import "testing"

// This original final checkpoint starts with a deliberately small energy
// reserve. It isolates boss survival; complete frontend tests earn their own
// equipment, energy, shop purchases and checkpoint admission.
func TestExpertThirdFinalSurvivesLowEnergyWithoutContinuesOptional(t *testing.T) {
	session, err := NewSession(playableOriginalWorldData(t, 3), 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := session.ActiveWorld()
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 176, 152
	w.MinimumScrollY = 0
	w.RestartCheckpoint()
	w.Equipment.Shield = 7
	p := PresentationPilot{PALRefreshes: 3}
	lives, credits := w.Equipment.Lives, w.ContinueCredits
	for pass := range 3000 {
		input := p.NormalInput(w)
		for range 3 {
			w.AdvancePALTick()
		}
		if _, err := session.Advance(input); err != nil {
			t.Fatal(err)
		}
		if !w.PlayerAlive || w.GameOver || w.Equipment.Lives != lives || w.ContinueCredits != credits || w.Cheats.Enabled() {
			t.Fatalf("ordinary low-energy duel lost its ship at pass%d: HP%d boss%d", pass, w.Equipment.Shield, w.ThirdFinal.Health)
		}
		if w.ShopReady {
			if !w.ThirdFinal.Defeated || w.ThirdFinal.Health != 0 || !w.LevelFinished || !w.ExitReady || w.PendingExitDrops != 0 {
				t.Fatal("duel bypassed its original guardian and reward gates")
			}
			t.Logf("Ordinary low-energy final defeated in%d passes, shield%d, ships%d, credits%d", pass+1, w.Equipment.Shield, lives, credits)
			return
		}
	}
	t.Fatal("bounded expert duel did not defeat the original final guardian")
}

func TestExpertThirdFinalForecastKeepsLiveStateOptional(t *testing.T) {
	w := thirdFinalAdmittedSourceFixture(t, 16)
	w.Player.X, w.Player.Y = 143, 176
	p := PresentationPilot{PALRefreshes: 3}
	before := forecastIsolationDigest(w)
	input := p.forecastThirdFinalInput(w, Input{})
	if forecastIsolationDigest(w) != before {
		t.Fatal("boss anticipation changed live state")
	}
	ordinary := false
	for _, motion := range demoDirections {
		ordinary = ordinary || input.Motion == motion
	}
	if !ordinary || input.Dive {
		t.Fatal("boss anticipation emitted an unsupported control")
	}
}
