package visualassets

import (
	"encoding/binary"
	"fmt"
)

type GuardianVisual struct {
	ID                string                  `json:"id"`
	InitialHealth     int                     `json:"initial_health"`
	InitialWorldY     int                     `json:"initial_world_y"`
	BodyX             int                     `json:"body_x"`
	Body              TilePatch               `json:"body"`
	EyeX              int                     `json:"eye_x"`
	EyeOffsetY        int                     `json:"eye_offset_y"`
	EyeFrames         []string                `json:"eye_frames"`
	SegmentSprite     string                  `json:"segment_sprite,omitempty"`
	SegmentCount      int                     `json:"segment_count,omitempty"`
	TailHeadingFrames []string                `json:"tail_heading_frames,omitempty"`
	FlameAnimation    ActorAnimation          `json:"flame_animation"`
	BodyAnimations    []GuardianBodyAnimation `json:"body_animations,omitempty"`
	Animations        []NamedActorAnimation   `json:"animations,omitempty"`
	MotionParameters  map[string]int          `json:"motion_parameters,omitempty"`
	TurnPoints        []GuardianTurnPoint     `json:"turn_points,omitempty"`
}

type GuardianTurnPoint struct {
	X        int    `json:"x"`
	WorldY   int    `json:"world_y"`
	Headings [3]int `json:"headings"`
}

type GuardianBodyAnimation struct {
	ID     string      `json:"id"`
	Column int         `json:"column"`
	Row    int         `json:"row"`
	Frames []TilePatch `json:"frames"`
}

type Guardians struct {
	Visuals []GuardianVisual `json:"visuals"`
	Atlas   SpriteAtlas      `json:"atlas"`
}

