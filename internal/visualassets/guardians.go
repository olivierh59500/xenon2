package visualassets

import (
	"encoding/binary"
	"fmt"
)

type GuardianVisual struct {
	ID                string         `json:"id"`
	InitialHealth     int            `json:"initial_health"`
	InitialWorldY     int            `json:"initial_world_y"`
	BodyX             int            `json:"body_x"`
	Body              TilePatch      `json:"body"`
	EyeX              int            `json:"eye_x"`
	EyeOffsetY        int            `json:"eye_offset_y"`
	EyeFrames         []string       `json:"eye_frames"`
	SegmentSprite     string         `json:"segment_sprite,omitempty"`
	SegmentCount      int            `json:"segment_count,omitempty"`
	TailHeadingFrames []string       `json:"tail_heading_frames,omitempty"`
	FlameAnimation    ActorAnimation `json:"flame_animation"`
}

type Guardians struct {
	Visuals []GuardianVisual `json:"visuals"`
	Atlas   SpriteAtlas      `json:"atlas"`
}

// DecodeGuardianArt exports verified guardian visuals independently of their
// phase controllers. The first level's tiled body and eye cycle are available;
// later guardian layouts are added only after their native formats are checked.
func DecodeGuardianArt(number int, level []byte, palette [16][4]uint8) (*Guardians, []uint16, error) {
	if number == 2 {
		const bodyStart = 0x5725e - levelBase
		if len(level) < bodyStart+108 {
			return nil, nil, fmt.Errorf("second guardian body tiles are truncated")
		}
		body := readTilePatch(level, bodyStart, 6, 9)
		guardian := GuardianVisual{ID: "final-guardian", InitialHealth: int(binary.BigEndian.Uint16(level[0x5e:])), InitialWorldY: 96, BodyX: 112, Body: body}
		return &Guardians{Visuals: []GuardianVisual{guardian}, Atlas: packSprites(nil)}, append([]uint16(nil), body.Tiles...), nil
	}
	if number == 5 {
		const bodyStart = 0x57666 - levelBase
		if len(level) < bodyStart+540 {
			return nil, nil, fmt.Errorf("final terrain guardian body is truncated")
		}
		body := readTilePatch(level, bodyStart, 15, 18)
		guardian := GuardianVisual{ID: "final-guardian", InitialHealth: 20, InitialWorldY: 96, BodyX: 48, Body: body}
		return &Guardians{Visuals: []GuardianVisual{guardian}, Atlas: packSprites(nil)}, append([]uint16(nil), body.Tiles...), nil
	}
	if number != 1 {
		return nil, nil, nil
	}
	const bodyStart, eyeTable = 0x567bc - levelBase, 0x56666 - levelBase
	if len(level) < bodyStart+84 {
		return nil, nil, fmt.Errorf("guardian body tiles are truncated")
	}
	body := readTilePatch(level, bodyStart, 6, 7)
	guardian := GuardianVisual{ID: "final-guardian", InitialHealth: int(binary.BigEndian.Uint16(level[0x58:])), InitialWorldY: 16, BodyX: 112, Body: body, EyeX: 152, EyeOffsetY: 66}
	images := make([]*Sprite, 0, 3)
	names := map[int]string{}
	for i := range 4 {
		address := int(binary.BigEndian.Uint32(level[eyeTable+i*4:]))
		name, ok := names[address]
		if !ok {
			name = fmt.Sprintf("guardian-eye-%d", len(images))
			picture, err := DecodeSprite(level, address-levelBase, name, palette)
			if err != nil {
				return nil, nil, err
			}
			images = append(images, picture)
			names[address] = name
		}
		guardian.EyeFrames = append(guardian.EyeFrames, name)
	}
	segment, err := DecodeActorSprite(level, 0x59e28-levelBase, "guardian-segment", palette)
	if err != nil {
		return nil, nil, err
	}
	images = append(images, segment)
	guardian.SegmentSprite = segment.Name
	guardian.SegmentCount = 8
	for heading := range 8 {
		address := int(binary.BigEndian.Uint32(level[0x5638a-levelBase+heading*4:]))
		name := fmt.Sprintf("guardian-tail-%d", heading)
		picture, err := DecodeActorSprite(level, address-levelBase, name, palette)
		if err != nil {
			return nil, nil, err
		}
		images = append(images, picture)
		guardian.TailHeadingFrames = append(guardian.TailHeadingFrames, name)
	}
	flame, err := decodeActorAnimation(level, 0x552b0-levelBase, func(address int) (string, error) {
		name := fmt.Sprintf("guardian-flame-%d", len(images))
		picture, err := DecodeActorSprite(level, address-levelBase, name, palette)
		if err != nil {
			return "", err
		}
		images = append(images, picture)
		return name, nil
	})
	if err != nil {
		return nil, nil, err
	}
	guardian.FlameAnimation = flame
	return &Guardians{Visuals: []GuardianVisual{guardian}, Atlas: packSprites(images)}, append([]uint16(nil), body.Tiles...), nil
}

func RemapGuardianTiles(guardians *Guardians, ids map[uint16]uint16) error {
	for i := range guardians.Visuals {
		for j, code := range guardians.Visuals[i].Body.Tiles {
			id, ok := ids[code]
			if !ok && code != 0 {
				return fmt.Errorf("guardian body tile is missing from the atlas")
			}
			guardians.Visuals[i].Body.Tiles[j] = id
		}
	}
	return nil
}
