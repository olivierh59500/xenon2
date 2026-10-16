package visualassets

import (
	"encoding/binary"
	"fmt"
)

// DecodePlayerShotArt exports all four ordinary firing directions and three
// tiers. These projectiles use native point collision rather than actor boxes.
func DecodePlayerShotArt(common []byte, palette [16][4]uint8) (SpriteAtlas, error) {
	sprites, err := decodePlayerShotSprites(common, palette)
	if err != nil {
		return SpriteAtlas{}, err
	}
	return packSprites(sprites), nil
}

func decodePlayerShotSprites(common []byte, palette [16][4]uint8) ([]*Sprite, error) {
	sprites := make([]*Sprite, 0, 12)
	for _, variant := range []struct {
		name  string
		table int
	}{{"left-shot", 0x27c2}, {"forward-shot", 0x27ce}, {"right-shot", 0x27da}, {"rear-shot", 0x27e6}} {
		if variant.table+12 > len(common) {
			return nil, fmt.Errorf("player shot table is truncated")
		}
		for tier := range 3 {
			address := int(binary.BigEndian.Uint32(common[variant.table+tier*4:]))
			name := fmt.Sprintf("%s-%d", variant.name, tier)
			picture, err := DecodeSprite(common, address, name, palette)
			if err != nil {
				return nil, err
			}
			sprites = append(sprites, picture)
		}
	}
	return sprites, nil
}
