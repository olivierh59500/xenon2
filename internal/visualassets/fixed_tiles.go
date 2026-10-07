package visualassets

import (
	"encoding/binary"
	"fmt"
)

type TilePatch struct {
	Columns int      `json:"columns"`
	Rows    int      `json:"rows"`
	Tiles   []uint16 `json:"tiles"`
}

type FixedTileVariant struct {
	ID            int         `json:"id"`
	OriginOffsetX int         `json:"origin_offset_x"`
	OriginOffsetY int         `json:"origin_offset_y"`
	Initial       TilePatch   `json:"initial"`
	Frames        []TilePatch `json:"frames"`
	Destroyed     TilePatch   `json:"destroyed"`
}

type FixedTileKind struct {
	Kind          int                `json:"kind"`
	Health        int                `json:"health"`
	Collision     CollisionBox       `json:"collision"`
	Mode          string             `json:"mode"`
	FrameDuration int                `json:"frame_duration"`
	Variants      []FixedTileVariant `json:"variants"`
}

type FixedTiles struct {
	Kinds []FixedTileKind `json:"kinds"`
}

// DecodeFixedTiles covers the fixed terrain formats whose native frame tables
// have been verified. Other level-specific mechanisms remain separate exports.
func DecodeFixedTiles(levelNumber int, data []byte) (*FixedTiles, []uint16, error) {
	switch levelNumber {
	case 1:
		return DecodeFirstLevelFixedTiles(data)
	case 2:
		return decodeSecondLevelFixedTiles(data)
	case 3:
		return decodeSmallCannonTiles(data, 4, 0x5e, 0x56eda-levelBase, 0x56ee8-levelBase, 0x56ef6-levelBase, 0x56ef8-levelBase, 7, 2)
	case 4:
		return decodeFourthLevelFixedTiles(data)
	case 5:
		return decodeSmallCannonTiles(data, 2, 0x68, 0x57524-levelBase, 0x57514-levelBase, 0x57534-levelBase, 0x57536-levelBase, 8, 1)
	default:
		return nil, nil, nil
	}
}

func decodeSmallCannonTiles(data []byte, kindID, health, frames0, frames1, dead0, dead1, frameCount, duration int) (*FixedTiles, []uint16, error) {
	limit := max(max(frames0, frames1)+frameCount*2, max(dead0, dead1)+2)
	if len(data) < limit {
		return nil, nil, fmt.Errorf("small cannon tile data is truncated")
	}
	result := &FixedTiles{}
	codes := make([]uint16, 0)
	read := func(start int) TilePatch {
		patch := readTilePatch(data, start, 1, 1)
		codes = append(codes, patch.Tiles...)
		return patch
	}
	cannon := FixedTileKind{Kind: kindID, Health: int(binary.BigEndian.Uint16(data[health:])), Collision: CollisionBox{X: 2, Y: 2, Width: 13, Height: 13}, Mode: "triggered-cycle", FrameDuration: duration}
	for variant := range 2 {
		start, dead := frames0, dead0
		if variant == 1 {
			start, dead = frames1, dead1
		}
		v := FixedTileVariant{ID: variant, OriginOffsetX: -8, OriginOffsetY: -8, Initial: read(start), Destroyed: read(dead)}
		for frame := range frameCount {
			v.Frames = append(v.Frames, read(start+frame*2))
		}
		cannon.Variants = append(cannon.Variants, v)
	}
	result.Kinds = append(result.Kinds, cannon)
	return result, codes, nil
}

func decodeSecondLevelFixedTiles(data []byte) (*FixedTiles, []uint16, error) {
	if len(data) < 0xc02 {
		return nil, nil, fmt.Errorf("second level fixed tile data is truncated")
	}
	result := &FixedTiles{}
	codes := make([]uint16, 0)
	read := func(start, columns, rows int) TilePatch {
		patch := readTilePatch(data, start, columns, rows)
		codes = append(codes, patch.Tiles...)
		return patch
	}
	cannon := FixedTileKind{Kind: 1, Health: int(binary.BigEndian.Uint16(data[0x4e:])), Collision: CollisionBox{X: -8, Y: 4, Width: 21, Height: 9}, Mode: "triggered-cycle", FrameDuration: 1}
	for variant := range 2 {
		start, dead := 0x559e6-levelBase, 0x559c6-levelBase
		if variant == 1 {
			start, dead = 0x559c8-levelBase, 0x559c4-levelBase
		}
		v := FixedTileVariant{ID: variant, OriginOffsetX: -8, OriginOffsetY: -8, Initial: read(start, 1, 1), Destroyed: read(dead, 1, 1)}
		for frame := range 15 {
			v.Frames = append(v.Frames, read(start+frame*2, 1, 1))
		}
		cannon.Variants = append(cannon.Variants, v)
	}
	result.Kinds = append(result.Kinds, cannon)
	hatch := FixedTileKind{Kind: 2, Mode: "screen-triggered-hatch", FrameDuration: 1}
	for variant := range 2 {
		table, initial := 0x5583c-levelBase, 0x558b8-levelBase
		if variant == 1 {
			table, initial = 0x557e8-levelBase, 0x55890-levelBase
		}
		v := FixedTileVariant{ID: variant, OriginOffsetX: -8, OriginOffsetY: -8, Initial: read(initial, 2, 2)}
		for frame := range 21 {
			start, err := offset(data, table+frame*4, 8)
			if err != nil {
				return nil, nil, err
			}
			v.Frames = append(v.Frames, read(start, 2, 2))
		}
		v.Destroyed = v.Frames[len(v.Frames)-1]
		v.Destroyed.Tiles = append([]uint16(nil), v.Destroyed.Tiles...)
		hatch.Variants = append(hatch.Variants, v)
	}
	result.Kinds = append(result.Kinds, hatch)
	return result, codes, nil
}

