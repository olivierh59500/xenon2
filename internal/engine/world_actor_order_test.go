package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestWorldWeaponCollisionUsesPhysicalListOrder(t *testing.T) {
	w := testWorld(t)
	makeActor := func(tag int) *WorldActor {
		actor := &WorldActor{Active: true, ActorList: "moving", Health: 3, Collision: CollisionRect{Left: 10, Top: 10, Right: 20, Bottom: 20}, part: &visualassets.ActorPart{ResourceTag: tag, DamageMode: "individual"}}
		if err := w.bindWorldActor(actor); err != nil {
			t.Fatal(err)
		}
		return actor
	}
	older, newer := makeActor(200), makeActor(204)
	w.Actors = []*WorldActor{older, newer}
	if !w.weaponHitPoint(15, 15, 1) || newer.Health != 2 || older.Health != 3 {
		t.Fatal("storage slice order replaced native physical-list collision order")
	}
	context := w.weaponContext(Input{}, false)
	if len(context.Targets) != 2 || context.Targets[0].ID != newer.ID || context.Targets[1].ID != older.ID {
		t.Fatal("homing candidate order differs from physical moving list")
	}
}
