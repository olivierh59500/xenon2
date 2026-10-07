package engine

import "testing"

// This explicitly arranges the original checkpoint before the fixed selector.
// The middle guardian is subsequently created by the real encounter stream.
// Health, equipment, map and damage remain original; only public commands and
// the two ordinary continue credits may be used after admission.
func TestDemoThirdMiddleShopThroughOrdinaryControlsOptional(t *testing.T) {
	session, err := NewSession(playableOriginalWorldData(t, 3), 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := session.ActiveWorld()
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 2832, 168
	w.RestartCheckpoint()
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("original middle checkpoint geometry is not clear")
	}
	pilot := DemoPilot{}
	continues := 0
	checkedReadOnly := false
	for pass := 0; pass < 3000; pass++ {
		w = session.ActiveWorld()
		if w.GameOver {
			if !session.AcceptContinue() {
				t.Fatal("ordinary middle policy exhausted both continue credits")
			}
			continues++
			w = session.ActiveWorld()
		}
		if w.ShopReady {
			if !checkedReadOnly || pass != 1082 || w.ThirdMiddle == nil || !w.ThirdMiddle.Defeated || w.ThirdMiddle.EyeHealth != [2]uint16{} || w.Equipment.Lives != 2 || continues != 2 || w.LevelFinished || w.ExitReady || w.PendingExitDrops != 0 {
				t.Fatalf("middle outcome changed: pass%d lives%d shield%d continues%d pending%d", pass, w.Equipment.Lives, w.Equipment.Shield, continues, w.PendingExitDrops)
			}
			t.Logf("Genuine middle shop after %d ordinary commands: both eyes destroyed, 2 lives and 2 legal continues", pass)
			return
		}
		for range 3 {
			w.AdvancePALTick()
		}
		input, handled := pilot.ThirdMiddleInput(w)
		if handled && !checkedReadOnly {
			middle, player, equipment, random, frame, pool := *w.ThirdMiddle, w.Player, w.Equipment, w.RandomState(), w.Frame, *w.Pool
			next, again := pilot.ThirdMiddleInput(w)
			if !again || next != input || *w.ThirdMiddle != middle || w.Player != player || w.Equipment != equipment || w.RandomState() != random || w.Frame != frame || *w.Pool != pool {
				t.Fatal("articulated prediction mutated the game or produced inconsistent controls")
			}
			checkedReadOnly = true
		}
		if !handled {
			input = pilot.NormalInput(w)
		}
		if _, err = session.Advance(input); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("bounded ordinary controls did not defeat both eyes and reach the middle shop")
}

// This measures steady controller work, not graphical FPS or mobile performance.
func BenchmarkDemoThirdMiddleInputOriginal(b *testing.B) {
	w, err := NewWorld(playableOriginalWorldData(b, 3))
	if err != nil {
		b.Fatal(err)
	}
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 2832, 168
	w.RestartCheckpoint()
	pilot := DemoPilot{}
	for pass := 0; pass < 80; pass++ {
		input, handled := pilot.ThirdMiddleInput(w)
		if !handled {
			input = pilot.NormalInput(w)
		}
		if err := w.Step(input); err != nil {
			b.Fatal(err)
		}
	}
	if w.ThirdMiddle == nil {
		b.Fatal("source fixed encounter did not create the middle guardian")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		pilot.ThirdMiddleInput(w)
	}
}
