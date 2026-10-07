package engine

import "testing"

func TestWorldShadowsFollowShipAndPreserveSlotOrdinal(t *testing.T) {
	w := testWorld(t)
	w.Player = PlayerMotionState{X: 200, Y: 120, Inertia: -6}
	w.shipTrail[0] = PlayerMotionState{X: 20, Y: 30}
	if err := w.advancePlayerShadows(Input{Motion: MotionInput{Down: true}}); err != nil {
		t.Fatal(err)
	}
	for i, state := range w.Shadows {
		counter := 3 - i
		if state.Counter != counter || state.Y != 120 || state.Visible != (counter >= 2) {
			t.Fatalf("shadow order/direction: %+v", state)
		}
		if state.X != 198 && state.X != 202 {
			t.Fatalf("shadow followed trail: %+v", state)
		}
		residue := w.Pool.Slot(w.poolShadows[i].Slot).Residue
		if residue.VerticalFraction != uint16(counter) || residue.Counter != 0 || residue.X != int16(state.X) || residue.Y != 120 {
			t.Fatalf("shadow changed unrelated slot residue: %+v", residue)
		}
	}
	if err := w.advancePlayerShadows(Input{Motion: MotionInput{Up: true}}); err != nil {
		t.Fatal(err)
	}
	for _, state := range w.Shadows {
		if state.Visible != (state.Counter < 2) {
			t.Fatal("reverse thrust silhouettes did not switch")
		}
	}
}
