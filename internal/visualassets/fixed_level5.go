package visualassets

import (
	"encoding/binary"
	"fmt"
)

func decodeFifthFixedTiles(data []byte) (*FixedTiles, []uint16, error) {
	result, codes, err := decodeSmallCannonTiles(data, 2, 0x68, 0x57524-levelBase, 0x57514-levelBase, 0x57534-levelBase, 0x57536-levelBase, 8, 1)
	if err != nil {
		return nil, nil, err
	}
	if len(data) < 0x573ac-levelBase {
		return nil, nil, fmt.Errorf("fifth level mechanism data is truncated")
	}
	read := func(address, columns, rows int) TilePatch {
		patch := readTilePatch(data, address-levelBase, columns, rows)
		codes = append(codes, patch.Tiles...)
		return patch
	}
	patch := func(columns, rows int, values ...uint16) TilePatch {
		codes = append(codes, values...)
		return TilePatch{Columns: columns, Rows: rows, Tiles: append([]uint16(nil), values...)}
	}
	barrier := FixedTileKind{Kind: 1, Behavior: "fifth-tile-barrier", Health: int(binary.BigEndian.Uint16(data[0x5e:])), Mode: "cycle", FrameDuration: 1}
	initial := []uint16{0x8ad3, 0, 0, 0x8add}
	for index, definition := range []struct {
		tag, x, width, table int
		box                  CollisionBox
	}{
		{296, 0, 1, 0x56c94, CollisionBox{X: 3, Y: 3, Width: 11, Height: 11}},
		{304, 16, 2, 0x56c30, CollisionBox{X: 0, Y: 2, Width: 33, Height: 13}},
		{300, 48, 1, 0x56c9c, CollisionBox{X: 3, Y: 3, Width: 11, Height: 11}},
	} {
		part := FixedTilePart{ResourceTag: definition.tag, OffsetX: definition.x, Collision: definition.box, Damageable: index != 1, Initial: patch(definition.width, 1, initial[definition.x/16:definition.x/16+definition.width]...)}
		for frame := 0; frame < 4; frame++ {
			part.Frames = append(part.Frames, read(definition.table+frame*definition.width*2, definition.width, 1))
		}
		if index != 1 {
			part.Flash = read(0x56e3c+(index/2)*2, 1, 1)
		}
		barrier.Parts = append(barrier.Parts, part)
	}
	barrier.Variants = []FixedTileVariant{{ID: 0, OriginOffsetX: -8, OriginOffsetY: -8, Initial: patch(4, 1, initial...), Destroyed: patch(4, 1, 0x87b5, 0, 0, 0x87ab)}}
	result.Kinds = append(result.Kinds, barrier)
	turret := FixedTileKind{Kind: 3, Behavior: "fifth-aiming-tile", Health: int(binary.BigEndian.Uint16(data[0x62:])), Collision: CollisionBox{X: 4, Y: 4, Width: 25, Height: 25}, Mode: "aiming-cycle", FireRate: int(data[0x65]), ShotSpeed: int(binary.BigEndian.Uint16(data[0x66:]))}
	frames := make([]TilePatch, 8)
	for i := range frames {
		frames[i] = read(0x5711e+i*8, 2, 2)
		turret.DirectionalOffsets[i] = [2]int{int(int16(binary.BigEndian.Uint16(data[0x572aa-levelBase+i*4:]))), int(int16(binary.BigEndian.Uint16(data[0x572ac-levelBase+i*4:])))}
	}
	for i := range frames {
		variantFrames := make([]TilePatch, len(frames))
		for j := range frames {
			variantFrames[j] = patch(2, 2, frames[j].Tiles...)
		}
		turret.Variants = append(turret.Variants, FixedTileVariant{ID: i, ResourceTag: 244, OriginOffsetX: -8, OriginOffsetY: -8, Initial: patch(2, 2, frames[i].Tiles...), Frames: variantFrames, Destroyed: patch(2, 2, 0, 0, 0, 0)})
	}
	turret.InitialChanges = []ConditionalTileReplacement{{ColumnOffset: -1, Before: 0x867d, After: patch(1, 2, 0x80d7, 0x8187)}, {ColumnOffset: 2, Before: 0x8687, After: patch(1, 2, 0x8287, 0x8339)}}
	turret.DestroyedChanges = []ConditionalTileReplacement{{ColumnOffset: -1, Before: 0x80d7, After: patch(1, 2, 0x867d, 0x871b)}, {ColumnOffset: 2, Before: 0x8287, After: patch(1, 2, 0x8687, 0x8725)}}
	for _, changes := range [][]ConditionalTileReplacement{turret.InitialChanges, turret.DestroyedChanges} {
		for _, change := range changes {
			codes = append(codes, change.Before)
		}
	}
	result.Kinds = append(result.Kinds, turret)
	radial := FixedTileKind{Kind: 4, Behavior: "fifth-radial-tile", Health: int(binary.BigEndian.Uint16(data[0x60:])), Collision: CollisionBox{X: 4, Y: 4, Width: 25, Height: 25}, Mode: "radial-cycle", FireRate: int(data[0x73]), ShotSpeed: int(binary.BigEndian.Uint16(data[0x74:]))}
	variant := FixedTileVariant{ID: 0, ResourceTag: 248, OriginOffsetX: -8, OriginOffsetY: -8, Initial: read(0x56ea2, 2, 2), Destroyed: patch(2, 2, 0, 0, 0, 0)}
	for i := 0; i < 8; i++ {
		variant.Frames = append(variant.Frames, read(0x56ea2+i*8, 2, 2))
	}
	radial.Variants = []FixedTileVariant{variant}
	radial.InitialChanges = []ConditionalTileReplacement{{ColumnOffset: -1, Before: 0x8669, After: patch(1, 2, 0x00c7, 0x0177)}, {ColumnOffset: 2, Before: 0x8673, After: patch(1, 2, 0x00c7, 0x0177)}}
	radial.DestroyedChanges = []ConditionalTileReplacement{{ColumnOffset: -1, Before: 0x00c7, After: patch(1, 2, 0x8669, 0x8707)}, {ColumnOffset: 2, Before: 0x00c7, After: patch(1, 2, 0x8673, 0x8711)}}
	for _, changes := range [][]ConditionalTileReplacement{radial.InitialChanges, radial.DestroyedChanges} {
		for _, change := range changes {
			codes = append(codes, change.Before)
		}
	}
	result.Kinds = append(result.Kinds, radial)
	// The persistent encounter shares the aiming turret's graphics and update
	// rules, but records its destruction for checkpoint re-entry.
	persistent := turret
	persistent.Kind, persistent.Behavior = 9, "fifth-persistent-aiming-tile"
	persistent.Variants = make([]FixedTileVariant, len(turret.Variants))
	for i, source := range turret.Variants {
		variant := source
		variant.Initial.Tiles = append([]uint16(nil), source.Initial.Tiles...)
		variant.Destroyed.Tiles = append([]uint16(nil), source.Destroyed.Tiles...)
		variant.Frames = make([]TilePatch, len(source.Frames))
		for j, frame := range source.Frames {
			variant.Frames[j] = frame
			variant.Frames[j].Tiles = append([]uint16(nil), frame.Tiles...)
		}
		persistent.Variants[i] = variant
	}
	persistent.InitialChanges = cloneFifthTileChanges(turret.InitialChanges)
	persistent.DestroyedChanges = cloneFifthTileChanges(turret.DestroyedChanges)
	result.Kinds = append(result.Kinds, persistent)
	return result, codes, nil
}

func cloneFifthTileChanges(changes []ConditionalTileReplacement) []ConditionalTileReplacement {
	result := append([]ConditionalTileReplacement(nil), changes...)
	for i := range result {
		result[i].After.Tiles = append([]uint16(nil), changes[i].After.Tiles...)
	}
	return result
}

func decodeFifthAimingTileShots(level []byte, add func(int) (string, error)) ([]string, error) {
	var sprites []string
	for i := 0; i < 8; i++ {
		root := int(binary.BigEndian.Uint32(level[0x572ca-levelBase+i*4:]))
		name, err := add(root)
		if err != nil {
			return nil, err
		}
		sprites = append(sprites, name)
	}
	return sprites, nil
}
