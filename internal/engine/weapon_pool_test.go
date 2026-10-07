package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestWeaponRuntimeSharedPoolRetirementAndFlameResidue(t *testing.T) {
	pool := NewActorPool()
	for i := range ActorPoolCapacity {
		allocation, err := pool.Allocate()
		if err != nil {
			t.Fatal(err)
		}
		if err := pool.AttachTail(allocation.Slot, ActorPoolMoving, i+1, 200); err != nil {
			t.Fatal(err)
		}
		pool.Slot(i).Residue.HorizontalDriftRemainder = uint16(0xa123 + i)
	}
	runtime, err := NewWeaponRuntime(&visualassets.SpriteAtlas{})
	if err != nil {
		t.Fatal(err)
	}
	nextID := ActorPoolCapacity
	equipment := NewEquipment()
	random, expected := NewRandomState(), NewRandomState()
	ctx := WeaponContext{Equipment: &equipment, ShipX: 160, ShipY: 100, NextRandom: random.Next}
	ctx.ReserveActor = func(tag int16, list ActorPoolList, tail bool) (ActorPoolBinding, error) {
		allocation, err := pool.Allocate()
		if err != nil {
			return ActorPoolBinding{}, err
		}
		if allocation.Stolen {
			runtime.DropActor(allocation.PreviousEntityID)
		}
		nextID++
		if tail {
			err = pool.AttachTail(allocation.Slot, list, nextID, tag)
		} else {
			err = pool.AttachHead(allocation.Slot, list, nextID, tag)
		}
		return ActorPoolBinding{Slot: allocation.Slot, EntityID: nextID, Residue: pool.Slot(allocation.Slot).Residue}, err
	}
	ctx.StoreActorResidue = func(binding ActorPoolBinding) {
		if n := pool.Slot(binding.Slot); n != nil && n.EntityID == binding.EntityID {
			n.Residue = binding.Residue
		}
	}
	ctx.RetireActor = func(binding ActorPoolBinding) {
		if n := pool.Slot(binding.Slot); n != nil && n.EntityID == binding.EntityID {
			if err := pool.MarkDead(binding.Slot); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := runtime.SynchronizeEquipment(ctx); err != nil {
		t.Fatal(err)
	}
	old := runtime.mounts[0].Binding.Slot
	equipment.ApplyItem(ItemFlamer)
	ctx.Held = true
	if err := runtime.AdvanceEquipment(ctx); err != nil {
		t.Fatal(err)
	}
	if pool.Slot(old).ResourceTag != 4 || pool.Slot(old).list != ActorPoolEquipment || pool.FreeFirst() != NoActorSlot {
		t.Fatal("equipment replacement freed its protected slot prematurely")
	}
	if len(runtime.projectiles) != 2 {
		t.Fatal("flamer did not emit into the shared pool")
	}
	for _, p := range runtime.projectiles {
		expected.Next()
		drift := int32(int8(uint8(expected.Next()))) << 10
		drift += int32(uint16(0xa125) >> 6)
		if p.Flame.VelocityX != drift || p.Binding.Slot != 2 {
			t.Fatalf("particle lost the allocated slot's drift: %+v expected=%d", p, drift)
		}
	}
	if runtime.projectiles[0].Render.Active || !runtime.projectiles[1].Render.Active {
		t.Fatal("full pool did not replace the first flame at the projectile head")
	}
	if random != expected {
		t.Fatal("pool construction changed random consumption")
	}
}

func TestHomingTargetFollowsReusedPhysicalSlot(t *testing.T) {
	state := HomingMissileState{X: 100, Y: 100, Direction: 0, TargetID: 17, TargetSlotIdentity: 5}
	targets := []WeaponTarget{{ID: 29, SlotIdentity: 5, Active: true, ResourceTag: 200, Bounds: CollisionRect{Left: 100, Top: 20, Right: 120, Bottom: 40}}}
	if !state.Advance(targets, func() uint32 { t.Fatal("reused target unnecessarily drew random selection"); return 0 }, false) {
		t.Fatal("reused physical target was lost")
	}
	if state.TargetSlotIdentity != 5 {
		t.Fatal("physical reference changed with creation ID")
	}
}
