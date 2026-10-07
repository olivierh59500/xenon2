package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestDemoNavigationTouchCacheObservesOriginalMapMutationsOptional(t *testing.T) {
	for level := 1; level <= 5; level++ {
		w, err := NewWorld(playableOriginalWorldData(t, level))
		if err != nil {
			t.Fatal(err)
		}
		n := demoNavigation{}
		n.refresh(w)
		n.cacheTouchWindow(1440)
		check := func() {
			for y := 1390; y < 1690; y += 7 {
				for x := 14; x <= 304; x += 5 {
					want := n.touchingUncached(x, y)
					for range 2 {
						if n.touching(x, y) != want {
							t.Fatalf("level%d cached stencil differs at%d/%d", level, x, y)
						}
					}
				}
			}
		}
		check()
		var replacement uint16
		for id := range w.Coverage.coverage {
			if id != w.Coverage.Map[90*20+9] {
				replacement = id
				break
			}
		}
		if err := w.Coverage.Patch(9, 90, visualassets.TilePatch{Columns: 1, Rows: 1, Tiles: []uint16{replacement}}); err != nil {
			t.Fatal(err)
		}
		if !n.refresh(w) {
			t.Fatal("test patch did not change the terrain map")
		}
		check()
		// A direct callback write and moving the cached window must also
		// invalidate learned results, including previously empty positions.
		w.Coverage.Map[90*20+9] = w.Coverage.Map[91*20+9]
		n.refresh(w)
		n.cacheTouchWindow(1632)
		check()
	}
}
