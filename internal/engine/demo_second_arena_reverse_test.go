package engine

import "testing"

// An isolated original-map arena checks callback ordering, not a campaign win.
// The actor renews its reverse bound after movement and before camera scrolling.
func TestSecondFinalMotionForecastRenewsOriginalReverseBoundOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	w.secondMiddleReleased, w.secondDefenseRemaining = true, 0
	w.Ready, w.MaterializationFrames = false, 0
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 0, 0, 16, 16
	w.Player.X, w.Player.Y, w.Player.SpeedTier = 280, 176, 2
	w.Equipment.SpeedTier = 2
	w.Rewind = NewTerrainRewind(0, 280, 176)
	w.cursor = EncounterCursor{MovingHighWater: 0, FixedHighWater: 0}
	if w.Coverage.Touches(280, 176, 0, *w.Level.PlayerStencil) {
		t.Fatal("original reverse-bound fixture is blocked")
	}
	prediction := newDemoMotionForecast(w)
	for pass := range 48 {
		input := MotionInput{Down: true}
		if !prediction.advance(w, input) {
			t.Fatalf("clear original reverse route rejected at pass%d", pass)
		}
		if err := w.Step(Input{Motion: input}); err != nil {
			t.Fatal(err)
		}
		if w.Player != prediction.player || w.ScrollY != prediction.scroll.Y || w.MaximumScrollY != prediction.scroll.Maximum || w.Rewind != prediction.rewind {
			t.Fatalf("source arena ordering differs at pass%d: actual C%d max%d P%+v, forecast%+v", pass, w.ScrollY, w.MaximumScrollY, w.Player, prediction)
		}
	}
	if w.ScrollY <= 16 || demoScrollMaximum(w, w.ScrollY, w.MaximumScrollY) != 288 {
		t.Fatal("forecast still treats the old camera buffer as the arena reverse limit")
	}
}
