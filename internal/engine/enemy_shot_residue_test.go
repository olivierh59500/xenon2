package engine

import "testing"

func TestOriginalEnemyShotRetainsExpiredSlotFractionsOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	w.spawnEnemyShot(80, 30, EnemyShot{Direction: 1, Speed: 6})
	first := w.Projectiles[0]
	for pass := 0; pass < 40 && first.Active; pass++ {
		if err := w.advancePooledProjectiles(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if first.Active || w.Pool.Slot(first.Binding.Slot).ResourceTag != 4 {
		t.Fatal("ordinary diagonal bullet did not leave the original playfield")
	}
	slot := first.Binding.Slot
	residue := w.Pool.Slot(slot).Residue
	if residue.XFraction == 0 || residue.YFraction == 0 {
		t.Fatal("the source movement did not produce fractional slot residue")
	}
	if err := w.advancePooledProjectiles(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.Pool.Slot(slot).allocated || w.Pool.FreeFirst() != slot {
		t.Fatal("the expired bullet did not return its physical slot")
	}
	w.spawnEnemyShot(100, 90, EnemyShot{Direction: 1, Speed: 6})
	second := w.Projectiles[0]
	if second.Binding.Slot != slot || second.ID == first.ID {
		t.Fatal("replacement bullet did not reuse the expired physical slot")
	}
	// Native 0x31ce/0x31d2 write whole X/Y words only. The retained low
	// words participate in the full fixed-point additions at 0x391c–0x394c.
	wantX := int32(100)<<16 | int32(residue.XFraction)
	wantY := int32(90)<<16 | int32(residue.YFraction)
	if second.Motion.X != wantX || second.Motion.Y != wantY {
		t.Errorf("native constructor retains fractions %04x,%04x: got %08x,%08x want %08x,%08x", residue.XFraction, residue.YFraction, uint32(second.Motion.X), uint32(second.Motion.Y), uint32(wantX), uint32(wantY))
	}
	// Heading one uses the native table pair 11585,-11585, multiplied by
	// speed six and shifted left twice; scrolling changes only whole Y.
	wantX += 11585 * 6 * 4
	wantY += -11585*6*4 + int32(w.ScrollDelta)<<16
	if err := w.advancePooledProjectiles(Input{}); err != nil {
		t.Fatal(err)
	}
	if second.X != float64(wantX>>16) || second.Y != float64(wantY>>16) || second.Motion.X != wantX || second.Motion.Y != wantY {
		t.Fatalf("reused bullet lost its native trajectory: got (%v,%v) fixed (%08x,%08x), want (%d,%d) fixed (%08x,%08x)", second.X, second.Y, uint32(second.Motion.X), uint32(second.Motion.Y), wantX>>16, wantY>>16, uint32(wantX), uint32(wantY))
	}
	stored := w.Pool.Slot(slot).Residue
	if stored.XFraction != uint16(wantX) || stored.YFraction != uint16(wantY) || stored.Health != residue.Health || stored.Counter != residue.Counter || stored.EmitterClock != residue.EmitterClock {
		t.Fatal("bullet update changed retained health, counter, emitter state or fractional motion")
	}
}
