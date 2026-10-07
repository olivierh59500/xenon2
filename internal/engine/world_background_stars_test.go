package engine

import "testing"

func TestInitialStarPassesPreserveWorldLogicAndBackdropOffset(t *testing.T) {
	w := testWorld(t)
	random := w.RandomState()
	for range 144 {
		random.Next()
	}
	w.ResetBackgroundStars()
	if w.RandomState() != random {
		t.Fatal("restoration changed native draw count")
	}
	initial := *w.BackgroundStars
	frame, scroll, offset := w.Frame, w.ScrollY, w.BackgroundY
	w.PrimeBackgroundStars(2)
	if w.Frame != frame || w.ScrollY != scroll || w.BackgroundY != offset || w.RandomState() != random {
		t.Fatal("initial drawing passes advanced combat, camera or random stream")
	}
	initial.Advance(w.ScrollDelta)
	initial.Advance(w.ScrollDelta)
	if *w.BackgroundStars != initial {
		t.Fatal("initial drawing passes did not advance both original star buffers")
	}
}
