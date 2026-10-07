package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestDemoNavigationWordCoverageMatchesSourceStencilAcrossFiveLevelsOptional(t *testing.T) {
	comparisons := 0
	for level := 1; level <= 5; level++ {
		w, err := NewWorld(playableOriginalWorldData(t, level))
		if err != nil {
			t.Fatal(err)
		}
		n := demoNavigation{}
		n.refresh(w)
		for y := 48; y < 4750; y += 127 {
			for x := 14; x <= 304; x += 7 {
				if got, want := n.touching(x, y), w.Coverage.Touches(x, y, 0, *w.Level.PlayerStencil); got != want {
					t.Fatalf("source stencil mismatch level%d xy%d/%d: %v want%v", level, x, y, got, want)
				}
				comparisons++
			}
		}
		patch := visualassets.TilePatch{Columns: 1, Rows: 1, Tiles: []uint16{0}}
		if err := w.Coverage.Patch(9, 90, patch); err != nil {
			t.Fatal(err)
		}
		n.refresh(w)
		for y := 1420; y < 1475; y++ {
			for x := 135; x < 175; x++ {
				if n.touching(x, y) != w.Coverage.Touches(x, y, 0, *w.Level.PlayerStencil) {
					t.Fatal("word cache ignored a source map patch")
				}
				comparisons++
			}
		}
	}
	t.Logf("%d exact source terrain/stencil comparisons", comparisons)
}
