package engine

import (
	"slices"
	"testing"

	"xenon2/internal/visualassets"
)

func TestSupernovaTraversalNativeTraceOptional(t *testing.T) {
	nativeCombatRows(t, "supernova-traversal-trace.csv", func(v []int64) {
		w := testWorld(t)
		var actors [7]*WorldActor
		for i, tag := range []int{int(v[2]), 208, 208, 80, 84, 200, 4} {
			binding, err := w.reserveWorldActor(int16(tag), ActorPoolMoving, true)
			if err != nil {
				t.Fatal(err)
			}
			actor := &WorldActor{ID: binding.EntityID, Binding: binding, Active: true, ActorList: "moving", part: &visualassets.ActorPart{ResourceTag: tag}}
			actors[i], w.poolActors[binding.Slot] = actor, actor
			w.Actors = append(w.Actors, actor)
			if v[1] != 0 && i < 3 {
				slot := w.Pool.Slot(binding.Slot)
				slot.Linked = true
				slot.Residue.OwnerSlot = actors[0].Binding.Slot
			}
		}
		var hits []int
		w.forEachSupernovaTarget(func(actor *WorldActor) { hits = append(hits, slices.Index(actors[:], actor)) })
		for i := range 7 {
			actual := -1
			if i < len(hits) {
				actual = hits[i]
			}
			if actual != int(v[3+i]) {
				t.Fatalf("supernova case%d order=%v expected%v", v[0], hits, v[3:10])
			}
		}
		// Keep the original list construction but prevent ordinary damage from
		// adding effects, so projectile filtering can be compared independently.
		for _, actor := range actors {
			actor.Active = false
		}
		var shots [6]ActorPoolBinding
		for i, tag := range []int{20, 16, 240, 8, 20, 4} {
			binding, err := w.reserveWorldActor(int16(tag), ActorPoolProjectile, true)
			if err != nil {
				t.Fatal(err)
			}
			shots[i] = binding
		}
		for i := range w.WaveBonuses.Entries {
			w.WaveBonuses.Entries[i] = WaveBonusEntry{ID: uint16(10 + i), Remaining: uint16(20 + i)}
		}
		w.finishSupernova()
		for i, binding := range shots {
			if w.Pool.Slot(binding.Slot).ResourceTag != int16(v[10+i]) {
				t.Fatalf("supernova case%d projectile%d tag=%d expected%d", v[0], i, w.Pool.Slot(binding.Slot).ResourceTag, v[10+i])
			}
		}
		if v[16] != 0 || w.WaveBonuses.Entries != [8]WaveBonusEntry{} {
			t.Fatal("supernova did not clear every wave bonus bucket")
		}
	})
}

func TestWorldWaveConstructorSetsNativeLinkedOwnerReferences(t *testing.T) {
	w := testWorld(t)
	part := visualassets.ActorPart{ResourceTag: 208, MotionMode: "path", Linked: true}
	w.kinds[2] = &visualassets.WaveActor{Kind: 2, Parts: []visualassets.ActorPart{part, part, part}}
	w.paths[2] = &visualassets.Path{ID: 2, Commands: []visualassets.PathCommand{{Kind: "pause", Duration: 40}, {Kind: "end"}}}
	if err := w.spawnWave(visualassets.Wave{EnemyKind: 2, PathID: 2, Count: 2, MotionBudget: 1}); err != nil {
		t.Fatal(err)
	}
	w.countSourceMovingActors()
	if w.MovingEnemyCount != 2 {
		t.Fatalf("linked group count=%d expected2", w.MovingEnemyCount)
	}
	for _, actor := range w.Actors {
		slot := w.Pool.Slot(actor.Binding.Slot)
		owner := actor
		if actor.leader != nil {
			owner = actor.leader
		}
		if !slot.Linked || slot.Residue.OwnerSlot != owner.Binding.Slot {
			t.Fatal("linked wave constructor lost its physical owner reference")
		}
	}
}
