package visualassets

import (
	"encoding/binary"
	"fmt"
)

type ThirdCrawlerArt struct {
	Health            int               `json:"health"`
	InitialDX         [8]int            `json:"initial_dx"`
	InitialDY         [8]int            `json:"initial_dy"`
	StepDX            [8]int            `json:"step_dx"`
	StepDY            [8]int            `json:"step_dy"`
	ForwardTileOffset [8]int            `json:"forward_tile_offset"`
	TurnDX            [8]int            `json:"turn_dx"`
	TurnDY            [8]int            `json:"turn_dy"`
	TurnHeading       [8]int            `json:"turn_heading"`
	Animations        [8]ActorAnimation `json:"animations"`
	FireRate          int               `json:"fire_rate"`
	ShotSpeed         int               `json:"shot_speed"`
	ShotSprite        string            `json:"shot_sprite"`
}
type ThirdChainArt struct {
	Health int               `json:"health"`
	Bodies [2]string         `json:"bodies"`
	Tails  [2]ActorAnimation `json:"tails"`
}
type ThirdFixedArt struct {
	Scenery TilePatch              `json:"scenery"`
	Cannon  ThirdCompoundCannonArt `json:"cannon"`
	Chain   ThirdChainArt          `json:"chain"`
	Crawler ThirdCrawlerArt        `json:"crawler"`
}

func decodeThirdFixedArt(level []byte, add func(int) (string, error)) (*ThirdFixedArt, error) {
	if len(level) < 0x56490-levelBase {
		return nil, fmt.Errorf("third fixed sprite data is truncated")
	}
	result := &ThirdFixedArt{}
	result.Chain.Health = int(binary.BigEndian.Uint16(level[0x48:]))
	for i, address := range []int{0x5b280, 0x5b1d0} {
		name, err := add(address)
		if err != nil {
			return nil, err
		}
		result.Chain.Bodies[i] = name
	}
	for i, root := range []int{0x560b4, 0x560de} {
		at := root - levelBase
		if binary.BigEndian.Uint32(level[at+30:]) != 0 || binary.BigEndian.Uint32(level[at+34:]) != 0xe2a || int(binary.BigEndian.Uint32(level[at+38:])) != root {
			return nil, fmt.Errorf("controlled chain tail format differs")
		}
		clip := ActorAnimation{Ending: "loop"}
		for frame := range 5 {
			name, err := add(int(binary.BigEndian.Uint32(level[at+frame*6:])))
			if err != nil {
				return nil, err
			}
			clip.Frames = append(clip.Frames, AnimationFrame{Sprite: name, Duration: int(binary.BigEndian.Uint16(level[at+frame*6+4:]))})
		}
		result.Chain.Tails[i] = clip
	}
	art := &result.Crawler
	art.Health = int(binary.BigEndian.Uint16(level[0x44:]))
	art.FireRate = int(level[0x4f])
	art.ShotSpeed = int(binary.BigEndian.Uint16(level[0x52:]))
	for _, definition := range []struct {
		at     int
		values *[8]int
	}{{0x55790, &art.InitialDX}, {0x557a0, &art.InitialDY}, {0x56290, &art.StepDX}, {0x562a0, &art.StepDY}, {0x56450, &art.ForwardTileOffset}, {0x56460, &art.TurnDX}, {0x56470, &art.TurnDY}, {0x56480, &art.TurnHeading}} {
		for i := range 8 {
			definition.values[i] = int(int16(binary.BigEndian.Uint16(level[definition.at-levelBase+i*2:])))
		}
	}
	for direction := range 8 {
		root := int(binary.BigEndian.Uint32(level[0x557b0-levelBase+direction*4:]))
		clip, err := decodeActorAnimation(level, root-levelBase, add)
		if err != nil {
			return nil, err
		}
		art.Animations[direction] = clip
	}
	shotChoice := int(binary.BigEndian.Uint16(level[0x50:]))
	shotTable, err := offset(level, 0x6ee26-levelBase+shotChoice*4, 4)
	if err != nil {
		return nil, err
	}
	art.ShotSprite, err = add(int(binary.BigEndian.Uint32(level[shotTable:])))
	if err != nil {
		return nil, err
	}
	result.Cannon, err = decodeThirdCompoundCannon(level, add)
	if err != nil {
		return nil, err
	}
	result.Scenery = TilePatch{Columns: 20, Rows: 20, Tiles: make([]uint16, 400)}
	for cursor := 0x56f9a - levelBase; binary.BigEndian.Uint32(level[cursor:]) != 0; cursor += 12 {
		at := int(binary.BigEndian.Uint32(level[cursor:])) - levelBase
		cols := int(binary.BigEndian.Uint16(level[cursor+4:])) + 1
		rows := int(binary.BigEndian.Uint16(level[cursor+6:])) + 1
		y := int(binary.BigEndian.Uint16(level[cursor+8:])) / 16
		x := int(binary.BigEndian.Uint16(level[cursor+10:])) / 8
		if x+cols > 20 || y+rows > 20 {
			return nil, fmt.Errorf("third final scenery leaves its patch")
		}
		patch := readTilePatch(level, at, cols, rows)
		for row := range rows {
			copy(result.Scenery.Tiles[(y+row)*20+x:][:cols], patch.Tiles[row*cols:][:cols])
		}
	}
	return result, nil
}
