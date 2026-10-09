package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/visualassets"
)

// The reference starts with the original constructors and real map records.
// Seeding tags directly would miss a reversed record-to-orientation mapping.
func TestWallCannonConstructionAndFireNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare original cannon constructors")
	}
	var banks [5]*visualassets.FixedTiles
	for index, file := range []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", file+".decoded"))
		if err != nil {
			t.Fatal(err)
		}
		banks[index], _, err = visualassets.DecodeFixedTiles(index+1, data)
		if err != nil {
			t.Fatal(err)
		}
	}
	var kind visualassets.FixedTileKind
	var variant visualassets.FixedTileVariant
	var state FixedTileState
	var random RandomState
	var tiles [4]uint16
	constructors, passes, shots := 0, 0, 0
	writePatch := func(patch visualassets.TilePatch) {
		for y := range patch.Rows {
			for x := range patch.Columns {
				tiles[y*2+x] = patch.Tiles[y*patch.Columns+x]
			}
		}
	}
	nativeCombatRows(t, "wall-cannon-trace.csv", func(v []int64) {
		level, record, frame := int(v[0]), int(v[1]), int(v[4])
		if frame == -1 {
			found := false
			for _, candidate := range banks[level-1].Kinds {
				if candidate.Kind == int(v[2]) {
					kind, found = candidate, true
					break
				}
			}
			if !found || int(v[3]) >= len(kind.Variants) {
				t.Fatalf("missing level %d kind %d variant %d", level, v[2], v[3])
			}
			variant = kind.Variants[v[3]]
			state = FixedTileState{Kind: kind.Kind, Variant: variant.ID, X: int(v[5]) + variant.OriginOffsetX, WorldY: int(v[6]) + variant.OriginOffsetY}
			random = NewRandomState()
			tiles = [4]uint16{}
			writePatch(variant.Initial)
			if variant.ResourceTag != int(v[10]) || kind.Health != int(v[11]) {
				t.Fatalf("level %d record %d constructor: tag %d health %d, want tag %d health %d", level, record, variant.ResourceTag, kind.Health, v[10], v[11])
			}
			constructors++
		} else {
			var event FixedTileEvents
			if level == 1 {
				event = StepFirstTileCannon(&state, kind, int(v[7]), 4607, &random)
			} else {
				event = StepTileFireCycle(&state, kind, int(v[7]), 4607, &random)
			}
			if event.WriteTiles {
				writePatch(variant.Frames[event.Frame])
			}
			wantCollision := CollisionRect{Left: int(v[14]), Top: int(v[15]), Right: int(v[16]), Bottom: int(v[17])}
			if event.Collision != wantCollision || event.Shot != (v[18] != 0) {
				t.Fatalf("level %d record %d pass %d: event %+v, native collision %+v shots %d", level, record, frame, event, wantCollision, v[18])
			}
			if event.Shot {
				if v[18] != 1 || event.ShotX != int(v[19]) || event.ShotY != int(v[20]) || event.ShotDirection != uint8(v[21]) || event.ShotSpeed != int(v[22]) {
					t.Fatalf("level %d record %d pass %d: shot %+v, native %v", level, record, frame, event, v[18:23])
				}
				shots++
			}
			passes++
		}
		wantTiles := [4]uint16{uint16(v[23]), uint16(v[24]), uint16(v[25]), uint16(v[26])}
		if state.X != int(v[8]) || state.WorldY != int(v[9]) || state.Phase != int(v[12]) || state.Accumulator != uint8(v[13]) || state.Removed || tiles != wantTiles || random.A != uint32(v[27]) || random.B != uint32(v[28]) {
			t.Fatalf("level %d record %d pass %d: state %+v tiles %v RNG %+v, native %v", level, record, frame, state, tiles, random, v)
		}
	})
	if constructors != 71 || passes != 11360 || shots != 339 {
		t.Fatalf("incomplete original cannon coverage: %d constructors, %d passes, %d shots", constructors, passes, shots)
	}
	t.Logf("Compared %d real cannon constructors, %d updates and %d emissions across five stages", constructors, passes, shots)
}

func TestFirstWallCannonShotsEnterPlayfield(t *testing.T) {
	for _, test := range []struct {
		kind, variant, x, offsetX int
		direction                 uint8
	}{
		{1, 0, 64, 32, 2}, {1, 1, 256, 0, 6},
		{2, 0, 16, 16, 2}, {2, 1, 272, 0, 6},
		{3, 0, 16, 26, 2}, {3, 1, 272, 0, 6},
	} {
		t.Run(fmt.Sprintf("kind-%d-variant-%d", test.kind, test.variant), func(t *testing.T) {
			state := FixedTileState{Kind: test.kind, Variant: test.variant, X: test.x, WorldY: 1000, Accumulator: 255}
			if test.kind == 3 {
				state.Phase = 23
			}
			random := NewRandomState()
			event := StepFirstTileCannon(&state, visualassets.FixedTileKind{FireRate: 4, ShotSpeed: 6}, 900, 4607, &random)
			if !event.Shot || event.ShotDirection != test.direction || event.ShotX != test.x+test.offsetX {
				t.Fatalf("wall cannon must fire from its inward muzzle: %+v", event)
			}
			projectile := DirectionalProjectile{X: int32(event.ShotX) << 16, Y: int32(event.ShotY) << 16, Direction: event.ShotDirection, Speed: event.ShotSpeed}
			before := projectile.X
			inside, err := projectile.Advance(0)
			if err != nil || !inside || test.variant == 0 && projectile.X <= before || test.variant == 1 && projectile.X >= before {
				t.Fatalf("shot moves toward the outside: %+v, inside %v, error %v", projectile, inside, err)
			}
		})
	}
}
