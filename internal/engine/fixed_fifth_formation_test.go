package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestFifthFormationNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local fifth formation reference not supplied")
	}
	level, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	common, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(level)
	if err != nil {
		t.Fatal(err)
	}
	resources, err := visualassets.DecodeFixedSprites(5, level, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := visualassets.DecodePaths(level, common)
	if err != nil {
		t.Fatal(err)
	}
	art := *resources.FifthFormation
	var states [10]FifthFormationState
	passes := 0
	nativeCombatRows(t, "fixed-fifth-formation.csv", func(v []int64) {
		variant, member, frame := int(v[0]), int(v[1]), int(v[2])
		if frame == -1 {
			states[member], err = NewFifthFormation(art, variant, member, 2000-variant*300)
			if err != nil {
				t.Fatal(err)
			}
		}
		s := &states[member]
		shots := false
		if frame >= 0 {
			shots, err = s.Advance(art, int(v[3]), &paths.SineTable, nil)
			if err != nil {
				t.Fatal(err)
			}
		}
		angle := uint32(s.Motion.AngleFixed)
		angle = angle<<16 | angle>>16
		if uint32(s.Motion.X) != uint32(v[4]) || uint32(s.Motion.Y) != uint32(v[5]) || angle != uint32(v[6]) || uint32(s.Motion.AngularVelocity) != uint32(v[7]) || int16(s.Motion.AngularAcceleration) != int16(v[8]) || s.Motion.Remaining != int(v[9]) || s.Motion.Budget != int(v[10]) || s.FireAccumulator != uint8(v[11]) || s.Sprite != resources.Atlas.SourceSpriteNames[int(v[12])] || s.Duration != int(v[13]) || s.Motion.Active != (v[15] == 220) || shots != (v[16] == 8) {
			t.Fatalf("variant%d member%d frame%d Go%+v native%v", variant, member, frame, s, v)
		}
		passes++
	})
	if passes != 27030 {
		t.Fatalf("incomplete native comparison: %d", passes)
	}
	t.Logf("Compared %d original formation states across all three paths and ten members.", passes)
}
