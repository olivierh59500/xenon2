package engine

import (
	"fmt"
	"testing"

	"xenon2/internal/visualassets"
)

func TestFirstWorldTileCannonUpdatesAndRestoresTerrain(t *testing.T) {
	w := testWorld(t)
	patch := func(a, b, c, d uint16) visualassets.TilePatch {
		return visualassets.TilePatch{Columns: 2, Rows: 2, Tiles: []uint16{a, b, c, d}}
	}
	variant := visualassets.FixedTileVariant{ID: 0, ResourceTag: 228, OriginOffsetX: -8, OriginOffsetY: -8, Initial: patch(1, 2, 3, 4), Destroyed: patch(9, 10, 11, 12), Frames: []visualassets.TilePatch{patch(1, 2, 3, 4), patch(5, 6, 7, 8), patch(1, 2, 3, 4), patch(5, 6, 7, 8)}}
	w.Level.FixedTiles = &visualassets.FixedTiles{Kinds: []visualassets.FixedTileKind{{Kind: 2, Health: 4, Behavior: "first-tile-cannon", Variants: []visualassets.FixedTileVariant{variant}}}}
	if !w.spawnFixedTile(visualassets.FixedEncounter{EnemyKind: 2, X: 40, Y: 1000}) {
		t.Fatal("tile encounter was not consumed")
	}
	actor := w.Actors[0]
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
				if actor.Active || w.Score != (variant+1)*100 {
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
