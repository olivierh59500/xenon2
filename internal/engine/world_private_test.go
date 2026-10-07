package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/visualassets"
)

// TestWorldOriginalLevelDataOptional exercises actual decoded formations and
// projectiles without a graphics context. It is not a full-game victory test:
// the initial guardian scroll bounds deliberately remain in force.
func TestWorldOriginalLevelDataOptional(t *testing.T) {
	root := os.Getenv("XENON2_RUNTIME_ASSET_DIR")
	if root == "" {
		t.Skip("set XENON2_RUNTIME_ASSET_DIR to local exported resources")
	}
	for number := 1; number <= 5; number++ {
		t.Run(fmt.Sprint(number), func(t *testing.T) {
			terrain, paths, encounters := &visualassets.Terrain{}, &visualassets.Paths{}, &visualassets.Encounters{}
			actors, fixed, rules := &visualassets.Actors{}, &visualassets.FixedSprites{}, &visualassets.LevelRules{}
			for _, item := range []struct {
				suffix string
				value  any
			}{{"", terrain}, {"-paths", paths}, {"-encounters", encounters}, {"-actors", actors}, {"-fixed-sprites", fixed}, {"-rules", rules}} {
				data, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("level-%d%s.json", number, item.suffix)))
				if err != nil {
					t.Fatal(err)
				}
				if err = json.Unmarshal(data, item.value); err != nil {
					t.Fatal(err)
				}
			}
			world, err := NewWorld(LevelData{Number: number, Terrain: terrain, Paths: paths, Encounters: encounters, Actors: actors, FixedSprites: fixed, Rules: rules})
			if err != nil {
				t.Fatal(err)
			}
			for tick := range 3200 {
				if err := world.Step(Input{Fire: true}); err != nil {
					t.Fatalf("pass %d: %v", tick, err)
				}
				if world.ScrollY < rules.InitialMinimumScrollY || world.ScrollY > rules.InitialMaximumScrollY {
					t.Fatalf("pass %d scroll %d outside original initial bounds", tick, world.ScrollY)
				}
				if len(world.Actors) > 2000 || len(world.Projectiles) > 2000 || len(world.SmallShots) > 2000 {
					t.Fatalf("unbounded active entities on pass %d", tick)
				}
			}
			if world.Frame != 3200 || world.Equipment.Shield != 39 {
				t.Fatal("diagnostic pass count or noncolliding ship state differs")
			}
		})
	}
}
