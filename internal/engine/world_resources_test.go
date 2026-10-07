package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/visualassets"
)

func originalWorldData(t *testing.T, number int) LevelData {
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
