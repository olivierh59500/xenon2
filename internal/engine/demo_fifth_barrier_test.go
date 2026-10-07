package engine

import "testing"

func TestExpertFifthOpeningClearsOriginalBarrierWithOrdinaryControlsOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 5))
	if err != nil {
		t.Fatal(err)
	}
	pilot := PresentationPilot{PALRefreshes: 3}
	var post *WorldActor
	checked := false
	for pass := 0; pass < 240; pass++ {
		for _, actor := range w.Actors {
			if post == nil && actor.fifthTile != nil && actor.fixedTileArt != nil && actor.fixedTileArt.Kind == 1 && actor.fifthTile.Part == 0 {
				post = actor
			}
		}
		if post != nil && !post.Active && post.Health == 0 {
			if !checked || w.Cheats.Enabled() || w.Equipment.Lives != 3 || w.Equipment.Shield != 39 || !w.PlayerAlive || w.Score < 200 {
				t.Fatal("first barrier did not retain ordinary health, life and damage rules")
			}
			if _, handled := pilot.planner.FifthBarrierInput(w); handled {
				t.Fatal("destroyed barrier kept the ship in its reverse firing hold")
			}
			t.Logf("First original fifth barrier opens at frame%d camera%d with3ships shield%d", w.Frame, w.ScrollY, w.Equipment.Shield)
			return
		}
		if !w.PlayerAlive {
			t.Fatalf("ship died before opening the first barrier: frame%d camera%d xy%d,%d", w.Frame, w.ScrollY, w.Player.X, w.Player.Y)
		}
		if !checked && fifthBarrierTarget(w) != nil {
			before := forecastDigest(w)
			if _, handled := pilot.planner.FifthBarrierInput(w); !handled || forecastDigest(w) != before {
				t.Fatal("post policy rejected its source role or changed live state")
			}
			checked = true
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(pilot.NormalInput(w)); err != nil {
			t.Fatal(err)
		}
	}
	if post != nil {
		t.Fatalf("ordinary expert did not open barrier: health%d camera%d xy%d,%d", post.Health, w.ScrollY, w.Player.X, w.Player.Y)
	}
	t.Fatal("source encounter stream did not create the first barrier")
}
