package engine

import "testing"

// This arranges an original post-middle checkpoint boundary, not a campaign
// replay. Terrain, enemies, equipment, ship health and collision stay unchanged;
// traversal after admission uses ordinary public controls and PAL ticks only.
func TestDemoSecondCorridorProactiveLeftExitOptional(t *testing.T) {
	session, err := NewSession(playableOriginalWorldData(t, 2), 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := session.ActiveWorld()
	w.secondMiddleReleased, w.secondDefenseRemaining = true, 0
	w.MinimumScrollY = 0
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 1232, 168
	w.RestartCheckpoint()
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("arranged original checkpoint starts in solid terrain")
	}
	pilot := DemoPilot{}
	for pass := 0; pass < 1100; pass++ {
		w = session.ActiveWorld()
		if w.ScrollY < 800 {
			if pass != 909 || w.Equipment.Lives != 1 || w.Equipment.Shield != 7 || w.Cheats.Enabled() {
				t.Fatalf("corridor outcome pass%d ships%d shield%d", pass, w.Equipment.Lives, w.Equipment.Shield)
			}
			t.Logf("Proactive corridor reached camera%d after%d ordinary commands: lives%d shield%d xy%d,%d", w.ScrollY, pass, w.Equipment.Lives, w.Equipment.Shield, w.Player.X, w.Player.Y)
			return
		}
		if w.GameOver {
			t.Fatal("corridor traversal exhausted its original ships")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		input := pilot.NormalInput(w)
		if _, err = session.Advance(input); err != nil {
			t.Fatal(err)
		}
		w = session.ActiveWorld()

	}
	w = session.ActiveWorld()
	t.Fatalf("bounded corridor did not cross: camera%d xy%d,%d lives%d shield%d", w.ScrollY, w.Player.X, w.Player.Y, w.Equipment.Lives, w.Equipment.Shield)
}
