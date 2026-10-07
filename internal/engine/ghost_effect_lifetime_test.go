package engine

import "testing"

// Common explosion creation at 0x31ec and update at 0x57b6 only touch position
// and animation. Unrelated physical values survive for the next constructor.
func TestOriginalExplosionRetainsPhysicalResidueThroughExpiryOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	slot := w.Pool.FreeFirst()
	want := secondResidueFixture()
	want.OwnerSlot, want.LeaderSlot, want.FollowingSlot = 19, 20, 21
	want.X, want.Y = 160, 80
	w.Pool.Slot(slot).Residue = want
	w.spawnSecondNamedExplosion(160, 80, "explosion-small")
	actor := w.Actors[0]
	if actor.Binding.Slot != slot {
		t.Fatal("explosion did not reclaim the expected physical slot")
	}
	for range 7 {
		if err := w.advancePooledProjectiles(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if actor.Active || w.Pool.Slot(slot).allocated {
		t.Fatal("finite explosion did not release its slot")
	}
	if got := w.Pool.Slot(slot).Residue; got != want {
		t.Fatalf("animation-only effect overwrote physical residue: got %+v want %+v", got, want)
	}
	if err := w.activateThirdMiddle(); err != nil {
		t.Fatal(err)
	}
	body := w.thirdMiddleActors[0]
	if body.Binding.Slot != slot || body.Health != int(want.Health) {
		t.Fatal("middle guardian did not inherit the expired explosion slot's health")
	}
	// The explosion retains the word; the later guardian constructor at
	// 0x55e68 explicitly clears it before the first member update.
	if w.ThirdMiddle.ResidualFireRate != 0 || body.Binding.Residue.EmitterClock != 0 || w.Pool.Slot(slot).Residue.EmitterClock != 0 {
		t.Fatal("middle constructor retained the expired explosion slot's cleared emitter")
	}
}

// The original common explosion descriptors are finite named sequences. Birth
// must retain that ending when storing the underlying actor animation.
func TestOriginalCommonExplosionsRetireAfterTheirFinalFrameOptional(t *testing.T) {
	for _, name := range []string{"explosion-small", "explosion-large"} {
		t.Run(name, func(t *testing.T) {
			w, err := NewWorld(playableOriginalWorldData(t, 1))
			if err != nil {
				t.Fatal(err)
			}
			w.Ready = false
			w.Level.Encounters.Moving = nil
			w.Level.Encounters.Fixed = nil
			clip := w.commonAnimations[name]
			if clip.Ending != "remove" || len(clip.Animation.Frames) == 0 {
				t.Fatal("missing original finite explosion")
			}
			w.spawnSecondNamedExplosion(160, 80, name)
			actor := w.Actors[0]
			slot := actor.Binding.Slot
			frames := 0
			for _, frame := range clip.Animation.Frames {
				frames += frame.Duration
			}
			for pass := 0; pass < frames; pass++ {
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if actor.Active || w.Pool.Slot(slot).ResourceTag != 4 {
				t.Fatalf("finite %s remained active after%d passes: ending%q frame%d tag%d", name, frames, actor.animation.Ending, actor.animationState.Frame, w.Pool.Slot(slot).ResourceTag)
			}
			if err := w.Step(Input{}); err != nil {
				t.Fatal(err)
			}
			if w.Pool.Slot(slot).allocated {
				t.Fatal("retired explosion did not return its physical slot")
			}
			for pass := 0; pass < 20; pass++ {
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			for _, remaining := range w.Actors {
				if remaining == actor && remaining.Active {
					t.Fatal("explosion reappeared after retirement")
				}
			}
		})
	}
}
