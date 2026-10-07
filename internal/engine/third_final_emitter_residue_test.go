package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestOriginalThirdFinalWormClearsSoldCannonEmitterOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.MaterializationFrames, w.Money = 0, 4000
	shop := ShopRules{Level: 3, StockLimit: 4000}
	if _, err := shop.Buy(&w.Equipment, &w.Money, ItemCannon); err != nil {
		t.Fatal(err)
	}
	if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
		t.Fatal(err)
	}
	mount := w.Weapons.mounts[1]
	if mount.Item != ItemCannon || mount.Binding.EntityID == 0 {
		t.Fatal("original cannon purchase did not construct its physical mount")
	}
	slot := mount.Binding.Slot
	if got := w.Pool.Slot(slot).Residue.EmitterClock; got != 0xffff {
		t.Fatalf("original cannon constructor did not establish emitter ffff: %04x", got)
	}
	if _, err := shop.Sell(&w.Equipment, &w.Money, SaleMount0); err != nil {
		t.Fatal(err)
	}
	if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
		t.Fatal(err)
	}
	w.releaseDeadPoolEntries(ActorPoolProjectile)
	w.releaseDeadPoolEntries(ActorPoolEquipment)
	if w.Pool.Slot(slot).allocated || w.Pool.FreeFirst() != slot || w.Pool.Slot(slot).Residue.EmitterClock != 0xffff {
		t.Fatal("ordinary cannon sale did not release its unchanged physical emitter")
	}

	// Run the original final-boundary constructor and all actor callbacks through
	// World.Step. Native 0x553b8 clears both emitter bytes for every worm member.
	w.Level.Encounters = &visualassets.Encounters{}
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 160, 0, 192, 192
	w.cursor = RestartEncounterCursor(w.ScrollY)
	random := w.RandomState()
	var head *WorldActor
	for pass := 1; pass <= 4; pass++ {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		if pass == 1 {
			members := 0
			for _, actor := range w.Actors {
				if actor.thirdFinalMember == nil {
					continue
				}
				members++
				if actor.thirdPart.Index == 0 {
					head = actor
				}
				if actor.thirdFinalMember.ResidualFireRate != 0 || w.Pool.Slot(actor.Binding.Slot).Residue.FireRate() != 0 {
					t.Errorf("worm member %d retained a firing rate cleared by its native constructor", actor.thirdPart.Index)
				}
			}
			if members != 11 || head == nil || head.Binding.Slot != slot || head.ID == mount.Binding.EntityID {
				t.Fatal("actual final-boundary worm did not reuse the sold cannon slot for its head")
			}
		}
	}
	// Source head rate 20 accumulates to 80 without a carry. A retained cannon
	// rate adds ordinary aimed fire, creating a projectile and consuming RNG.
	if len(w.Projectiles) != 0 || w.RandomState() != random || head.thirdFinalMember.FireAccumulator != 80 {
		t.Fatalf("first four native head callbacks must emit nothing without RNG: shots=%d fire=%d randomChanged=%v", len(w.Projectiles), head.thirdFinalMember.FireAccumulator, w.RandomState() != random)
	}
	if !w.PlayerAlive || w.Pool.Slot(slot).Residue.EmitterClock != 80<<8 {
		t.Fatal("head callback did not retain the native cleared-rate emitter state")
	}
}
