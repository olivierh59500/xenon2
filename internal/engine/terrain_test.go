package engine

import (
	"image"
	"image/color"
	"reflect"
	"testing"

	"xenon2/internal/visualassets"
)

func TestTerrainCoverageUsesMasksAndOpaqueBlack(t *testing.T) {
	atlas := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	atlas.SetNRGBA(16+7, 4, color.NRGBA{A: 255})
	terrain := &visualassets.Terrain{Columns: 2, Rows: 1, TileSize: 16, Map: []uint16{1, 2}, Atlas: atlas,
		Tiles: []visualassets.Tile{{ID: 1}, {ID: 2, X: 16, Masked: true}}}
	c, err := NewTerrainCoverage(terrain)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Solid(7, 4) || !c.Solid(23, 4) || c.Solid(22, 4) || c.Solid(-1, 4) {
		t.Fatal("coverage must preserve original opaque and masked tile semantics")
	}
	mask := visualassets.PlayerTerrainStencil{Width: 1, Height: 1, Rows: []uint32{0x80000000}}
	if !c.Touches(23, 4, 0, mask) || c.Touches(22, 4, 0, mask) {
		t.Fatal("ship stencil contact differs at a single mask pixel")
	}
	before := append([]uint16(nil), c.Map...)
	if err := c.Patch(0, 0, visualassets.TilePatch{Columns: 2, Rows: 1, Tiles: []uint16{0, 99}}); err == nil || !reflect.DeepEqual(before, c.Map) {
		t.Fatal("invalid patch must fail without changing earlier cells")
	}
	if err := c.Patch(0, 0, visualassets.TilePatch{Columns: 1, Rows: 1, Tiles: []uint16{0}}); err != nil || c.Solid(7, 4) {
		t.Fatal("destroyed terrain did not clear collision coverage")
	}
	if terrain.Map[0] != 1 {
		t.Fatal("mutable collision map must not change reusable source data")
	}
}

func TestTerrainRewindRestoresPositionWithoutDamage(t *testing.T) {
	s := NewTerrainRewind(100, 80, 176)
	s.Record(99, 83, 176)
	s.Timer = 1
	p := PlayerMotionState{X: 86, Y: 176}
	handled, crushed := s.Advance(&p, 98, 1, true)
	if !handled || crushed || p.X != 80 || p.Y != 176 || p.ScrollStep != -1 || s.Timer != -1 {
		t.Fatalf("rewind got %+v timer=%d handled=%v crushed=%v", p, s.Timer, handled, crushed)
	}
	handled, crushed = s.Advance(&p, 99, 1, false)
	if handled || crushed || s.Timer != 0 {
		t.Fatal("clear terrain should resume ordinary movement immediately")
	}
}

func TestTerrainRewindCrushesOnlyAfterSeventeenReversePasses(t *testing.T) {
	s := NewTerrainRewind(100, 80, 100)
	s.Timer = 1
	p := PlayerMotionState{}
	for pass := range 17 {
		if _, crushed := s.Advance(&p, 100, 1, true); crushed {
			t.Fatalf("crushed too early on rewind pass %d", pass)
		}
	}
	if _, crushed := s.Advance(&p, 100, 1, true); !crushed {
		t.Fatal("persistently trapped ship must be crushed")
	}
}
