package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestPickupRetainsInheritedDirectionBeforeUpdateAndImmediateReclaim(t *testing.T) {
	w := testWorld(t)
	w.commonAnimations["pickup-14"] = visualassets.NamedActorAnimation{ResourceTag: 104, Ending: "hold", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "gift"}}}}
	slot := w.Pool.FreeFirst()
	w.Pool.Slot(slot).Residue = ActorResidue{X: 11, Y: 23, XFraction: 0x1234, YFraction: 0x5678, Direction: 0x1234, Counter: 65, VerticalFraction: 71, EmitterClock: 0xa55a}
	equipment, random, frame := w.Equipment, w.RandomState(), w.Frame
	w.spawnPickup(14, 96, 88)
	pickup := w.Collectibles[0]
	if pickup.Motion.Direction != 0x1234 {
		t.Fatal("pickup creation discarded the previous owner's direction word")
	}
	for w.Pool.FreeFirst() != NoActorSlot {
		if _, err := w.reserveWorldActor(220, ActorPoolMoving, false); err != nil {
			t.Fatal(err)
		}
	}
	replacement := &WorldActor{Active: true, ActorList: "moving", part: &visualassets.ActorPart{ResourceTag: 200}}
	if err := w.bindWorldActor(replacement); err != nil {
		t.Fatal(err)
	}
	r := replacement.Binding.Residue
	if replacement.Binding.Slot != slot || pickup.Active || r.X != 96 || r.Y != 88 || r.Counter != 7 || r.Direction != 0x1234 || r.VerticalFraction != 14 || r.XFraction != 0x1234 || r.YFraction != 0x5678 || r.EmitterClock != 0xa55a {
		t.Fatalf("replacement did not inherit initialized and untouched pickup words: %+v", r)
	}
	if w.Equipment != equipment || w.RandomState() != random || w.Frame != frame || w.Money != 0 || w.Score != 0 {
		t.Fatal("immediate reclamation collected the unvisited pickup or advanced gameplay")
	}
}

func TestCashMovementMasksItsLookupWithoutErasingTheDirectionWord(t *testing.T) {
	cash := CashMotion{X: 80, Y: 20, Mode: 7, Direction: 0xf123}
	if !cash.Advance() || cash.X != 82 || cash.Y != 22 || cash.Mode != 8 || cash.Direction != 0xf123 {
		t.Fatalf("movement erased the stored word instead of masking its lookup: %+v", cash)
	}
	if !cash.Advance() || cash.Direction > 7 {
		t.Fatal("the next native center-aim pass did not replace the full word")
	}
	cash = CashMotion{X: 80, Y: 20, Mode: -1, Direction: 0xffff}
	if !cash.Advance() || cash.Direction != 0 || cash.Mode != 0 {
		t.Fatal("spiral increment did not retain native word wrap and heading mask")
	}
}

func TestCollectedRewardsUseOriginalEffectVoiceOptional(t *testing.T) {
	data := originalWorldData(t, 1)
	rows := 0
	nativeCombatRows(t, "combat-pickup-trace.csv", func(v []int64) {
		w, err := NewWorld(data)
		if err != nil {
			t.Fatal(err)
		}
		// The fixture isolates the real collection callback and its sound route.
		// Original selector results are checked independently in combat tests.
		w.playerCollision = CollisionRect{Left: -1024, Top: -1024, Right: 1024, Bottom: 1024}
		w.spawnPickup(int(v[0]), 96, 88)
		w.SoundRequests[2] = "sampled-effect-03"
		item := w.Collectibles[0]
		w.advanceCollectible(item)
		if item.Active || w.SoundRequests[1] != nativeRewardSound(v[30]) || w.SoundRequests[2] != "sampled-effect-03" {
			t.Fatalf("reward%d collection did not match native sound%d on its own voice: active%v requests%v", v[0], v[30], item.Active, w.SoundRequests)
		}
		rows++
	})
	if rows != 19 {
		t.Fatalf("incomplete collection audio coverage: %d selectors", rows)
	}
}
