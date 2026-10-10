package engine

import (
	"fmt"
	"testing"

	"xenon2/internal/visualassets"
)

func terrainCannonFixture(t *testing.T, level int, retained ActorResidue) (*World, *WorldActor) {
	t.Helper()
	w := testWorld(t)
	w.Level.Number, w.ScrollY = level, 900
	w.Pool.Slot(w.Pool.FreeFirst()).Residue = retained
	patch := func(a, b, c, d uint16) visualassets.TilePatch {
		return visualassets.TilePatch{Columns: 2, Rows: 2, Tiles: []uint16{a, b, c, d}}
	}
	variant := visualassets.FixedTileVariant{ID: 0, ResourceTag: 228, OriginOffsetX: -8, OriginOffsetY: -8, Initial: patch(1, 2, 3, 4), Destroyed: patch(9, 10, 11, 12), Frames: []visualassets.TilePatch{patch(1, 2, 3, 4), patch(5, 6, 7, 8), patch(1, 2, 3, 4), patch(5, 6, 7, 8)}}
	w.Level.FixedTiles = &visualassets.FixedTiles{Kinds: []visualassets.FixedTileKind{{Kind: 2, Health: 4, Behavior: "first-tile-cannon", Variants: []visualassets.FixedTileVariant{variant}}}}
	if !w.spawnFixedTile(visualassets.FixedEncounter{EnemyKind: 2, X: 40, Y: 1000}) {
		t.Fatal("tile encounter was not consumed")
	}
	return w, w.Actors[0]
}

func TestFirstWorldTileCannonUpdatesAndRestoresTerrain(t *testing.T) {
	w, actor := terrainCannonFixture(t, 1, ActorResidue{})
	if actor.part.ResourceTag != 228 || actor.Binding.EntityID == 0 || actor.Visible {
		t.Fatal("terrain actor must use its moving-list slot without a sprite placeholder")
	}
	index := 62*20 + 2
	if w.Level.Terrain.Map[index] != 1 || w.Level.Terrain.Map[index+21] != 4 {
		t.Fatal("creation did not install the original terrain patch")
	}
	w.ScrollY, w.MaximumScrollY = 900, 1000
	w.advanceFixedTile(actor)
	w.advanceFixedTile(actor)
	if w.Level.Terrain.Map[index] != 5 || actor.Collision.Left != 36 || actor.Collision.Top != 96 {
		t.Fatal("tile animation or original collision bounds differ")
	}
	w.damageActor(actor, 4)
	if actor.Active || w.Score != 100 || w.Level.Terrain.Map[index] != 9 || w.Level.Terrain.Map[index+21] != 12 || w.Pool.Slot(actor.Binding.Slot).ResourceTag != 4 {
		t.Fatal("death must score once, restore the terrain and leave a dead slot")
	}
}

func TestTerrainCannonPublishesWorldPositionAndRetainsUnassignedWords(t *testing.T) {
	for level := 1; level <= 5; level++ {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			retained := waveConstructorResidueFixture(7)
			w, actor := terrainCannonFixture(t, level, retained)
			want := retained
			want.X, want.Y, want.Counter = 32, 992, 0
			want.Health, want.VerticalFraction, want.EmitterClock = 4, 0, 0
			if level != 4 {
				want.PowerOrScore = 100
			}
			if got := w.Pool.Slot(actor.Binding.Slot).Residue; got != want {
				t.Fatalf("terrain constructor fields %v, want %v", got, want)
			}
			if actor.Y != 92 || !actor.part.StrongHealth {
				t.Fatal("screen anchor or inherited contact strength differs")
			}
			w.advanceFixedTile(actor)
			w.finishActorUpdate(actor)
			if got := w.Pool.Slot(actor.Binding.Slot).Residue; got.Y != 992 || got.XFraction != retained.XFraction || got.Direction != retained.Direction || got.WaveBonusToken != retained.WaveBonusToken || got.PowerOrScore != want.PowerOrScore {
				t.Fatal("terrain update replaced world Y or unrelated retained words")
			}
		})
	}
}

