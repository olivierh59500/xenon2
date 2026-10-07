package engine

import (
	"fmt"
	"image"

	"xenon2/internal/visualassets"
)

// TerrainCoverage caches tile coverage as one sixteen-bit row per tile. Masked
// tiles use the recovered coverage alpha; opaque tiles are solid in the source
// collision routine, including black pixels. Map changes remain inexpensive.
type TerrainCoverage struct {
	Columns, Rows int
	Map           []uint16
	coverage      map[uint16][16]uint16
}

func NewTerrainCoverage(terrain *visualassets.Terrain) (*TerrainCoverage, error) {
	if terrain == nil || terrain.TileSize != 16 || len(terrain.Map) != terrain.Columns*terrain.Rows || terrain.Atlas == nil {
		return nil, fmt.Errorf("terrain coverage needs a complete tile atlas")
	}
	c := &TerrainCoverage{Columns: terrain.Columns, Rows: terrain.Rows, Map: append([]uint16(nil), terrain.Map...), coverage: make(map[uint16][16]uint16, len(terrain.Tiles))}
	for _, tile := range terrain.Tiles {
		if !image.Rect(tile.X, tile.Y, tile.X+16, tile.Y+16).In(terrain.Atlas.Bounds()) {
			return nil, fmt.Errorf("terrain tile %d leaves its atlas", tile.ID)
		}
		var rows [16]uint16
		for y := range 16 {
			if !tile.Masked {
				rows[y] = 0xffff
				continue
			}
			for x := range 16 {
				if terrain.Atlas.NRGBAAt(tile.X+x, tile.Y+y).A != 0 {
					rows[y] |= 1 << uint(15-x)
				}
			}
		}
		c.coverage[tile.ID] = rows
	}
	return c, nil
}

func (c *TerrainCoverage) Solid(x, mapY int) bool {
	if x < 0 || x >= c.Columns*16 || mapY < 0 || mapY >= c.Rows*16 {
		return false
	}
	id := c.Map[(mapY/16)*c.Columns+x/16]
	if id == 0 {
		return false
	}
	return c.coverage[id][mapY%16]&(1<<uint(15-x%16)) != 0
}

// Touches tests the source stencil against map coverage without allocating.
func (c *TerrainCoverage) Touches(shipX, shipY, scrollY int, mask visualassets.PlayerTerrainStencil) bool {
	for y, row := range mask.Rows {
		if row == 0 {
			continue
		}
		for x := range mask.Width {
			if row&(uint32(1)<<uint(31-x)) != 0 && c.Solid(shipX+mask.OriginOffsetX+x, scrollY+shipY+mask.OriginOffsetY+y) {
				return true
			}
		}
	}
	return false
}

// Patch changes named atlas IDs in the mutable map. It validates the entire
// patch before mutation, preserving the map when a resource is malformed.
func (c *TerrainCoverage) Patch(x, y int, patch visualassets.TilePatch) error {
	if x < 0 || y < 0 || patch.Columns < 1 || patch.Rows < 1 || x+patch.Columns > c.Columns || y+patch.Rows > c.Rows || len(patch.Tiles) != patch.Columns*patch.Rows {
		return fmt.Errorf("terrain patch leaves the map")
	}
	for _, id := range patch.Tiles {
		if id != 0 {
			if _, ok := c.coverage[id]; !ok {
				return fmt.Errorf("terrain patch references unknown tile %d", id)
			}
		}
	}
	for row := range patch.Rows {
		copy(c.Map[(y+row)*c.Columns+x:][:patch.Columns], patch.Tiles[row*patch.Columns:][:patch.Columns])
	}
	return nil
}

type ShipHistory struct {
	ScrollY, X, Y int
}

// TerrainRewind keeps the seventeen positions used when scenery pushes the ship
// back. The timer remains negative while contact persists; after seventeen
// failed reverse passes the original crushes the ship.
type TerrainRewind struct {
	History [17]ShipHistory
	Timer   int
}

func NewTerrainRewind(scrollY, x, y int) TerrainRewind {
	s := TerrainRewind{}
	for i := range s.History {
		s.History[i] = ShipHistory{ScrollY: scrollY, X: x, Y: y}
	}
	return s
}

func (s *TerrainRewind) Record(scrollY, x, y int) {
	copy(s.History[1:], s.History[:16])
	s.History[0] = ShipHistory{ScrollY: scrollY, X: x, Y: y}
}

// Advance returns whether the rewind handled this pass and whether the ship
// was crushed. The ordinary motion routine runs only when handled is false.
func (s *TerrainRewind) Advance(player *PlayerMotionState, scrollY, baseStep int, touching bool) (handled, crushed bool) {
	if s.Timer == 0 {
		return false, false
	}
	if s.Timer < 0 {
		if !touching {
			s.Timer = 0
			return false, false
		}
		if s.Timer == -17 {
			return true, true
		}
	}
	s.Timer--
	position := s.History[1]
	player.X = position.X
	player.Y = position.ScrollY - scrollY + position.Y
	player.ScrollStep = baseStep
	if player.Y > 176 {
		player.ScrollStep -= player.Y - 176
		player.Y = 176
	} else if player.Y < 16 {
		player.ScrollStep -= player.Y - 16
		player.Y = 16
	}
	copy(s.History[:16], s.History[1:])
	if s.Timer == 0 {
		s.Timer = -1
	}
	return true, false
}
