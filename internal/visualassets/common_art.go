package visualassets

import (
	"encoding/binary"
	"fmt"
)

// DecodeCommonActorArt exports the checked contiguous common image bank. It
// contains the ship, small weapons, pickups and effects; every image retains
// its own anchor and collision prefix. The trailing non-image data is omitted.
func DecodeCommonActorArt(common []byte, palette [16][4]uint8) (SpriteAtlas, error) {
	return DecodeCommonActorArtWithEquipment(common, nil, nil, palette)
}

// DecodeCommonActorArtWithEquipment adds exact equipment-name aliases and any
// additional common images reached through the shop's checked frame lists.
func DecodeCommonActorArtWithEquipment(common, shop []byte, catalogue *ShopCatalogue, palette [16][4]uint8) (SpriteAtlas, error) {
	const first, last, count = 0xe78a, 0x154be, 136
	if len(common) < last {
		return SpriteAtlas{}, fmt.Errorf("common image bank is truncated")
	}
	images := make([]*Sprite, 0, count)
	names := map[int]string{}
	cursor := first
	for i := range count {
		name := fmt.Sprintf("common-actor-%03d", i)
		sprite, err := DecodeActorSprite(common, cursor, name, palette)
		if err != nil {
			return SpriteAtlas{}, err
		}
		images = append(images, sprite)
		names[cursor] = name
		cursor += 8 + sprite.Image.Bounds().Dx()/8*5*sprite.Height + 8
	}
	if cursor != last {
		return SpriteAtlas{}, fmt.Errorf("unsupported common image bank boundaries")
	}
	equipment := make([]ItemAnimation, 0)
	if catalogue != nil {
		for itemIndex, item := range catalogue.Items {
			start, err := offset(shop, 0x5581c-levelBase+itemIndex*4, 4)
			if err != nil {
				return SpriteAtlas{}, err
			}
			animation := ItemAnimation{ID: item.ID}
			offsets := map[int]int{}
			cursor = start
			for len(animation.Frames) < 256 {
				if cursor+4 > len(shop) {
					return SpriteAtlas{}, fmt.Errorf("equipment frame list is truncated")
				}
				address := int(binary.BigEndian.Uint32(shop[cursor:]))
				offsets[cursor] = len(animation.Frames)
				cursor += 4
				if address == 0 {
					if cursor+4 > len(shop) {
						return SpriteAtlas{}, fmt.Errorf("equipment loop is truncated")
					}
					target := int(binary.BigEndian.Uint32(shop[cursor:])) - levelBase
					index, ok := offsets[target]
					if !ok {
						return SpriteAtlas{}, fmt.Errorf("equipment loop leaves its frame list")
					}
					animation.LoopFrom = index
					break
				}
				name, ok := names[address]
				if !ok {
					name = fmt.Sprintf("common-actor-%03d", len(images))
					var picture *Sprite
					var err error
					if address >= levelBase {
						picture, err = DecodeSprite(shop, address-levelBase, name, palette)
					} else {
						picture, err = DecodeActorSprite(common, address, name, palette)
					}
					if err != nil {
						return SpriteAtlas{}, err
					}
					images = append(images, picture)
					names[address] = name
				}
				animation.Frames = append(animation.Frames, name)
			}
			if len(animation.Frames) == 256 {
				return SpriteAtlas{}, fmt.Errorf("equipment frame list is too long")
			}
			equipment = append(equipment, animation)
		}
	}
	atlas := packSprites(images)
	atlas.Equipment = equipment
	return atlas, nil
}