func TestTerrainCannonNonlethalDamagePublishesHealthImmediately(t *testing.T) {
	w, actor := terrainCannonFixture(t, 1, ActorResidue{})
	w.damageActor(actor, 1)
	slot := w.Pool.Slot(actor.Binding.Slot)
	if !actor.Active || !actor.Visible || actor.Patch == nil || actor.Health != 3 || slot.Residue.Health != 3 || slot.Residue.Y != 992 || slot.ResourceTag != 228 || w.Score != 0 || len(w.Actors) != 1 {
		t.Fatal("nonlethal terrain damage deferred health or ran the death callback")
	}
}

func TestTerrainCannonRestoresValidTilesWhenExplosionReclaimsItsSlot(t *testing.T) {
	for _, level := range []int{1, 4} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			w, actor := terrainCannonFixture(t, level, waveConstructorResidueFixture(7))
			w.commonAnimations["explosion-large"] = visualassets.NamedActorAnimation{Ending: "hold", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "fx"}}}}
			slot, identity := actor.Binding.Slot, actor.ID
			fillWaveDamageMovingTail(t, w)
			w.damageActor(actor, 4)
			current, effect := w.Pool.Slot(slot), w.poolActors[slot]
			points := 100
			if level == 4 {
				points = 400
			}
			if w.poolError != nil || actor.Active || current.EntityID == identity || current.ResourceTag != 4 || current.list != ActorPoolProjectile || effect == nil || effect.Active || w.Score != points {
				t.Fatalf("terrain death lost source score or physical replacement retirement: %+v score %d error %v", current, w.Score, w.poolError)
			}
			for i, index := range []int{62*20 + 2, 62*20 + 3, 63*20 + 2, 63*20 + 3} {
				if w.Level.Terrain.Map[index] != uint16(9+i) {
					t.Fatal("reclaimed explosion corrupted the intended destroyed terrain")
				}
			}
			if w.SoundRequests[2] != "sampled-effect-03" {
				t.Fatal("terrain destruction lost its original sound voice")
			}
			if err := w.advancePooledProjectiles(Input{}); err != nil {
				t.Fatal(err)
			}
			if w.Pool.Slot(slot).allocated {
				t.Fatal("next projectile traversal retained the reclaimed dead effect")
			}
		})
	}
}

func TestWorldTileCannonsUseOriginalResourcesAcrossFiveLevelsOptional(t *testing.T) {
	for number := 1; number <= 5; number++ {
		t.Run(fmt.Sprint(number), func(t *testing.T) {
			w, err := NewWorld(originalWorldData(t, number))
			if err != nil {
				t.Fatal(err)
			}
			w.Actors = nil
			kind := w.Level.FixedTiles.Kinds[0]
			for variant := range 2 {
				if !w.spawnFixedTile(visualassets.FixedEncounter{EnemyKind: kind.Kind, X: 100, Y: 1000, Variant: variant}) {
					t.Fatal("verified terrain cannon was not created")
				}
				actor := w.Actors[0]
				w.ScrollY, w.MaximumScrollY = 900, 1000
				for range 160 {
					w.advanceFixedTile(actor)
				}
				if actor.Collision.Empty() || !actor.Active {
					t.Fatal("cannon lost source collision or lifetime")
				}
				w.damageActor(actor, uint16(actor.Health))
				points := 100
				if number == 4 {
					points = 400
				}
				if actor.Active || w.Score != (variant+1)*points {
					t.Fatal("cannon death or score differs")
				}
			}
			if len(w.Projectiles) == 0 {
				t.Fatal("native tile cannon never emitted")
			}
			for _, shot := range w.Projectiles {
				if shot.Sprite == "" || shot.Binding.EntityID == 0 {
					t.Fatal("shot lacks exported artwork or shared allocation")
				}
			}
		})
	}
}
