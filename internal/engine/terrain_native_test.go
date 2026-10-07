package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestWorldTerrainStencilNativeAcrossFiveLevelsOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to local source comparisons")
	}
	dir := filepath.Join(filepath.Dir(root), "imported")
	common, err := os.ReadFile(filepath.Join(dir, "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	stencil, err := visualassets.DecodePlayerTerrainStencil(common)
	if err != nil {
		t.Fatal(err)
	}
	var coverage [5]*TerrainCoverage
	for i, name := range []string{"000B00E5.decoded", "00FA00FE.decoded", "02020113.decoded", "031F0159.decoded", "04820138.decoded"} {
		bytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		terrain, err := visualassets.DecodeTerrain(bytes)
		if err != nil {
			t.Fatal(err)
		}
		coverage[i], err = NewTerrainCoverage(terrain)
		if err != nil {
			t.Fatal(err)
		}
	}
	comparisons := 0
	nativeCombatRows(t, "terrain-collision-trace.csv", func(v []int64) {
		if contact := coverage[int(v[0])-1].Touches(int(v[1]), int(v[2]), int(v[3]), *stencil); contact != (v[4] != 0) {
			t.Fatalf("terrain stencil differs%v: contact=%t", v, contact)
		}
		comparisons++
	})
	if comparisons != 10455 {
		t.Fatalf("incomplete terrain comparison:%d", comparisons)
	}
}
