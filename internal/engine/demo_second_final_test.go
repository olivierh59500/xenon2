package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

// This is an explicitly arranged post-middle boundary, not a campaign replay.
// It retains the original map, stencil, basic equipment and guardian health.
// All subsequent terrain clearing and damage must come from public commands.
func TestDemoSecondFinalBarrierWithOrdinaryShotsOptional(t *testing.T) {
	session, err := NewSession(playableOriginalWorldData(t, 2), 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := session.ActiveWorld()
	w.secondMiddleReleased, w.secondDefenseRemaining = true, 0
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 288, 152
	w.MinimumScrollY = 0
	w.RestartCheckpoint()
	pilot := DemoPilot{}
	cleared, continues := 0, 0
	for pass := 0; pass < 1800; pass++ {
		w = session.ActiveWorld()
		if w.GameOver {
			if !session.AcceptContinue() {
				t.Fatalf("credits exhausted; cleared%d HP%d", cleared, w.secondGuardianActor.Health)
			}
			continues++
			w = session.ActiveWorld()
		}
		if w.ShopReady && w.LevelFinished {
			if pass != 1037 || w.secondGuardianActor.Active || w.secondGuardianActor.Health != 0 || !w.ExitReady || w.PendingExitDrops != 0 || w.Equipment.Lives != 1 || w.Equipment.Shield != 23 || cleared != 6 || continues != 0 {
				t.Fatalf("final boundary differs: pass%d HP%d lives%d shield%d cleared%d continues%d", pass, w.secondGuardianActor.Health, w.Equipment.Lives, w.Equipment.Shield, cleared, continues)
			}
			t.Logf("Boundary final shop pass%d HP%d lives%d shield%d cleared%d continues%d", pass, w.secondGuardianActor.Health, w.Equipment.Lives, w.Equipment.Shield, cleared, continues)
			return
		}
		for range 3 {
			w.AdvancePALTick()
		}
		input, handled := pilot.SecondFinalInput(w)
		if !handled {
			input = pilot.NormalInput(w)
		}
		before := append([]bool(nil), w.secondTerrainCells.Intact...)
		if _, err = session.Advance(input); err != nil {
			t.Fatal(err)
		}
		w = session.ActiveWorld()
		for index, intact := range w.secondTerrainCells.Intact {
			if before[index] && !intact {
				cleared++
			}
		}
		if pass%100 == 0 {
			t.Logf("pass%d camera%d xy%d,%d alive%v cleared%d wait%d HP%d", pass, w.ScrollY, w.Player.X, w.Player.Y, w.PlayerAlive, cleared, w.SecondGuardian.WaitTimer, w.secondGuardianActor.Health)
		}
	}
	w = session.ActiveWorld()
	t.Fatalf("bounded final-shop request not reached: cleared%d wait%d HP%d camera%d xy%d,%d", cleared, w.SecondGuardian.WaitTimer, w.secondGuardianActor.Health, w.ScrollY, w.Player.X, w.Player.Y)
}

func TestDemoSecondFinalUsesCommandsAndRetainsActivatedNegativeWait(t *testing.T) {
	w := testWorld(t)
	w.Level.Number, w.ScrollY, w.Player.X, w.Player.Y = 2, 288, 152, 176
	w.secondMiddleReleased = true
	state := NewSecondGuardianState()
	w.SecondGuardian = &state
	w.secondGuardianActor = &WorldActor{Active: true, Health: 75}
	w.secondTerrainCells = NewSecondTerrainCells([]visualassets.GuardianTerrainCell{{Quadrant: 0, X: 192, WorldY: 336}})
	demoPilotCoverage(w)
	pilot := DemoPilot{}
	player, equipment, random, guardian := w.Player, w.Equipment, w.RandomState(), state
	input, handled := pilot.SecondFinalInput(w)
	if !handled || !input.Motion.Right || !input.Motion.Down {
		t.Fatalf("intact barrier not opened by ordinary aiming: %+v", input)
	}
	if w.Player != player || w.Equipment != equipment || w.RandomState() != random || state != guardian || !w.secondTerrainCells.Intact[0] || w.secondGuardianActor.Health != 75 {
		t.Fatal("final policy changed world state instead of returning controls")
	}
	state.MotionRemaining = 1000
	input, handled = pilot.SecondFinalInput(w)
	if !handled || input.Motion.Down {
		t.Fatal("activated negative-wait traversal incorrectly reentered barrier-opening phase")
	}
	w.blockedFireUntilRelease = true
	if input, _ = pilot.SecondFinalInput(w); input.Fire {
		t.Fatal("final policy bypassed READY trigger release")
	}
	w.Ready = true
	if _, handled = pilot.SecondFinalInput(w); handled {
		t.Fatal("final policy took over READY admission")
	}
}
