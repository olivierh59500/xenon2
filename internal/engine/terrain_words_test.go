package engine

import (
	"fmt"
	"testing"

	"xenon2/internal/visualassets"
)

// Keep the original pixel loop as an independent oracle for word alignment,
// clipping and immediate mutation of the map used by the gameplay callback.
func terrainPixelTouches(c *TerrainCoverage, shipX, shipY, scrollY int, mask visualassets.PlayerTerrainStencil) bool {
	for y, row := range mask.Rows {
		for x := range mask.Width {
			if row&(uint32(1)<<uint(31-x)) != 0 && c.Solid(shipX+mask.OriginOffsetX+x, scrollY+shipY+mask.OriginOffsetY+y) {
				return true
			}
		}
	}
	return false
}

func TestTerrainWordsMatchPixelStencilAtTileAndMapEdges(t *testing.T) {
	c := &TerrainCoverage{Columns: 4, Rows: 3, Map: []uint16{0, 1, 2, 3, 2, 99, 1, 0, 3, 2, 0, 1}, coverage: make(map[uint16][16]uint16)}
	var solid, checker, stripes [16]uint16
	for row := range solid {
		solid[row], stripes[row] = 0xffff, uint16(1)<<uint(row)
		checker[row] = 0xaaaa
		if row&1 != 0 {
			checker[row] = 0x5555
		}
	}
	// ID zero remains empty even when a coverage table entry exists; missing
	// nonzero IDs also remain empty, matching Solid's source behavior.
	c.coverage[0], c.coverage[1], c.coverage[2], c.coverage[3] = solid, checker, stripes, solid
	comparisons := 0
	for width := 1; width <= 32; width++ {
		for _, offset := range []int{-31, -15, -1, 0, 15} {
			mask := visualassets.PlayerTerrainStencil{Width: width, Height: 1, OriginOffsetX: offset, OriginOffsetY: -2,
				Rows: []uint32{0xffffffff, 0x80000001, 0x00aa55ff, 0, 0x1fff8000}}
			for x := -20; x <= 84; x += 3 {
				for mapY := -20; mapY <= 68; mapY += 5 {
					scroll := x - mapY
					got, want := c.Touches(x, mapY-scroll, scroll, mask), terrainPixelTouches(c, x, mapY-scroll, scroll, mask)
					if got != want {
						t.Fatalf("width%d offset%d xy%d/%d scroll%d: got%v want%v", width, offset, x, mapY-scroll, scroll, got, want)
					}
					comparisons++
				}
			}
		}
	}
	// Isolate every mask bit at every word alignment, including a third tile
	// column and masks whose metadata height differs from their row count.
	for bit := 0; bit < 32; bit++ {
		mask := visualassets.PlayerTerrainStencil{Width: 32, Height: 0, Rows: []uint32{uint32(1) << uint(31-bit)}}
		for x := -32; x < 96; x++ {
			for y := -1; y <= 48; y++ {
				if got, want := c.Touches(x, y, 0, mask), terrainPixelTouches(c, x, y, 0, mask); got != want {
					t.Fatalf("single bit%d xy%d/%d: got%v want%v", bit, x, y, got, want)
				}
				comparisons++
			}
		}
	}
	for _, width := range []int{-1, 0, 33, 40} {
		mask := visualassets.PlayerTerrainStencil{Width: width, Rows: []uint32{0xffffffff}}
		if c.Touches(15, 16, 0, mask) != terrainPixelTouches(c, 15, 16, 0, mask) {
			t.Fatalf("width%d changed the original empty/outside-word behavior", width)
		}
	}
	t.Logf("%d independent pixel/word comparisons", comparisons)
}

func TestTerrainWordsMatchOriginalFiveMapsAndLivePatchesOptional(t *testing.T) {
	comparisons := 0
	for level := 1; level <= 5; level++ {
		t.Run(fmt.Sprintf("level%d", level), func(t *testing.T) {
			data := playableOriginalWorldData(t, level)
			c, err := NewTerrainCoverage(data.Terrain)
			if err != nil {
				t.Fatal(err)
			}
			for mapY := -24; mapY <= c.Rows*16+24; mapY += 37 {
				for x := -24; x <= c.Columns*16+24; x += 11 {
					if got, want := c.Touches(x, 176, mapY-176, *data.PlayerStencil), terrainPixelTouches(c, x, 176, mapY-176, *data.PlayerStencil); got != want {
						t.Fatalf("original xy%d/%d got%v want%v", x, mapY, got, want)
					}
					comparisons++
				}
			}
			var id uint16
			pixelX, pixelY := -1, -1
			for candidate, rows := range c.coverage {
				if candidate == 0 {
					continue
				}
				for y, row := range rows {
					for x := 0; x < 16; x++ {
						if row&(uint16(1)<<uint(15-x)) != 0 {
							id, pixelX, pixelY = candidate, x, y
							break
						}
					}
					if pixelX >= 0 {
						break
					}
				}
				if pixelX >= 0 {
					break
				}
			}
			if pixelX < 0 {
				t.Fatal("original map has no covered tile")
			}
			point := visualassets.PlayerTerrainStencil{Width: 1, Height: 1, Rows: []uint32{0x80000000}}
			x, y := 160+pixelX, 1440+pixelY
			for _, tile := range []uint16{id, 0, id, 0} {
				if err := c.Patch(10, 90, visualassets.TilePatch{Columns: 1, Rows: 1, Tiles: []uint16{tile}}); err != nil {
					t.Fatal(err)
				}
				if got, want := c.Touches(x, y, 0, point), terrainPixelTouches(c, x, y, 0, point); got != want || got != (tile != 0) {
					t.Fatalf("live patch tile%d got%v want%v", tile, got, want)
				}
			}
			c.Map[90*c.Columns+10] = id
			if !c.Touches(x, y, 0, point) {
				t.Fatal("direct source map write was hidden by cached words")
			}
		})
	}
	t.Logf("%d original-map pixel/word comparisons plus immediate patches", comparisons)
}

var terrainWordsBenchmarkResult bool

func BenchmarkTerrainWordsOriginalClearStencil(b *testing.B) {
	data := playableOriginalWorldData(b, 3)
	c, err := NewTerrainCoverage(data.Terrain)
	if err != nil {
		b.Fatal(err)
	}
	for _, words := range []bool{false, true} {
		b.Run(fmt.Sprintf("words%v", words), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if words {
					terrainWordsBenchmarkResult = c.Touches(168, 176, 2815, *data.PlayerStencil)
				} else {
					terrainWordsBenchmarkResult = terrainPixelTouches(c, 168, 176, 2815, *data.PlayerStencil)
				}
			}
		})
	}
}
