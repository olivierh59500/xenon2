package visualassets

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

// PlayerTerrainStencil is the original collision footprint, independently of
// the visible banking image. Rows are MSB-first thirty-pixel masks.
type PlayerTerrainStencil struct {
	OriginOffsetX int          `json:"origin_offset_x"`
	OriginOffsetY int          `json:"origin_offset_y"`
	Width         int          `json:"width"`
	Height        int          `json:"height"`
	Rows          []uint32     `json:"rows"`
	Image         *image.NRGBA `json:"-"`
}

func (s *PlayerTerrainStencil) Contains(x, y int) bool {
	return x >= 0 && y >= 0 && x < s.Width && y < s.Height && y < len(s.Rows) && s.Rows[y]&(uint32(1)<<uint(31-x)) != 0
}

// DecodePlayerTerrainStencil verifies all sixteen native word-alignment copies
// against one unshifted mask, then exports only the logical collision footprint.
func DecodePlayerTerrainStencil(common []byte) (*PlayerTerrainStencil, error) {
	const table, height = 0x3ae96, 25
	if len(common) < table+64 {
		return nil, fmt.Errorf("player terrain stencil table is truncated")
	}
	readRows := func(field int) ([]uint64, error) {
		start := uint64(binary.BigEndian.Uint32(common[field:]))
		if start+height*6 > uint64(len(common)) {
			return nil, fmt.Errorf("player terrain stencil pixels are outside source")
		}
		rows := make([]uint64, height)
		for y := range height {
			for x := range 6 {
				rows[y] = rows[y]<<8 | uint64(common[int(start)+y*6+x])
			}
		}
		return rows, nil
	}
	base, err := readRows(table)
	if err != nil {
		return nil, err
	}
	for shift := 1; shift < 16; shift++ {
		rows, err := readRows(table + shift*4)
		if err != nil {
			return nil, err
		}
		for y, row := range rows {
			if row != base[y]>>uint(shift) {
				return nil, fmt.Errorf("player terrain stencil alignment %d row %d differs", shift, y)
			}
		}
	}
	stencil := &PlayerTerrainStencil{OriginOffsetX: -14, OriginOffsetY: -14, Width: 30, Height: height, Rows: make([]uint32, height), Image: image.NewNRGBA(image.Rect(0, 0, 30, height))}
	for y, row := range base {
		if row&0x3ffff != 0 {
			return nil, fmt.Errorf("player terrain stencil exceeds thirty pixels")
		}
		stencil.Rows[y] = uint32(row >> 16)
		for x := range stencil.Width {
			if stencil.Contains(x, y) {
				stencil.Image.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}
	return stencil, nil
}
