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

func TestTerrainRewindNativeHistoryAndCrushingOptional(t *testing.T) {
	cases := 0
	nativeCombatRows(t, "terrain-rewind-trace.csv", func(v []int64) {
		state := NewTerrainRewind(int(v[4]), int(v[5]), int(v[6]))
		state.Timer = int(v[0])
		player := PlayerMotionState{ScrollStep: int(v[3])}
		handled, crushed := state.Advance(&player, int(v[2]), int(v[3]), v[1] != 0)
		if !handled || crushed != (v[11] != 0) || state.Timer != int(v[7]) || !crushed && (player.X != int(v[8]) || player.Y != int(v[9]) || player.ScrollStep != int(v[10])) {
			t.Fatalf("rewind differs%v: %+v timer=%d handled=%t crushed=%t", v, player, state.Timer, handled, crushed)
		}
		cases++
	})
	if cases != 24 {
		t.Fatalf("incomplete rewind comparisons:%d", cases)
	}
}
