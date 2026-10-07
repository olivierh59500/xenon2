package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestOriginalThirdMiddleClearsSoldCannonEmitterOptional(t *testing.T) {
	data := originalWorldData(t, 3)
	var selector visualassets.FixedEncounter
	for _, record := range data.Encounters.Fixed {
		if record.EnemyKind == 3 {
			selector = record
			break
		}
	}
	if selector.EnemyKind != 3 {
		t.Fatal("original middle selector is missing")
	}
	world := func() *World {
		w, err := NewWorld(data)
		if err != nil {
			t.Fatal(err)
		}
		w.MaterializationFrames = 0
		w.Level.Encounters = &visualassets.Encounters{Fixed: []visualassets.FixedEncounter{selector}}
		w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = selector.TriggerY, selector.TriggerY+16, selector.TriggerY+16
		w.cursor = RestartEncounterCursor(w.ScrollY)
		return w
	}
	fresh, reused := world(), world()
	reused.Money = 4000
	shop := ShopRules{Level: 3, StockLimit: 4000}
	if _, err := shop.Buy(&reused.Equipment, &reused.Money, ItemCannon); err != nil {
		t.Fatal(err)
	}
	if err := reused.Weapons.SynchronizeEquipment(reused.weaponContext(Input{}, false)); err != nil {
		t.Fatal(err)
	}
	mount := reused.Weapons.mounts[1]
	slot := mount.Binding.Slot
	retained := reused.Pool.Slot(slot).Residue
	if mount.Item != ItemCannon || mount.Binding.EntityID == 0 || retained.EmitterClock != 0xffff {
		t.Fatal("ordinary cannon purchase did not establish its physical ffff emitter")
	}
	if _, err := shop.Sell(&reused.Equipment, &reused.Money, SaleMount0); err != nil {
		t.Fatal(err)
	}
	if err := reused.Weapons.SynchronizeEquipment(reused.weaponContext(Input{}, false)); err != nil {
		t.Fatal(err)
	}
	reused.releaseDeadPoolEntries(ActorPoolProjectile)
	reused.releaseDeadPoolEntries(ActorPoolEquipment)
	if reused.Pool.Slot(slot).allocated || reused.Pool.FreeFirst() != slot || reused.Pool.Slot(slot).Residue.EmitterClock != 0xffff {
		t.Fatal("ordinary sale did not release the unchanged cannon emitter")
	}
	for _, w := range []*World{fresh, reused} {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		if w.ThirdMiddle == nil {
			t.Fatal("real encounter traversal did not construct the middle guardian")
		}
	}
	head := reused.thirdMiddleActors[0]
	if head.Binding.Slot != slot || head.ID == mount.Binding.EntityID {
		t.Fatal("actual middle head did not reuse the sold cannon slot")
	}
	if head.Health != int(retained.Health) || head.Binding.Residue.Health != retained.Health || reused.ThirdMiddle.EyeHealth != [2]uint16{20, 20} {
		t.Fatal("middle birth changed its retained slot health or original shared eye health")
	}
	// Native 0x55e68 (4268005e) clears both bytes in the common
	// seventeen-member constructor, before any head or arm callback runs.
	for index, actor := range reused.thirdMiddleActors {
		if got := reused.Pool.Slot(actor.Binding.Slot).Residue.EmitterClock; got != 0 {
			t.Errorf("middle member %d retained native-cleared emitter %04x", index, got)
		}
	}
	if reused.ThirdMiddle.ResidualFireRate != 0 {
		t.Error("middle head retained the sold cannon firing rate")
	}
	freshID, reusedID := fresh.nextActorID, reused.nextActorID
	for range 4 {
		if err := fresh.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		if err := reused.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	// Arms legitimately consume RNG. Their draws must match the fresh source
	// constructor; a stale rate adds an aimed shot and changes that stream.
	if reused.nextActorID != reusedID || fresh.nextActorID != freshID || reused.ThirdMiddle.Parts[0].FireAccumulator != 80 || reused.RandomState() != fresh.RandomState() {
		t.Fatalf("native four head callbacks: projectile births%d fire%d randomDiff%v; want zero births, fire80 and fresh-constructor RNG", reused.nextActorID-reusedID, reused.ThirdMiddle.Parts[0].FireAccumulator, reused.RandomState() != fresh.RandomState())
	}
}
