package visualassets

import (
	"encoding/binary"
	"fmt"
)

type ThirdCompoundCannonArt struct {
	Health                                                                                          [2]int `json:"health"`
	Base, SecondBase, Destroyed                                                                     TilePatch
	FirstFrames                                                                                     []TilePatch `json:"first_frames"`
	SecondFrames                                                                                    []TilePatch `json:"second_frames"`
	FlashFirst, FlashSecond                                                                         TilePatch
	FirstThreshold, RepeatThreshold, FirstFireRate, FirstShotSpeed, SecondFireRate, SecondShotSpeed int
	FirstShotSprite                                                                                 string            `json:"first_shot_sprite"`
	RadialAnimations                                                                                [8]ActorAnimation `json:"radial_animations"`
}

func decodeThirdCompoundCannon(level []byte, add func(int) (string, error)) (ThirdCompoundCannonArt, error) {
	var art ThirdCompoundCannonArt
	if len(level) < 0x56df6-levelBase {
		return art, fmt.Errorf("third compound cannon data is truncated")
	}
	art.Health = [2]int{int(binary.BigEndian.Uint16(level[0x4c:])), int(binary.BigEndian.Uint16(level[0x60:]))}
	art.Base = readTilePatch(level, 0x561f0-levelBase, 4, 6)
	art.SecondBase = TilePatch{Columns: art.Base.Columns, Rows: art.Base.Rows, Tiles: append([]uint16(nil), art.Base.Tiles...)}
	for _, index := range []int{8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 21, 22} {
		art.SecondBase.Tiles[index] = 0
	}
	art.Destroyed = TilePatch{Columns: 4, Rows: 6, Tiles: make([]uint16, 24)}
	art.FirstThreshold, art.RepeatThreshold = int(level[0x57]), int(level[0x59])
	art.FirstFireRate, art.FirstShotSpeed = int(level[0x5b]), int(binary.BigEndian.Uint16(level[0x5e:]))
	art.SecondFireRate, art.SecondShotSpeed = int(level[0x63]), int(binary.BigEndian.Uint16(level[0x64:]))
	art.FlashFirst, art.FlashSecond = readTilePatch(level, 0x56a74-levelBase, 4, 3), readTilePatch(level, 0x56d2c-levelBase, 2, 2)
	for index := range 9 {
		art.FirstFrames = append(art.FirstFrames, readTilePatch(level, 0x569a2-levelBase+index*8, 2, 2))
	}
	for index := range 5 {
		art.SecondFrames = append(art.SecondFrames, readTilePatch(level, 0x56d2c-levelBase+index*8, 2, 2))
	}
	choice := int(binary.BigEndian.Uint16(level[0x5c:]))
	table, err := offset(level, 0x6ee26-levelBase+choice*4, 4)
	if err != nil {
		return art, err
	}
	art.FirstShotSprite, err = add(int(binary.BigEndian.Uint32(level[table:])))
	if err != nil {
		return art, err
	}
	for direction := range 8 {
		root := int(binary.BigEndian.Uint32(level[0x56bec-levelBase+direction*4:]))
		animation, err := decodeActorAnimation(level, root-levelBase, add)
		if err != nil {
			return art, err
		}
		art.RadialAnimations[direction] = animation
	}
	return art, nil
}
func ThirdFixedTileCodes(art *ThirdFixedArt) []uint16 {
	if art == nil {
		return nil
	}
	result := append([]uint16(nil), art.Scenery.Tiles...)
	for _, patch := range []TilePatch{art.Cannon.Base, art.Cannon.SecondBase, art.Cannon.Destroyed, art.Cannon.FlashFirst, art.Cannon.FlashSecond} {
		result = append(result, patch.Tiles...)
	}
	for _, patch := range art.Cannon.FirstFrames {
		result = append(result, patch.Tiles...)
	}
	for _, patch := range art.Cannon.SecondFrames {
		result = append(result, patch.Tiles...)
	}
	return result
}
func RemapThirdFixedTiles(art *ThirdFixedArt, ids map[uint16]uint16) error {
	if art == nil {
		return nil
	}
	patches := []*TilePatch{&art.Scenery, &art.Cannon.Base, &art.Cannon.SecondBase, &art.Cannon.Destroyed, &art.Cannon.FlashFirst, &art.Cannon.FlashSecond}
	for i := range art.Cannon.FirstFrames {
		patches = append(patches, &art.Cannon.FirstFrames[i])
	}
	for i := range art.Cannon.SecondFrames {
		patches = append(patches, &art.Cannon.SecondFrames[i])
	}
	for _, patch := range patches {
		for i, code := range patch.Tiles {
			id, ok := ids[code]
			if !ok && code != 0 {
				return fmt.Errorf("third fixed terrain tile is missing")
			}
			patch.Tiles[i] = id
		}
	}
	return nil
}
