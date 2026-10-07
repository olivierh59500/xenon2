package engine

import "testing"

func TestFourthWorldStageSpawnerWindowAndSourceFamiliesOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 3200, 4607, 4607
	w.Frame = 16
	before := w.RandomState()
	expected := before
	family := int(expected.Next() & 1)
	fire := uint8(expected.Next())
	if err := w.advanceFourthStage(); err != nil {
		t.Fatal(err)
	}
	if len(w.Actors) != 1 || w.Actors[0].fourthFalling == nil || w.Actors[0].fourthFalling.Variant != family || w.Actors[0].fourthFalling.PrimaryClock != fire || w.RandomState() != expected || w.MaximumScrollY != 3568 {
		t.Fatal("periodic source spawner did not consume exactly two random calls after allocation")
	}
	actor := w.Actors[0]
	slot := actor.Binding.Slot
	w.Frame = 17
	if err := w.advanceFourthStage(); err != nil {
		t.Fatal(err)
	}
	if len(w.Actors) != 1 || w.RandomState() != expected {
		t.Fatal("spawner ran outside its sixteen-pass cadence")
	}
	w.ScrollDelta = 1
	if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
	if actor.Y != -32+float64(1+[]int{w.Level.FixedSprites.FourthStage.HatchSpeed, w.Level.FixedSprites.FourthStage.GunSpeed}[family]) || actor.Binding.Slot != slot {
		t.Fatal("stage-born actor did not update on the current traversal")
	}
	w.InvulnerableFrames = 100000
	for pass := 0; pass < 500; pass++ {
		w.Frame = uint64(pass + 32)
		w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 3200, 3568, 3568
		w.ScrollDelta = 0
		if err := w.advanceFourthStage(); err != nil {
			t.Fatal(err)
		}
		if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
			t.Fatal(err)
		}
		if err := w.advancePooledProjectiles(Input{}); err != nil {
			t.Fatal(err)
		}
		w.compactActors()
	}
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
	if w.Pool.FreeFirst() == NoActorSlot {
		t.Fatal("falling families failed to expire within their source window")
	}
}

func TestFourthWorldPodConvertsInPlaceAndSpawnsLoopingChildOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	w.spawnFourthPod(0, 32, 40)
	actor := w.Actors[0]
	slot, id := actor.Binding.Slot, actor.ID
	for pass := 0; actor.fourthPod != nil && pass < 30; pass++ {
		if err := w.advancePooledProjectiles(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if actor.ID != id || actor.Binding.Slot != slot || actor.fourthPod != nil || actor.part.ResourceTag != 12 || actor.Atlas != "common" {
		t.Fatal("pod did not become its finite explosion in the same physical slot")
	}
	frames := 0
	for _, frame := range w.commonAnimations["explosion-small"].Animation.Frames {
		frames += frame.Duration
	}
	for range frames {
		if err := w.advancePooledProjectiles(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if actor.Active || w.Pool.Slot(slot).ResourceTag != 4 {
		t.Fatal("converted pod explosion restarted after its final frame")
	}
	if err := w.advancePooledProjectiles(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.Pool.Slot(slot).allocated {
		t.Fatal("converted pod explosion retained its physical slot")
	}
	var child *WorldActor
	for _, candidate := range w.Actors {
		if candidate.fourthChild != nil {
			child = candidate
		}
	}
	if child == nil || child.part.ResourceTag != 248 || child.Health != w.Level.FixedSprites.FourthStage.PodChildHealth || child.WaveToken == 0 || w.WaveBonuses.Entries[0].ID != child.WaveToken || w.WaveBonuses.Entries[0].Remaining != 1 {
		t.Fatal("pod did not create its separately damageable source child")
	}
	for range 600 {
		if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if !child.Active || child.fourthChild.Motion.PathID != w.Level.FixedSprites.FourthStage.PodChildPaths[child.fourthChild.PathIndex].ID {
		t.Fatal("child stopped instead of rerouting after its source path ended")
	}
	for _, bucket := range w.WaveBonuses.Entries {
		if bucket.ID == child.WaveToken {
			t.Fatal("rerouted child retained the original path's bonus bucket")
		}
	}
}
