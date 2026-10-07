package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestWorldMovingRemovalTraversalNativeOptional(t *testing.T) {
	var world *World
	var first, second *WorldActor
	previousCase := -1
	nativeCombatRows(t, "actor-removal-phase-trace.csv", func(v []int64) {
		if int(v[0]) != previousCase {
			world = testWorld(t)
			part := &visualassets.ActorPart{ResourceTag: 208, MotionMode: "path"}
			second = &WorldActor{Active: true, ActorList: "moving", fixed: true, mapY: 100, part: part}
			if err := world.bindWorldActor(second); err != nil {
				t.Fatal(err)
			}
			first = &WorldActor{Active: true, ActorList: "moving", fixed: true, mapY: 100, part: part}
			if v[0] == 0 {
				path := &visualassets.Path{ID: 1, Commands: []visualassets.PathCommand{{Kind: "end"}}}
				motion, err := NewPathMotion(path, PathMotionConfig{Budget: 1})
				if err != nil {
					t.Fatal(err)
				}
				first.fixed, first.path, first.motion = false, path, motion
			} else {
				second.Active = false
				world.storeActorResidue(second)
			}
			if err := world.bindWorldActor(first); err != nil {
				t.Fatal(err)
			}
			world.Actors = []*WorldActor{first, second}
			previousCase = int(v[0])
		}
		if err := world.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
			t.Fatal(err)
		}
		logicalSlot := func(slot int) int {
			if slot == first.Binding.Slot {
				return 1
			}
			if slot == second.Binding.Slot {
				return 2
			}
			return 0
		}
		if int(world.Pool.Slot(first.Binding.Slot).ResourceTag) != int(v[2]) || int(world.Pool.Slot(second.Binding.Slot).ResourceTag) != int(v[3]) || logicalSlot(world.Pool.FreeFirst()) != int(v[4]) || logicalSlot(world.Pool.First(ActorPoolMoving)) != int(v[5]) {
			t.Fatalf("moving source traversal %v: types=%d/%d free=%d head=%d", v, world.Pool.Slot(first.Binding.Slot).ResourceTag, world.Pool.Slot(second.Binding.Slot).ResourceTag, logicalSlot(world.Pool.FreeFirst()), logicalSlot(world.Pool.First(ActorPoolMoving)))
		}
	})
}

func TestWorldMovingGroupReleasesLaterDeadPartsInCurrentTraversal(t *testing.T) {
	world := testWorld(t)
	part := visualassets.ActorPart{ResourceTag: 208, MotionMode: "path", Linked: true}
	world.kinds[2] = &visualassets.WaveActor{Kind: 2, Parts: []visualassets.ActorPart{part, part}}
	path := &visualassets.Path{ID: 2, Commands: []visualassets.PathCommand{{Kind: "end"}}}
	world.paths[2] = path
	if err := world.spawnWave(visualassets.Wave{EnemyKind: 2, PathID: 2, Count: 1, MotionBudget: 1}); err != nil {
		t.Fatal(err)
	}
	head, tail := world.Actors[0], world.Actors[1]
	if err := world.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
	if world.Pool.Slot(head.Binding.Slot).ResourceTag != 4 || !world.Pool.Slot(head.Binding.Slot).allocated || world.Pool.Slot(tail.Binding.Slot).allocated {
		t.Fatal("linked self-removal must retain current head and release its later dead part")
	}
}