func decodeFourthLevelFixedTiles(data []byte) (*FixedTiles, []uint16, error) {
	if len(data) < 0x2446 {
		return nil, nil, fmt.Errorf("fourth level fixed tile data is truncated")
	}
	result := &FixedTiles{}
	codes := make([]uint16, 0)
	read := func(start, columns, rows int) TilePatch {
		patch := readTilePatch(data, start, columns, rows)
		codes = append(codes, patch.Tiles...)
		return patch
	}
	cannon := FixedTileKind{Kind: 1, Health: int(binary.BigEndian.Uint16(data[0x66:])), Collision: CollisionBox{X: 2, Y: 2, Width: 29, Height: 29}, Mode: "triggered-cycle", FrameDuration: 1}
	for variant := range 2 {
		start, dead := 0x57190-levelBase, 0x5723e-levelBase
		if variant == 1 {
			start, dead = 0x57160-levelBase, 0x57236-levelBase
		}
		v := FixedTileVariant{ID: variant, OriginOffsetX: -8, OriginOffsetY: -8, Initial: read(start, 2, 2), Destroyed: read(dead, 2, 2)}
		for frame := range 6 {
			v.Frames = append(v.Frames, read(start+frame*8, 2, 2))
		}
		cannon.Variants = append(cannon.Variants, v)
	}
	result.Kinds = append(result.Kinds, cannon)
	return result, codes, nil
}

func readTilePatch(data []byte, start, columns, rows int) TilePatch {
	patch := TilePatch{Columns: columns, Rows: rows, Tiles: make([]uint16, columns*rows)}
	for i := range patch.Tiles {
		patch.Tiles[i] = binary.BigEndian.Uint16(data[start+i*2:])
	}
	return patch
}

// DecodeFirstLevelFixedTiles exports the verified mutable-tile encounter tables.
// The returned codes are translated into stable atlas IDs by RemapFixedTiles
// before serialization; they are never interpreted as native memory pointers.
func DecodeFirstLevelFixedTiles(data []byte) (*FixedTiles, []uint16, error) {
	if len(data) < 0xdde {
		return nil, nil, fmt.Errorf("fixed tile tables are truncated")
	}
	result := &FixedTiles{}
	codes := make([]uint16, 0)
	read := func(start, columns, rows int) TilePatch {
		patch := TilePatch{Columns: columns, Rows: rows, Tiles: make([]uint16, columns*rows)}
		for i := range patch.Tiles {
			patch.Tiles[i] = binary.BigEndian.Uint16(data[start+i*2:])
			codes = append(codes, patch.Tiles[i])
		}
		return patch
	}
	for _, table := range []struct {
		kind, health, initial0, initial1, frames0, frames1, dead0, dead1, frameRows, frameCount, duration int
		mode                                                                                              string
	}{
		{1, 0x5c, 0x5568c - levelBase, 0x556ac - levelBase, 0x5568c - levelBase, 0x556ac - levelBase, 0x556d4 - levelBase, 0x556cc - levelBase, 2, 4, 1, "cycle"},
		{2, 0x5a, 0x553d2 - levelBase, 0x553b2 - levelBase, 0x553d2 - levelBase, 0x553b2 - levelBase, 0x553fa - levelBase, 0x553f2 - levelBase, 2, 4, 1, "cycle"},
		{3, 0x5e, 0x55ad4 - levelBase, 0x55adc - levelBase, 0x554fa - levelBase, 0x5554a - levelBase, 0x5559a - levelBase, 0x5559a - levelBase, 1, 20, 2, "triggered-cycle"},
	} {
		kind := FixedTileKind{Kind: table.kind, Health: int(binary.BigEndian.Uint16(data[table.health:])), Collision: CollisionBox{X: 4, Y: 4, Width: 25, Height: 25}, Mode: table.mode, FrameDuration: table.duration}
		if table.kind == 3 {
			kind.Collision = CollisionBox{X: 8, Y: 8, Width: 17, Height: 17}
		}
		for variant := range 2 {
			initial, frames, dead := table.initial0, table.frames0, table.dead0
			if variant == 1 {
				initial, frames, dead = table.initial1, table.frames1, table.dead1
			}
			v := FixedTileVariant{ID: variant, OriginOffsetX: -8, OriginOffsetY: -8, Initial: read(initial, 2, 2), Destroyed: read(dead, 2, 2)}
			if table.kind == 2 && variant == 0 {
				v.OriginOffsetX = -24
			}
			for frame := range table.frameCount {
				v.Frames = append(v.Frames, read(frames+frame*2*2*table.frameRows, 2, table.frameRows))
			}
			kind.Variants = append(kind.Variants, v)
		}
		result.Kinds = append(result.Kinds, kind)
	}
	return result, codes, nil
}

func RemapFixedTiles(fixed *FixedTiles, ids map[uint16]uint16) error {
	convert := func(patch *TilePatch) error {
		for i, code := range patch.Tiles {
			id, ok := ids[code]
			if !ok && code != 0 {
				return fmt.Errorf("fixed tile frame is missing from the atlas")
			}
			patch.Tiles[i] = id
		}
		return nil
	}
	for i := range fixed.Kinds {
		for j := range fixed.Kinds[i].Variants {
			variant := &fixed.Kinds[i].Variants[j]
			if err := convert(&variant.Initial); err != nil {
				return err
			}
			if err := convert(&variant.Destroyed); err != nil {
				return err
			}
			for k := range variant.Frames {
				if err := convert(&variant.Frames[k]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
