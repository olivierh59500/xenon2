package engine

import (
	"reflect"
	"testing"
)

// This arranges the last original checkpoint before the final fixed selector.
// Subsequent activation, all eye/core damage and coin handling use the normal
// encounter stream and public commands, without health or equipment grants.
func TestDemoFourthFinalBoundaryThroughOrdinaryShotsOptional(t *testing.T) {
	s, err := NewSession(playableOriginalWorldData(t, 4), 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := s.ActiveWorld()
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 176, 152
	w.MinimumScrollY = 0
	w.RestartCheckpoint()
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("original final checkpoint geometry is not clear")
	}
	pilot := DemoPilot{}
	continues := 0
	for pass := 0; pass < 5000; pass++ {
		w = s.ActiveWorld()
		if w.GameOver {
			if !s.AcceptContinue() {
				if w.FourthFinal != nil {
					t.Fatalf("credits exhausted with core%d eyes%d,%d", w.FourthFinal.Parts[0].Health, w.FourthFinal.Parts[1].Health, w.FourthFinal.Parts[2].Health)
				}
				t.Fatal("credits exhausted before guardian activation")
			}
			continues++
			w = s.ActiveWorld()
		}
		if w.ShopReady {
			if w.FourthFinal == nil || !w.FourthFinal.Defeated || !w.LevelFinished || !w.ExitReady || w.PendingExitDrops != 0 {
				t.Fatal("shop reached without real final guardian victory")
			}
			// Native inherited coin directions shorten this fixture's reward
			// collection by ten passes compared with zero-initialized headings.
			if pass != 1775 || w.Equipment.Lives != 2 || w.Equipment.Shield != 23 || continues != 0 || w.FourthFinal.Parts[0].Health != 0 || w.FourthFinal.Parts[1].Health != 0 || w.FourthFinal.Parts[2].Health != 0 {
				t.Fatalf("verified ordinary final outcome changed: pass%d ships%d shield%d continues%d health%v", pass, w.Equipment.Lives, w.Equipment.Shield, continues, w.FourthFinal.Parts[:3])
			}
			t.Logf("Final shop at%d commands lives%d shield%d continues%d", pass, w.Equipment.Lives, w.Equipment.Shield, continues)
			return
		}
		for range 3 {
			w.AdvancePALTick()
		}
		input := pilot.NormalInput(w)
		if _, err = s.Advance(input); err != nil {
			t.Fatal(err)
		}
		if pass%500 == 0 {
			w = s.ActiveWorld()
			if w.FourthFinal != nil {
				t.Logf("pass%d camera%d xy%d,%d lives%d shield%d core%d eyes%d,%d", pass, w.ScrollY, w.Player.X, w.Player.Y, w.Equipment.Lives, w.Equipment.Shield, w.FourthFinal.Parts[0].Health, w.FourthFinal.Parts[1].Health, w.FourthFinal.Parts[2].Health)
			}
		}
	}
	t.Fatal("bounded ordinary shooting did not defeat the final guardian")
}

func TestFourthFinalProbePredictionDoesNotMutateWorld(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 16, 0, 16, 16
	if err := w.activateFourthGuardian(true); err != nil {
		t.Fatal(err)
	}
	player, equipment, random, state, frame, camera := w.Player, w.Equipment, w.RandomState(), *w.FourthFinal, w.Frame, w.ScrollY
	actors := make([]WorldActor, len(w.Actors))
	for i, a := range w.Actors {
		actors[i] = *a
	}
	p := DemoPilot{}
	if _, handled := p.FourthFinalInput(w); !handled {
		t.Fatal("source final arena was not admitted to the probe")
	}
	if w.Player != player || w.Equipment != equipment || w.RandomState() != random || *w.FourthFinal != state || w.Frame != frame || w.ScrollY != camera {
		t.Fatal("prediction changed game state rather than issuing controls")
	}
	for i, a := range w.Actors {
		if !reflect.DeepEqual(*a, actors[i]) {
			t.Fatal("prediction changed an actor")
		}
	}
}

func TestFourthFinalProbeHonorsAdmissionAndMissingResources(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY = 16
	p := DemoPilot{}
	w.blockedFireUntilRelease = true
	if input, handled := p.FourthFinalInput(w); !handled || input.Fire {
		t.Fatal("final admission bypassed the original trigger release")
	}
	if err := w.activateFourthGuardian(true); err != nil {
		t.Fatal(err)
	}
	if input, handled := p.FourthFinalInput(w); !handled || input.Fire {
		t.Fatal("active final policy bypassed trigger release")
	}
	art := w.fourthFinalArt
	w.fourthFinalArt = nil
	if _, handled := p.FourthFinalInput(w); handled {
		t.Fatal("policy used an unavailable final resource")
	}
	w.fourthFinalArt = art
	w.Level.Paths = nil
	if _, handled := p.FourthFinalInput(w); handled {
		t.Fatal("policy used an unavailable motion table")
	}
}

func BenchmarkFourthFinalProbeInput(b *testing.B) {
	w, err := NewWorld(playableOriginalWorldData(b, 4))
	if err != nil {
		b.Fatal(err)
	}
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 0, 0, 16, 16
	if err := w.activateFourthGuardian(true); err != nil {
		b.Fatal(err)
	}
	p := DemoPilot{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.FourthFinalInput(w)
	}
}