// DecodeGuardianArt exports verified tiled guardians independently of their
// controllers, including mutable body cells and attached image sequences.
func DecodeGuardianArt(number int, level []byte, palette [16][4]uint8) (*Guardians, []uint16, error) {
	if number == 2 {
		const bodyStart = 0x5725e - levelBase
		if len(level) < bodyStart+108 {
			return nil, nil, fmt.Errorf("second guardian body tiles are truncated")
		}
		body := readTilePatch(level, bodyStart, 6, 9)
		guardian := GuardianVisual{ID: "final-guardian", InitialHealth: int(binary.BigEndian.Uint16(level[0x5e:])), InitialWorldY: 96, BodyX: 112, Body: body, MotionParameters: map[string]int{"fire_rate": int(level[0x65]), "shot_speed": int(binary.BigEndian.Uint16(level[0x66:]))}}
		extra := append([]uint16(nil), body.Tiles...)
		for _, definition := range []struct {
			id                        string
			column, row, table, count int
		}{{"center-hatch", 2, 6, 0x56c44, 4}, {"left-thruster", 0, 7, 0x56e46, 2}, {"right-thruster", 4, 7, 0x56e4e, 2}} {
			animation := GuardianBodyAnimation{ID: definition.id, Column: definition.column, Row: definition.row}
			for frame := range definition.count {
				start, err := offset(level, definition.table-levelBase+frame*4, 8)
				if err != nil {
					return nil, nil, err
				}
				patch := readTilePatch(level, start, 2, 2)
				animation.Frames = append(animation.Frames, patch)
				extra = append(extra, patch.Tiles...)
			}
			guardian.BodyAnimations = append(guardian.BodyAnimations, animation)
		}
		images := []*Sprite{}
		names := map[int]string{}
		add := func(address int) (string, error) {
			if name, ok := names[address]; ok {
				return name, nil
			}
			name := fmt.Sprintf("guardian-projectile-%03d", len(images))
			picture, err := DecodeActorSprite(level, address-levelBase, name, palette)
			if err != nil {
				return "", err
			}
			images = append(images, picture)
			names[address] = name
			return name, nil
		}
		for _, definition := range []struct {
			id   string
			root int
		}{{"guardian-single-shot", 0x56e8e}, {"guardian-radial-shot", 0x56eac}, {"guardian-hatch-minion", 0x55116}} {
			animation, err := decodeActorAnimation(level, definition.root-levelBase, add)
			if err != nil {
				return nil, nil, err
			}
			guardian.Animations = append(guardian.Animations, NamedActorAnimation{ID: definition.id, Ending: animation.Ending, Animation: animation})
		}
		attack, err := decodeActorAnimation(level, 0x5513a-levelBase, add)
		if err != nil {
			return nil, nil, err
		}
		guardian.Animations = append(guardian.Animations, NamedActorAnimation{ID: "guardian-minion-transform", Ending: attack.Ending, Animation: attack})
		for index, frame := range attack.Frames {
			if names[0x5b5d6] == frame.Sprite {
				guardian.MotionParameters["minion_transform_frame"] = index
				break
			}
		}
		for heading := range 8 {
			root := int(binary.BigEndian.Uint32(level[0x563c0-levelBase+heading*4:]))
			animation, err := decodeActorAnimation(level, root-levelBase, add)
			if err != nil {
				return nil, nil, err
			}
			guardian.Animations = append(guardian.Animations, NamedActorAnimation{ID: fmt.Sprintf("guardian-turret-%d", heading), Ending: animation.Ending, Animation: animation})
		}
		guardian.MotionParameters["turret_health"] = int(binary.BigEndian.Uint16(level[0x56:]))
		guardian.MotionParameters["turret_fire_rate"] = int(level[0x59])
		guardian.MotionParameters["turret_shot_speed"] = int(binary.BigEndian.Uint16(level[0x5c:]))
		choice := int(binary.BigEndian.Uint16(level[0x5a:]))
		shotRoot, err := offset(level, 0x6c89c-levelBase+choice*4, 4)
		if err != nil {
			return nil, nil, err
		}
		shot, err := add(int(binary.BigEndian.Uint32(level[shotRoot:])))
		if err != nil {
			return nil, nil, err
		}
		guardian.Animations = append(guardian.Animations, NamedActorAnimation{ID: "guardian-turret-shot", Ending: "hold", Animation: ActorAnimation{Frames: []AnimationFrame{{Sprite: shot}}, Static: true}})
		if binary.BigEndian.Uint16(level[0x56524-levelBase:]) != 7 {
			return nil, nil, fmt.Errorf("guardian turn point count differs")
		}
		for i := range 8 {
			at := 0x56526 - levelBase + i*10
			guardian.TurnPoints = append(guardian.TurnPoints, GuardianTurnPoint{X: int(int16(binary.BigEndian.Uint16(level[at:]))), WorldY: int(int16(binary.BigEndian.Uint16(level[at+2:]))), Headings: [3]int{int(binary.BigEndian.Uint16(level[at+4:])), int(binary.BigEndian.Uint16(level[at+6:])), int(binary.BigEndian.Uint16(level[at+8:]))}})
		}
		atlas := packSprites(images)
		atlas.SourceSpriteNames = names
		return &Guardians{Visuals: []GuardianVisual{guardian}, Atlas: atlas}, extra, nil
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
	extra := append([]uint16(nil), body.Tiles...)
	decoration := GuardianBodyAnimation{ID: "lower-body-decoration", Column: 1, Row: 5}
	for frame := range 4 {
		start, err := offset(level, 0x565fc-levelBase+frame*4, 16)
		if err != nil {
			return nil, nil, err
		}
		patch := readTilePatch(level, start, 4, 2)
		decoration.Frames = append(decoration.Frames, patch)
		extra = append(extra, patch.Tiles...)
	}
	guardian.BodyAnimations = append(guardian.BodyAnimations, decoration)
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
	return &Guardians{Visuals: []GuardianVisual{guardian}, Atlas: packSprites(images)}, extra, nil
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
		for j := range guardians.Visuals[i].BodyAnimations {
			for k := range guardians.Visuals[i].BodyAnimations[j].Frames {
				for l, code := range guardians.Visuals[i].BodyAnimations[j].Frames[k].Tiles {
					id, ok := ids[code]
					if !ok && code != 0 {
						return fmt.Errorf("guardian animated tile is missing from the atlas")
					}
					guardians.Visuals[i].BodyAnimations[j].Frames[k].Tiles[l] = id
				}
			}
		}
	}
	return nil
}
