package engine

import "testing"

func launcherResiduePass(t *testing.T, w *World, pulse, smallWeapons bool) {
	t.Helper()
	w.releaseDeadPoolEntries(ActorPoolEquipment)
	context := w.weaponContext(Input{}, pulse)
	context.SkipSmallWeapons = !smallWeapons
	if err := w.Weapons.AdvanceEquipment(context); err != nil {
		t.Fatal(err)
	}
	if err := w.advancePooledProjectiles(Input{}); err != nil {
		t.Fatal(err)
	}
	w.Weapons.Compact()
}

func expiredLaserResidueSlot(t *testing.T, w *World) int {
	t.Helper()
	w.Equipment.ApplyItem(ItemLaser)
	w.Equipment.ApplyItem(ItemMissileLauncher)
	launcherResiduePass(t, w, true, false)
	slot, id := NoActorSlot, 0
	for _, p := range w.Weapons.projectiles {
		if p.Render.Kind == "laser" {
			slot, id = p.Binding.Slot, p.Render.ID
		}
	}
	if slot == NoActorSlot {
		t.Fatal("ordinary trigger did not emit the original laser")
	}
	launcherResiduePass(t, w, false, false)
	// Selling the completed mount prevents another beam on the next trigger;
	// the already emitted beam retains its physical owner and normal lifetime.
	if _, err := (ShopRules{Level: 1}).Sell(&w.Equipment, &w.Money, SaleMount0); err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 16 && w.Pool.Slot(slot).ResourceTag != 4; pass++ {
		launcherResiduePass(t, w, false, false)
	}
	physical := w.Pool.Slot(slot)
	if physical.EntityID != id || physical.ResourceTag != 4 || physical.Residue.Counter != 65 {
		t.Fatal("original laser expiry did not leave its native length counter")
	}
	launcherResiduePass(t, w, false, false)
	if physical.allocated || w.Pool.FreeFirst() != slot {
		t.Fatal("expired beam did not return its physical slot")
	}
	return slot
}

func TestOriginalLauncherCounterSurvivesFlamerSlotReuseOptional(t *testing.T) {
	for _, test := range []struct {
		name, kind string
		counter    int16
		small      bool
	}{{"launcher", "launcher-missile", 65, false}, {"basic-shot", "small-shot", 0, true}} {
		t.Run(test.name, func(t *testing.T) {
			w, err := NewWorld(originalWorldData(t, 1))
			if err != nil {
				t.Fatal(err)
			}
			w.MaterializationFrames = 0
			// Use real equipment and its source projectile traversal, without
			// advancing encounters during this isolated callback comparison.
			slot := expiredLaserResidueSlot(t, w)
			launcherResiduePass(t, w, true, test.small)
			if !test.small {
				launcherResiduePass(t, w, false, false)
			}
			id := 0
			for _, p := range w.Weapons.projectiles {
				if p.Binding.Slot == slot && p.Render.Kind == test.kind {
					id = p.Render.ID
				}
			}
			if id == 0 {
				t.Fatal("ordinary weapon emission did not reuse the expired beam slot")
			}
			// Launcher emission at 0x41da–0x421c leaves +0x28 untouched;
			// the basic-shot constructor deliberately clears it at 0x61d8.
			if got := w.Pool.Slot(slot).Residue.Counter; got != test.counter {
				t.Errorf("%s emission counter %d, want %d", test.kind, got, test.counter)
			}
			for pass := 0; pass < 40 && w.Pool.Slot(slot).ResourceTag != 4; pass++ {
				launcherResiduePass(t, w, false, false)
				if got := w.Pool.Slot(slot).Residue.Counter; got != test.counter {
					t.Errorf("%s update changed retained counter to %d", test.kind, got)
					break
				}
			}
			// Finish expiry even when the before-fix counter assertion failed,
			// so the later owner's observable audio behavior is also compared.
			for pass := 0; pass < 40 && w.Pool.Slot(slot).ResourceTag != 4; pass++ {
				launcherResiduePass(t, w, false, false)
			}
			if physical := w.Pool.Slot(slot); physical.EntityID != id || physical.ResourceTag != 4 {
				t.Fatal("emitted projectile did not expire in its original slot")
			}
			launcherResiduePass(t, w, false, false)
			if w.Pool.Slot(slot).allocated || w.Pool.FreeFirst() != slot {
				t.Fatal("expired projectile did not expose the same slot for the flamer")
			}
			w.Equipment.ApplyItem(ItemFlamer)
			context := w.weaponContext(Input{}, false)
			if err := w.Weapons.SynchronizeEquipment(context); err != nil {
				t.Fatal(err)
			}
			mount := &w.Weapons.mounts[0]
			if mount.Binding.Slot != slot || mount.Binding.EntityID == id {
				t.Fatal("flamer did not reuse the retired projectile's physical slot")
			}
			wantStop := test.counter != 0
			if mount.FlamerSound.Counter != test.counter || mount.FlamerSound.Started != wantStop {
				t.Errorf("flamer lost inherited %s counter: %+v", test.kind, mount.FlamerSound)
			}
			// Flamer initialization retains +0x28, and an unheld update at
			// 0x3c68–0x3c74 stops effects once before clearing a nonzero counter.
			if err := w.Weapons.AdvanceEquipment(context); err != nil {
				t.Fatal(err)
			}
			if w.StopEffectsRequested != wantStop || mount.FlamerSound.Counter != 0 {
				t.Fatalf("flamer release after %s: stop=%v counter=%d, want stop=%v counter=0", test.kind, w.StopEffectsRequested, mount.FlamerSound.Counter, wantStop)
			}
		})
	}
}
