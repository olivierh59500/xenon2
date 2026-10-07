package engine

import "testing"

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
