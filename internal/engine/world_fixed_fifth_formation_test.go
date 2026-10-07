package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestFifthPersistentTileNativeDamageOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local persistent turret reference not supplied")
	}
	level, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	art, _, err := visualassets.DecodeFixedTiles(5, level)
	if err != nil {
		t.Fatal(err)
	}
	comparisons := 0
	nativeCombatRows(t, "fixed-fifth-persistent.csv", func(v []int64) {
		w := testWorld(t)
		w.Level.Number, w.Level.FixedTiles, w.ScrollY = 5, art, 900
		clip := visualassets.NamedActorAnimation{Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "explosion", Duration: 2}}}}
		w.commonAnimations["explosion-large"] = clip
		record := visualassets.FixedEncounter{EnemyKind: 9, State2: int(v[0]), X: 104, Y: 1008, Variant: int(v[1])}
		w.spawnFixed(record)
		actor := w.Actors[0]
		if actor.fifthTile == nil || actor.fifthPersistentSelector != record.State2 {
			t.Fatal("missing persistent controller")
		}
		w.damageActor(actor, uint16(v[2]))
		flag := 0
		if w.fifthDestroyedTurrets[record.State2] {
			flag = 255
		}
		tag := actor.part.ResourceTag
		if !actor.Active {
			tag = 4
		}
		explosions := 0
		for _, candidate := range w.Actors {
			if candidate.ActorList == "transient" {
				explosions++
			}
		}
		if actor.Health != int(v[4]) || w.Score != int(v[5]) || explosions != int(v[6]) || tag != int(v[7]) || flag != int(v[8]) {
			t.Fatalf("damage/remembered state differs: health%d score%d explosions%d flag%d native%v", actor.Health, w.Score, explosions, flag, v)
		}
		before := w.nextActorID
		w.spawnFixed(record)
		if w.nextActorID-before != int(v[9]) {
			t.Fatalf("same selector re-entry differs: native%v", v)
		}
		record.State2 = 1 - record.State2
		before = w.nextActorID
		w.spawnFixed(record)
		if w.nextActorID-before != int(v[10]) {
			t.Fatalf("independent selector re-entry differs: native%v", v)
		}
		comparisons++
	})
	if comparisons != 80 {
		t.Fatalf("incomplete persistent comparison: %d", comparisons)
	}
	t.Logf("Compared %d original persistent turret damage and re-entry cases.", comparisons)
}

func TestFifthFormationOriginalResourcesAndTailOrderOptional(t *testing.T) {
	data := originalWorldData(t, 5)
	for _, record := range data.Encounters.Fixed {
		if record.EnemyKind != 7 {
			continue
		}
		w, err := NewWorld(data)
		if err != nil {
			t.Fatal(err)
		}
		w.ScrollY = record.TriggerY
		w.spawnFixed(record)
		members := 0
		for index := w.Pool.First(ActorPoolMoving); index != NoActorSlot; index = w.Pool.Next(index) {
			actor := w.poolActors[index]
			if actor == nil || actor.fifthFormation == nil {
				continue
			}
			if actor.fifthFormation.Motion.Remaining != -members*data.FixedSprites.FifthFormation.MemberSpacing {
				t.Fatal("formation member delay or tail order changed")
			}
			members++
		}
		if members != 10 {
			t.Fatalf("formation has %d members", members)
		}
		for range 64 {
			w.ScrollDelta = 1
			if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
				t.Fatal(err)
			}
		}
		shots := 0
		for _, shot := range w.Projectiles {
			if shot.Active && shot.Sprite == data.FixedSprites.FifthFormation.ShotSprite && shot.Atlas == "fixed" {
				shots++
			}
		}
		if shots == 0 || shots%8 != 0 {
			t.Fatalf("radial formation bursts incomplete: %d", shots)
		}
		for _, actor := range w.Actors {
			if actor.fifthFormation != nil && actor.Active && (actor.Sprite == "" || actor.Collision.Right < actor.Collision.Left || actor.Collision.Bottom < actor.Collision.Top) {
				t.Fatalf("formation lost original image or collider: %+v", actor)
			}
		}
	}
}

func TestFifthPersistentTurretsSurviveCheckpointRestartOptional(t *testing.T) {
	data := originalWorldData(t, 5)
	for _, record := range data.Encounters.Fixed {
		if record.EnemyKind != 9 {
			continue
		}
		w, err := NewWorld(data)
		if err != nil {
			t.Fatal(err)
		}
		w.ScrollY = record.TriggerY
		w.spawnFixed(record)
		var target *WorldActor
		for _, actor := range w.Actors {
			if actor.fifthTile != nil && actor.fixedTileArt.Kind == 9 {
				target = actor
				break
			}
		}
		if target == nil {
			t.Fatal("persistent encounter missing")
		}
		w.damageActor(target, uint16(target.Health))
		w.RestartCheckpoint()
		before := w.nextActorID
		w.spawnFixed(record)
		if w.nextActorID != before {
			t.Fatal("checkpoint restart resurrected a destroyed persistent turret")
		}
	}
}
