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

func TestPlayerTerrainLoopNativeHistoryOptional(t *testing.T) {
	data := playableOriginalWorldData(t, 5)
	coverage, err := NewTerrainCoverage(data.Terrain)
	if err != nil {
		t.Fatal(err)
	}
	player := PlayerMotionState{X: 157, Y: 173}
	rewind := NewTerrainRewind(4608, 160, 176)
	scroll, maximum := 4607, 4608
	passes := 0
	nativeCombatRows(t, "player-terrain-loop.csv", func(v []int64) {
		input := MotionInput{Up: true}
		if passes >= 80 {
			input.Left, input.Right = passes%80 < 40, passes%80 >= 40
		}
		touching := coverage.Touches(player.X, player.Y, scroll, *data.PlayerStencil)
		handled, crushed := rewind.Advance(&player, scroll, 1, touching)
		if crushed {
			t.Fatal("source comparison crushed unexpectedly")
		}
		if !handled {
			player.Advance(input, MotionContext{ScrollY: scroll, VisitedScrollY: maximum, BaseScrollStep: 1})
			rewind.Record(scroll, player.X, player.Y)
			if coverage.Touches(player.X, player.Y, scroll, *data.PlayerStencil) {
				rewind.Timer, player.Inertia = 1, 0
				maximum = max(maximum, scroll+16)
			}
		}
		scroll = min(maximum, scroll-player.ScrollStep)
		maximum = min(maximum, scroll+16)
		contact := coverage.Touches(player.X, player.Y, scroll, *data.PlayerStencil)
		if player.X != int(v[1]) || player.Y != int(v[2]) || scroll != int(v[3]) || maximum != int(v[4]) || rewind.Timer != int(v[5]) || player.ScrollStep != int(v[6]) || contact != (v[7] != 0) {
			t.Fatalf("source terrain pass%d differs: player%+v scroll%d maximum%d rewind%d contact%v native%v", passes, player, scroll, maximum, rewind.Timer, contact, v)
		}
		passes++
	})
	if passes != 180 {
		t.Fatalf("incomplete player-terrain loop comparison: %d", passes)
	}
}
