package engine

import "testing"

func TestDemoPilotUsesCommandsWithoutMutatingWorld(t *testing.T) {
	w := testWorld(t)
	pilot := DemoPilot{}
	player, equipment, random, frame, pool := w.Player, w.Equipment, w.RandomState(), w.Frame, *w.Pool
	first := pilot.NormalInput(w)
	for range 20 {
		if got := pilot.NormalInput(w); got != first {
			t.Fatal("same world state produced different commands")
		}
	}
	if w.Player != player || w.Equipment != equipment || w.RandomState() != random || w.Frame != frame || *w.Pool != pool {
		t.Fatal("pilot changed game state instead of producing ordinary input")
	}
	w.Ready = true
	if input := pilot.NormalInput(w); !input.Fire || input.Motion != (MotionInput{}) {
		t.Fatal("READY admission was not an ordinary fire command")
	}
	w.Ready = false
	w.blockedFireUntilRelease = true
	if pilot.NormalInput(w).Fire {
		t.Fatal("pilot did not release the trigger after READY")
	}
}

func TestDemoPilotSecondOpeningCheckpointOptional(t *testing.T) {
	data := playableOriginalWorldData(t, 2)
	s, err := NewSession(data, 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	pilot := DemoPilot{}
	for pass := 0; pass < 700; pass++ {
		w := s.ActiveWorld()
		for range 3 {
			w.AdvancePALTick()
		}
		if _, err := s.Advance(pilot.NormalInput(w)); err != nil {
			t.Fatal(err)
		}
		w = s.ActiveWorld()
		if w.Checkpoint.ScrollY <= 4032 {
			if pass+1 != 578 || w.Checkpoint.ScrollY != 4032 || w.Equipment.Lives != 3 || w.Equipment.Shield != 27 || w.ContinueCredits != 2 || w.LevelFinished {
				t.Fatalf("verified opening policy changed: pass%d checkpoint%d lives%d shield%d credits%d", pass+1, w.Checkpoint.ScrollY, w.Equipment.Lives, w.Equipment.Shield, w.ContinueCredits)
			}
			t.Logf("Reached level2 checkpoint4032 through%d ordinary public commands with all3 ships", pass+1)
			return
		}
	}
	t.Fatal("bounded pilot opening did not reach checkpoint4032")
}
