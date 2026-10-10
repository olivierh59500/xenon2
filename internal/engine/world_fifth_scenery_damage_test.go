package engine

import (
	"fmt"
	"testing"

	"xenon2/internal/visualassets"
)

func fifthSceneryFixture(t *testing.T, kind int) (*World, []*WorldActor) {
	t.Helper()
	w := testWorld(t)
	w.Level.Number, w.ScrollY = 5, 900
	patch := visualassets.TilePatch{Columns: 2, Rows: 2, Tiles: []uint16{1, 2, 3, 4}}
	art := visualassets.FixedTileKind{Kind: kind, Health: 20, Variants: []visualassets.FixedTileVariant{{ID: 0, ResourceTag: 244, Initial: patch, Destroyed: visualassets.TilePatch{Columns: 2, Rows: 2, Tiles: make([]uint16, 4)}}}}
	if kind == 4 {
		art.Variants[0].ResourceTag = 248
	}
	if kind == 1 {
		art.Health = 10
		art.Variants[0].Initial = visualassets.TilePatch{Columns: 4, Rows: 1, Tiles: []uint16{1, 2, 3, 4}}
		art.Variants[0].Destroyed = visualassets.TilePatch{Columns: 4, Rows: 1, Tiles: []uint16{9, 0, 0, 10}}
		for index, tag := range []int{296, 304, 300} {
			art.Parts = append(art.Parts, visualassets.FixedTilePart{ResourceTag: tag, OffsetX: []int{0, 16, 48}[index], Damageable: index != 1, Flash: visualassets.TilePatch{Columns: 1, Rows: 1, Tiles: []uint16{8}}})
		}
	}
	w.Level.FixedTiles = &visualassets.FixedTiles{Kinds: []visualassets.FixedTileKind{art}}
	for slot := w.Pool.FreeFirst(); slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
		w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	}
	for _, name := range []string{"explosion-small", "explosion-large"} {
		w.commonAnimations[name] = visualassets.NamedActorAnimation{Ending: "remove", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "fx", Duration: 2}}}}
	}
	if !w.spawnFifthTile(visualassets.FixedEncounter{EnemyKind: kind, State2: 1, X: 40, Y: 1008}) || w.poolError != nil {
		t.Fatalf("fifth scenery constructor failed: %v", w.poolError)
	}
	return w, append([]*WorldActor(nil), w.Actors...)
}

func TestFifthSceneryPublishesNonlethalHealthAndOriginalSelectorWords(t *testing.T) {
	for _, kind := range []int{1, 3, 4, 9} {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			w, group := fifthSceneryFixture(t, kind)
			actor := group[0]
			want := uint16(0xffff)
			if kind == 1 {
				want = waveConstructorResidueFixture(actor.Binding.Slot).VerticalFraction
			} else if kind == 4 {
				want = 0
			} else if kind == 9 {
				want = 1
			}
			if w.Pool.Slot(actor.Binding.Slot).Residue.VerticalFraction != want {
				t.Fatal("constructor lost its original selector/sentinel word")
			}
			health := actor.Health
			w.damageActor(actor, 1)
			if actor.Health != health-1 || w.Pool.Slot(actor.Binding.Slot).Residue.Health != uint16(health-1) || !actor.Active || !actor.Visible || !actor.Flash || actor.Patch == nil || w.Score != 0 {
				t.Fatal("nonlethal scenery hit deferred health or entered destruction")
			}
		})
	}
}

func TestFifthBarrierSaturationRetiresItsCohortWithoutKillingOtherEffects(t *testing.T) {
	for _, part := range []int{0, 2} {
		t.Run(fmt.Sprint(part), func(t *testing.T) {
			w, group := fifthSceneryFixture(t, 1)
			left := group[0].Binding.Slot
			fillWaveDamageMovingTail(t, w)
			w.damageActor(group[part], 10)
			for _, member := range group {
				if member.Active || member.Visible {
					t.Fatal("destroyed barrier retained an active logical member")
				}
				if slot := w.Pool.Slot(member.Binding.Slot); slot.EntityID == member.ID && slot.ResourceTag != 4 {
					t.Fatal("barrier retirement missed an original physical member")
				}
			}
			effect := w.poolActors[left]
			if effect == nil || effect.ID == group[0].ID || effect.Active != (part == 2) || w.Score != 200 || w.poolError != nil {
				t.Fatalf("barrier replacement lifetime, score or pool differs: effect %+v score %d error %v", effect, w.Score, w.poolError)
			}
			for index, value := range []uint16{9, 0, 0, 10} {
				if w.Level.Terrain.Map[62*20+2+index] != value {
					t.Fatal("barrier destruction failed to restore its full passage")
				}
			}
			if err := w.advancePooledProjectiles(Input{}); err != nil {
				t.Fatal(err)
			}
			if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
				t.Fatal(err)
			}
			for _, member := range group {
				if slot := w.Pool.Slot(member.Binding.Slot); slot.allocated && slot.EntityID == member.ID {
					t.Fatal("next owning traversal kept a retired barrier entry")
				}
			}
		})
	}
}

func TestFifthTurretDeathRetiresAReclaimedParentAndKeepsPersistentSelection(t *testing.T) {
	for _, kind := range []int{3, 4, 9} {
		w, group := fifthSceneryFixture(t, kind)
		actor := group[0]
		slot, identity := actor.Binding.Slot, actor.ID
		fillWaveDamageMovingTail(t, w)
		w.damageActor(actor, 127)
		points := 400
		if kind == 4 {
			points = 500
		}
		current, effect := w.Pool.Slot(slot), w.poolActors[slot]
		if actor.Active || current.EntityID == identity || current.ResourceTag != 4 || effect == nil || effect.Active || current.Residue.Health != uint16(65536+20-127) || w.Score != points || w.SoundRequests[2] != "sampled-effect-03" {
			t.Fatalf("kind %d lost source parent subtraction/retirement or score: %+v score %d", kind, current, w.Score)
		}
		if kind == 9 {
			before := w.nextActorID
			w.spawnFifthTile(visualassets.FixedEncounter{EnemyKind: 9, State2: 1, X: 40, Y: 1008})
			if !w.fifthDestroyedTurrets[1] || w.nextActorID != before {
				t.Fatal("reclaimed persistent turret was resurrected on re-entry")
			}
		}
	}
}

func TestFifthAdjacentTileChangesFollowLinearMapNeighboursAtOuterColumns(t *testing.T) {
	for _, example := range []struct{ column, offset, position int }{{18, 2, 220}, {0, -1, 199}} {
		w := testWorld(t)
		w.Level.Terrain.Map[example.position] = 5
		changes := []visualassets.ConditionalTileReplacement{{ColumnOffset: example.offset, Before: 5, After: visualassets.TilePatch{Columns: 1, Rows: 2, Tiles: []uint16{6, 7}}}}
		w.replaceFifthAdjacentTiles(example.column, 10, changes)
		if w.Level.Terrain.Map[example.position] != 6 || w.Level.Terrain.Map[example.position+20] != 7 {
			t.Fatal("outer-column neighbour was clipped instead of following its linear address")
		}
		changes[0].Before, changes[0].After.Tiles = 6, []uint16{8, 9}
		w.replaceFifthAdjacentTiles(example.column, 10, changes)
		if w.Level.Terrain.Map[example.position] != 8 || w.Level.Terrain.Map[example.position+20] != 9 {
			t.Fatal("restoration did not follow the same wrapped neighbour")
		}
	}
}
