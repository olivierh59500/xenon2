package engine

import "testing"

func TestOriginalThirdChainMarkerRetainsBulletStateThroughSlotReuseOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	// An ordinary diagonal bullet produces both fractions through movement;
	// no initial-memory fixture supplies the marker's retained values.
	w.spawnEnemyShot(80, 30, EnemyShot{Direction: 1, Speed: 6})
	first := w.Projectiles[0]
	for pass := 0; pass < 40 && first.Active; pass++ {
		if err := w.advancePooledProjectiles(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	slot := first.Binding.Slot
	want := w.Pool.Slot(slot).Residue
	if first.Active || want.XFraction == 0 || want.YFraction == 0 {
		t.Fatal("ordinary bullet did not establish fractional expiry residue")
	}
	if err := w.advancePooledProjectiles(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.Pool.FreeFirst() != slot {
		t.Fatal("expired bullet slot was not the free head")
	}
	found := false
	for _, record := range w.Level.Encounters.Fixed {
		if record.EnemyKind == 1 && record.Variant == 0 {
			w.ScrollY = record.TriggerY
			w.spawnThirdChain(record)
			found = true
			break
		}
	}
	if !found {
		t.Fatal("original chain encounter is missing")
	}
	marker := w.poolActors[slot]
	if marker == nil || !marker.thirdChainSentinel || marker.part.ResourceTag != 188 {
		t.Fatal("chain marker did not reuse the expired bullet's physical slot")
	}
	markerID := marker.ID
	if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
	// Native 0x5605a–0x560ae creates aggregate markers. Their callbacks
	// 0x3130/0x37b2 do not replace gameplay residue with display coordinates.
	checkResidue := func(stage string) {
		got := w.Pool.Slot(slot).Residue
		// Marker link construction is separate from retained gameplay words.
		got.OwnerSlot, got.LeaderSlot, got.FollowingSlot = want.OwnerSlot, want.LeaderSlot, want.FollowingSlot
		if got != want {
			t.Errorf("%s marker overwrote retained bullet state: got %+v want %+v", stage, got, want)
		}
	}
	checkResidue("live")
	for pass := 0; pass < 300 && marker.Active; pass++ {
		if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if marker.Active {
		t.Fatal("original chain did not leave its natural 232-pixel boundary")
	}
	if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
	if w.Pool.Slot(slot).allocated {
		t.Fatal("expired marker did not release its physical slot")
	}
	checkResidue("retired")
	// Ordinary allocations consume the chain's released slots in source order
	// until the former marker is reused; no slot is manually released or moved.
	var second *WorldProjectile
	for count := 0; count < 12; count++ {
		w.spawnEnemyShot(100, 90, EnemyShot{Direction: 1, Speed: 6})
		if w.Projectiles[0].Binding.Slot == slot {
			second = w.Projectiles[0]
			break
		}
	}
	if second == nil || second.ID == markerID {
		t.Fatal("ordinary allocator did not replace the retired marker in the same slot")
	}
	if err := w.advancePooledProjectiles(Input{}); err != nil {
		t.Fatal(err)
	}
	// Native heading one adds 11585,-11585 times speed six, shifted twice.
	// The constructor replaces whole coordinate words while retaining fractions.
	wantX := (int32(100)<<16 | int32(want.XFraction)) + 11585*6*4
	wantY := (int32(90)<<16 | int32(want.YFraction)) - 11585*6*4 + int32(w.ScrollDelta)<<16
	if second.Motion.X != wantX || second.Motion.Y != wantY || second.X != float64(wantX>>16) || second.Y != float64(wantY>>16) {
		t.Fatalf("marker reuse changed the next native bullet: got (%v,%v) fixed (%08x,%08x), want (%d,%d) fixed (%08x,%08x)", second.X, second.Y, uint32(second.Motion.X), uint32(second.Motion.Y), wantX>>16, wantY>>16, uint32(wantX), uint32(wantY))
	}
}
