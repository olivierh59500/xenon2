package engine

import (
	"fmt"
	"testing"
	"xenon2/internal/visualassets"
)

// This integration check reaches later animation frames and emitted artwork;
// it checks resource completeness, independently of graphical scene fidelity.
func TestOriginalFixedEncounterVisualReferencesOptional(t *testing.T) {
	for level := 1; level <= 5; level++ {
		data := originalWorldData(t, level)
		atlases := map[string]*visualassets.SpriteAtlas{"moving": &data.Actors.Atlas, "fixed": &data.FixedSprites.Atlas, "enemy-shots": &data.Rules.EnemyShots, "common": data.Common, "guardian-parts": data.GuardianParts}
		if data.Guardians != nil {
			atlases["guardians"] = &data.Guardians.Atlas
		}
		names := make(map[string]map[string]bool)
		for name, atlas := range atlases {
			names[name] = make(map[string]bool)
			if atlas != nil {
				for _, sprite := range atlas.Sprites {
					names[name][sprite.Name] = true
				}
			}
		}
		checkImage := func(atlas, name string) {
			if name != "" && !names[atlas][name] {
				t.Fatalf("level%d references missing artwork %s/%s", level, atlas, name)
			}
		}
		tiles := make(map[uint16]bool)
		for _, tile := range data.Terrain.Tiles {
			tiles[tile.ID] = true
		}
		checkPatch := func(patch visualassets.TilePatch) {
			if len(patch.Tiles) != patch.Columns*patch.Rows {
				t.Fatal("compound tile patch has inconsistent dimensions")
			}
			for _, tile := range patch.Tiles {
				if tile != 0 && !tiles[tile] {
					t.Fatalf("level%d references missing tile%d", level, tile)
				}
			}
		}
		for index, record := range data.Encounters.Fixed {
			t.Run(fmt.Sprintf("%d/%d", level, index), func(t *testing.T) {
				w, err := NewWorld(data)
				if err != nil {
					t.Fatal(err)
				}
				w.ScrollY, w.RenderScrollY = record.TriggerY, record.TriggerY
				w.spawnFixed(record)
				for range 96 {
					w.Frame++
					w.ScrollDelta = 1
					if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
						t.Fatal(err)
					}
					if err := w.advancePooledProjectiles(Input{}); err != nil {
						t.Fatal(err)
					}
					if err := w.advanceActorPhase(ActorPoolScenery, Input{}); err != nil {
						t.Fatal(err)
					}
					for _, actor := range w.Actors {
						if actor.Active && actor.Visible {
							checkImage(actor.Atlas, actor.Sprite)
							if actor.Patch != nil {
								checkPatch(*actor.Patch)
							}
							for _, overlay := range actor.TileOverlays {
								checkPatch(overlay.Patch)
							}
							for _, extra := range actor.Extras {
								checkImage(extra.Atlas, extra.Sprite)
							}
						}
					}
					for _, shot := range w.Projectiles {
						if shot.Active {
							checkImage(shot.Atlas, shot.Sprite)
						}
					}
					w.compactActors()
				}
			})
		}
	}
}

func TestOriginalMovingWaveVisualReferencesOptional(t *testing.T) {
	for level := 1; level <= 5; level++ {
		data := originalWorldData(t, level)
		names := make(map[string]map[string]bool)
		for name, atlas := range map[string]*visualassets.SpriteAtlas{"moving": &data.Actors.Atlas, "common": data.Common, "enemy-shots": &data.Rules.EnemyShots} {
			names[name] = make(map[string]bool)
			for _, sprite := range atlas.Sprites {
				names[name][sprite.Name] = true
			}
		}
		for index, record := range data.Encounters.Moving {
			t.Run(fmt.Sprintf("%d/%d", level, index), func(t *testing.T) {
				w, err := NewWorld(data)
				if err != nil {
					t.Fatal(err)
				}
				if err := w.spawnWave(record); err != nil {
					t.Fatal(err)
				}
				for range 64 {
					if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
						t.Fatal(err)
					}
					if err := w.advancePooledProjectiles(Input{}); err != nil {
						t.Fatal(err)
					}
					for _, actor := range w.Actors {
						if actor.Active && actor.Visible && !actor.firstGuardian && !actor.secondGuardian && actor.secondNode == nil && !names[actor.Atlas][actor.Sprite] {
							t.Fatalf("wave%d level%d references missing %s/%s", index, level, actor.Atlas, actor.Sprite)
						}
					}
					for _, shot := range w.Projectiles {
						if shot.Active && !names[shot.Atlas][shot.Sprite] {
							t.Fatalf("wave%d level%d shot references missing %s/%s", index, level, shot.Atlas, shot.Sprite)
						}
					}
					w.compactActors()
				}
			})
		}
	}
}
