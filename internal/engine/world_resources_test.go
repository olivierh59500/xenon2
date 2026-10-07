package engine

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/visualassets"
)

func originalWorldData(t testing.TB, number int) LevelData {
	t.Helper()
	root := os.Getenv("XENON2_RUNTIME_ASSET_DIR")
	if root == "" {
		t.Skip("set XENON2_RUNTIME_ASSET_DIR to local exported resources")
	}
	data := LevelData{Number: number, Terrain: &visualassets.Terrain{}, Paths: &visualassets.Paths{},
		Encounters: &visualassets.Encounters{}, Actors: &visualassets.Actors{}, FixedSprites: &visualassets.FixedSprites{},
		FixedTiles: &visualassets.FixedTiles{}, Rules: &visualassets.LevelRules{}, Ships: &visualassets.ShipArt{},
		Common: &visualassets.SpriteAtlas{}, Guardians: &visualassets.Guardians{}}
	var groups struct {
		Groups []visualassets.GuardianGroup `json:"groups"`
		Atlas  visualassets.SpriteAtlas     `json:"atlas"`
	}
	for _, item := range []struct {
		name   string
		target any
	}{
		{fmt.Sprintf("level-%d.json", number), data.Terrain},
		{fmt.Sprintf("level-%d-paths.json", number), data.Paths},
		{fmt.Sprintf("level-%d-encounters.json", number), data.Encounters},
		{fmt.Sprintf("level-%d-actors.json", number), data.Actors},
		{fmt.Sprintf("level-%d-fixed-sprites.json", number), data.FixedSprites},
		{fmt.Sprintf("level-%d-fixed-tiles.json", number), data.FixedTiles},
		{fmt.Sprintf("level-%d-rules.json", number), data.Rules},
		{fmt.Sprintf("level-%d-guardians.json", number), data.Guardians},
		{fmt.Sprintf("level-%d-guardian-groups.json", number), &groups},
		{"ships.json", data.Ships}, {"common-actors.json", data.Common},
	} {
		bytes, err := os.ReadFile(filepath.Join(root, item.name))
		if number == 1 && item.target == &groups && os.IsNotExist(err) {
			continue
		}
		if (number == 3 || number == 4) && item.target == data.Guardians && os.IsNotExist(err) {
			data.Guardians = nil
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(bytes, item.target); err != nil {
			t.Fatal(err)
		}
	}
	data.GuardianGroups, data.GuardianParts = groups.Groups, &groups.Atlas
	return data
}

// playableOriginalWorldData adds the actual tile coverage and ship stencil.
// Tests using metadata alone cannot verify wall contact or crushing behavior.
func playableOriginalWorldData(t testing.TB, number int) LevelData {
	t.Helper()
	data := originalWorldData(t, number)
	root := os.Getenv("XENON2_RUNTIME_ASSET_DIR")
	file, err := os.Open(filepath.Join(root, fmt.Sprintf("level-%d-tiles.png", number)))
	if err != nil {
		t.Fatal(err)
	}
	picture, err := png.Decode(file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	data.Terrain.Atlas = image.NewNRGBA(picture.Bounds())
	draw.Draw(data.Terrain.Atlas, picture.Bounds(), picture, picture.Bounds().Min, draw.Src)
	stencilBytes, err := os.ReadFile(filepath.Join(root, "player-terrain-stencil.json"))
	if err != nil {
		t.Fatal(err)
	}
	data.PlayerStencil = &visualassets.PlayerTerrainStencil{}
	if err := json.Unmarshal(stencilBytes, data.PlayerStencil); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestOriginalLevelStartsExerciseTerrainAndNormalShipLossOptional(t *testing.T) {
	for number := 1; number <= 5; number++ {
		t.Run(fmt.Sprint(number), func(t *testing.T) {
			w, err := NewWorld(playableOriginalWorldData(t, number))
			if err != nil {
				t.Fatal(err)
			}
			if w.Coverage == nil {
				t.Fatal("normal resource test omitted terrain coverage")
			}
			for pass := range 800 {
				motion := MotionInput{Left: pass%120 < 30, Right: pass%120 >= 60 && pass%120 < 90}
				if err := w.Step(Input{Motion: motion, Fire: pass%12 < 8}); err != nil {
					t.Fatalf("pass%d:%v", pass, err)
				}
				if w.Ready {
					if err := w.Step(Input{Fire: true}); err != nil {
						t.Fatal(err)
					}
				}
				if w.GameOver {
					break
				}
			}
			if w.Frame == 0 || len(w.Level.Terrain.Map) != 6000 || w.poolError != nil {
				t.Fatal("normal game loop lost playable state")
			}
			t.Logf("Normal input exercised %d passes; remaining ships=%d shield=%d", w.Frame, w.Equipment.Lives, w.Equipment.Shield)
		})
	}
}

func BenchmarkOriginalMiddleArenaWorld(b *testing.B) {
	data := originalWorldData(b, 1)
	w, err := NewWorld(data)
	if err != nil {
		b.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 3000, 3344, 3344
	w.InvulnerableFrames = 1 << 30
	w.cursor = RestartEncounterCursor(w.ScrollY)
	if err := w.advanceFirstMiddleStage(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		w.Player.X = 160
		w.Player.Y = 176
		w.ScrollY, w.MaximumScrollY = 3000, 3344
		w.ShopReady, w.Ready, w.GameOver = false, false, false
		if err := w.Step(Input{Fire: true}); err != nil {
			b.Fatal(err)
		}
	}
}

func TestWorldWeaponsSharePoolWithOriginalEnemiesOptional(t *testing.T) {
	data := originalWorldData(t, 1)
	for _, item := range []Item{ItemCannon, ItemMissileLauncher, ItemLaser, ItemFlamer, ItemDrone, ItemElectroBall, ItemMineSmall, ItemMineLarge, ItemBomb, ItemHomingMissile} {
		t.Run(fmt.Sprint(item), func(t *testing.T) {
			w, err := NewWorld(data)
			if err != nil {
				t.Fatal(err)
			}
			w.Equipment.ApplyItem(item)
			w.InvulnerableFrames = 10000
			for pass := range 400 {
				if err := w.Step(Input{Fire: true, Motion: MotionInput{Up: pass%60 < 20, Down: pass%60 >= 40}}); err != nil {
					t.Fatalf("pass %d: %v", pass, err)
				}
				allocated := 0
				for _, slot := range w.Pool.slots {
					if slot.allocated {
						allocated++
					}
				}
				if allocated != len(w.poolBindings) || allocated > ActorPoolCapacity {
					t.Fatalf("pass %d: allocated=%d tracked=%d", pass, allocated, len(w.poolBindings))
				}
			}
			w.RestartCheckpoint()
			if w.poolError != nil {
				t.Fatal(w.poolError)
			}
			if w.Weapons.mounts[0].Binding.EntityID == 0 || w.Pool.Slot(w.Weapons.mounts[0].Binding.Slot).ResourceTag != equipmentResourceTag(w.Checkpoint.Loadout.Primary.Item) {
				t.Fatal("checkpoint did not reconstruct the saved primary weapon")
			}
		})
	}
}
