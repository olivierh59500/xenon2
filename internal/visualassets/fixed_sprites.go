package visualassets

import (
	"encoding/binary"
	"fmt"
)

type FixedSpriteVariant struct {
	ID        int            `json:"id"`
	Animation ActorAnimation `json:"animation"`
}

type FixedSpriteKind struct {
	Kind             int                  `json:"kind"`
	Health           int                  `json:"health,omitempty"`
	Score            int                  `json:"score,omitempty"`
	VariantSelection string               `json:"variant_selection"`
	Variants         []FixedSpriteVariant `json:"variants"`
}

type FixedSprites struct {
	Kinds []FixedSpriteKind `json:"kinds"`
	Atlas SpriteAtlas       `json:"atlas"`
}

// DecodeFixedSprites exports the checked ordinary sprite animations used by
// fixed encounter streams. Boss-specific compound mechanisms are separate.
func DecodeFixedSprites(levelNumber int, level []byte, palette [16][4]uint8) (*FixedSprites, error) {
	type descriptor struct {
		kind, health, score, root0, root1 int
		selection                         string
	}
	var descriptors []descriptor
	switch levelNumber {
	case 1:
		descriptors = []descriptor{{4, 0x60, 250, 0x551d4, 0x55212, "record-variant"}}
	case 2:
		descriptors = []descriptor{{3, 0x50, 200, 0x55cd0, 0x55d12, "record-variant"}}
	case 3:
		descriptors = []descriptor{{2, 0x4a, 300, 0x567ce, 0x5681c, "initial-x-side"}}
	case 4:
		descriptors = []descriptor{{5, 0, 0, 0x56a74, 0x56a98, "record-variant"}}
	case 5:
		descriptors = []descriptor{{8, 0x48, 250, 0x55516, 0x5553a, "record-variant"}}
	default:
		return nil, fmt.Errorf("invalid gameplay level %d", levelNumber)
	}
	result := &FixedSprites{}
	images := make([]*Sprite, 0)
	names := map[int]string{}
	add := func(address int) (string, error) {
		if name, ok := names[address]; ok {
			return name, nil
		}
		name := fmt.Sprintf("fixed-image-%03d", len(images))
		picture, err := DecodeActorSprite(level, address-levelBase, name, palette)
		if err != nil {
			return "", err
		}
		images = append(images, picture)
		names[address] = name
		return name, nil
	}
	for _, d := range descriptors {
		kind := FixedSpriteKind{Kind: d.kind, Score: d.score, VariantSelection: d.selection}
		if d.health != 0 {
			if d.health+2 > len(level) {
				return nil, fmt.Errorf("fixed sprite health data is truncated")
			}
			kind.Health = int(binary.BigEndian.Uint16(level[d.health:]))
		}
		for variant, root := range []int{d.root0, d.root1} {
			animation, err := decodeActorAnimation(level, root-levelBase, add)
			if err != nil {
				return nil, err
			}
			kind.Variants = append(kind.Variants, FixedSpriteVariant{ID: variant, Animation: animation})
		}
		result.Kinds = append(result.Kinds, kind)
	}
	result.Atlas = packSprites(images)
	return result, nil
}
