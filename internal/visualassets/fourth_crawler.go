package visualassets

import (
	"encoding/binary"
	"fmt"
)

func decodeFourthCrawlerKind(data []byte, add func(int) (string, error)) (FixedSpriteKind, error) {
	var kind FixedSpriteKind
	if len(data) < 0x5700a-levelBase {
		return kind, fmt.Errorf("fourth crawler artwork is truncated")
	}
	kind = FixedSpriteKind{Kind: 3, Health: int(binary.BigEndian.Uint16(data[0x60:])), Score: 200, StrongHealth: true, Damageable: true, ActorList: "moving", CollisionMode: "sprite-prefix", Behavior: "fourth-crawler", VariantSelection: "record-variant", MotionParameters: map[string]int{"speed": int(binary.BigEndian.Uint16(data[0x64:])), "activation_chance": int(data[0x63]), "emitter_clock": int(binary.BigEndian.Uint16(data[0x7c:])), "clip_margin": 208, "draw_bottom": 220}, MotionTables: map[string][]int{}}
	for _, table := range []struct {
		name    string
		address int
	}{{"cover_x", 0x56fd2}, {"cover_world_y", 0x56fe8}} {
		values := make([]int, 11)
		for i := range 11 {
			values[i] = int(int16(binary.BigEndian.Uint16(data[table.address-levelBase+i*2:])))
			if table.name == "cover_x" {
				values[i] *= 2
			}
		}
		kind.MotionTables[table.name] = values
	}
	for variant := range 2 {
		root, cover := 0x56cf8, 0x56ffe
		if variant == 1 {
			root, cover = 0x56cd4, 0x57004
		}
		v := FixedSpriteVariant{ID: variant, ResourceTag: 224 + variant*4, OriginOffsetX: 8, OriginOffsetY: 20}
		if variant == 1 {
			v.OriginOffsetX = -8
		}
		var frames []AnimationFrame
		for i := range 6 {
			address := int(binary.BigEndian.Uint32(data[root-levelBase+i*6:]))
			name, err := add(address)
			if err != nil {
				return kind, err
			}
			frames = append(frames, AnimationFrame{Sprite: name, Duration: int(binary.BigEndian.Uint16(data[root-levelBase+i*6+4:]))})
		}
		v.Animation = ActorAnimation{Frames: frames, Static: false, Ending: "hold"}
		patch := readTilePatch(data, cover-levelBase, 1, 3)
		v.Cover = &patch
		kind.Variants = append(kind.Variants, v)
	}
	return kind, nil
}

func FourthCrawlerTileCodes(data []byte) ([]uint16, error) {
	if len(data) < 0x5700a-levelBase {
		return nil, fmt.Errorf("fourth crawler cover tiles are truncated")
	}
	var codes []uint16
	for _, address := range []int{0x56ffe, 0x57004} {
		codes = append(codes, readTilePatch(data, address-levelBase, 1, 3).Tiles...)
	}
	return codes, nil
}

func RemapFixedSpriteTiles(data *FixedSprites, ids map[uint16]uint16) error {
	for i := range data.Kinds {
		for j := range data.Kinds[i].Variants {
			if patch := data.Kinds[i].Variants[j].Cover; patch != nil {
				for n, code := range patch.Tiles {
					id, ok := ids[code]
					if !ok && code != 0 {
						return fmt.Errorf("fixed sprite cover tile is missing")
					}
					patch.Tiles[n] = id
				}
			}
		}
	}
	return nil
}
