package engine

import (
	"strconv"
	"testing"
)

func TestOriginalFourthPodClearsExpiredLaserCounterBeforeFlamerReuseOptional(t *testing.T) {
	for side := range 2 {
		t.Run(strconv.Itoa(side), func(t *testing.T) {
			testOriginalFourthPodCounterReuse(t, side)
		})
	}
}

func testOriginalFourthPodCounterReuse(t *testing.T, side int) {
	t.Helper()
	w, err := NewWorld(originalWorldData(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	w.MaterializationFrames = 0
	w.Equipment.ApplyItem(ItemLaser)
	context := w.weaponContext(Input{}, true)
	context.SkipSmallWeapons = true
	if err := w.Weapons.AdvanceEquipment(context); err != nil {
		t.Fatal(err)
	}
	slot, beamID := NoActorSlot, 0
	for _, p := range w.Weapons.projectiles {
		if p.Render.Kind == "laser" {
			slot, beamID = p.Binding.Slot, p.Render.ID
		}
	}
	if slot == NoActorSlot {
		t.Fatal("ordinary equipment callback did not create the original beam")
	}
	for pass := 0; pass < 16 && w.Pool.Slot(slot).ResourceTag != 4; pass++ {
		if err := w.advancePooledProjectiles(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if physical := w.Pool.Slot(slot); physical.EntityID != beamID || physical.ResourceTag != 4 || physical.Residue.Counter != 65 {
		t.Fatal("ordinary laser expiry did not establish native counter65 residue")
	}
	if err := w.advancePooledProjectiles(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.Pool.Slot(slot).allocated || w.Pool.FreeFirst() != slot {
		t.Fatal("expired beam did not expose its slot for the source pod")
	}
	w.spawnFourthPod(side, w.Level.FixedSprites.FourthStage.Variants[side*2].PodX, 40)
	pod := w.Actors[0]
	if pod.Binding.Slot != slot || pod.ID == beamID || pod.fourthPod == nil {
		t.Fatal("original capsule did not replace the expired beam in its physical slot")
	}
	// Native 0x552f2 clears +0x28 at birth. The subsequent pod and common
	// explosion callbacks retain it, instead of resetting unrelated slot state.
	if got := w.Pool.Slot(slot).Residue.Counter; got != 0 {
		t.Errorf("native capsule constructor clears counter65: got %d", got)
	}
	for pass := 0; pass < 40 && pod.fourthPod != nil; pass++ {
		if err := w.advancePooledProjectiles(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if pod.fourthPod != nil || pod.Atlas != "common" || pod.part.ResourceTag != 12 {
		t.Fatal("capsule did not perform its source in-place explosion conversion")
	}
	frames := 0
	for _, frame := range w.commonAnimations["explosion-small"].Animation.Frames {
		frames += frame.Duration
	}
	for range frames + 1 {
		if err := w.advancePooledProjectiles(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if pod.Active || w.Pool.Slot(slot).allocated || w.Pool.FreeFirst() != slot {
		t.Fatal("converted finite explosion did not release the original capsule slot")
	}
	if got := w.Pool.Slot(slot).Residue.Counter; got != 0 {
		t.Errorf("capsule/explosion lifetime retained the wrong counter: %d", got)
	}
	w.Equipment.ApplyItem(ItemFlamer)
	context = w.weaponContext(Input{}, false)
	if err := w.Weapons.SynchronizeEquipment(context); err != nil {
		t.Fatal(err)
	}
	mount := &w.Weapons.mounts[0]
	if mount.Binding.Slot != slot || mount.Binding.EntityID == pod.ID {
		t.Fatal("flamer did not reuse the retired capsule's physical slot")
	}
	if mount.FlamerSound.Counter != 0 || mount.FlamerSound.Started {
		t.Errorf("flamer inherited a counter the native capsule cleared: %+v", mount.FlamerSound)
	}
	if err := w.Weapons.AdvanceEquipment(context); err != nil {
		t.Fatal(err)
	}
	if w.StopEffectsRequested {
		t.Fatal("native zero-counter flamer release must not request effect cleanup")
	}
}
