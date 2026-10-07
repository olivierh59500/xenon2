package visualassets

import (
	"encoding/binary"
	"fmt"
)

type GuardianComponent struct {
	Index          int            `json:"index"`
	ResourceTag    int            `json:"resource_tag"`
	InitialX       int            `json:"initial_x"`
	InitialWorldY  int            `json:"initial_world_y"`
	OffsetX        int            `json:"offset_x"`
	OffsetY        int            `json:"offset_y"`
	Health         int            `json:"health"`
	StrongHealth   bool           `json:"strong_health"`
	InitialHeading int            `json:"initial_heading,omitempty"`
	Sprite         string         `json:"sprite,omitempty"`
	Animation      ActorAnimation `json:"animation"`
}

type GuardianGroup struct {
	ID               string              `json:"id"`
	TriggerFixedKind int                 `json:"trigger_fixed_kind"`
	Components       []GuardianComponent `json:"components"`
}

// DecodeCompoundGuardianArt reads the verified component tables. Motion and
// damage callbacks remain the independent Go controller's responsibility.
func DecodeCompoundGuardianArt(number int, level []byte, palette [16][4]uint8) ([]GuardianGroup, SpriteAtlas, error) {
	groups := make([]GuardianGroup, 0)
	images := make([]*Sprite, 0)
	names := map[int]string{}
	add := func(address int) (string, error) {
		if name, ok := names[address]; ok {
			return name, nil
		}
		name := fmt.Sprintf("guardian-part-%03d", len(images))
		picture, err := DecodeActorSprite(level, address-levelBase, name, palette)
		if err != nil {
			return "", err
		}
		images = append(images, picture)
		names[address] = name
		return name, nil
	}
	if number == 4 {
		const table, count = 0x5613a - levelBase, 20
		if len(level) < table+count*20 {
			return nil, SpriteAtlas{}, fmt.Errorf("middle guardian component table is truncated")
		}
		group := GuardianGroup{ID: "middle-guardian", TriggerFixedKind: 2}
		for i := range count {
			entry := level[table+i*20 : table+(i+1)*20]
			root := int(binary.BigEndian.Uint32(entry[8:])) - levelBase
			animation, err := decodeActorAnimation(level, root, add)
			if err != nil {
				return nil, SpriteAtlas{}, err
			}
			group.Components = append(group.Components, GuardianComponent{Index: i, ResourceTag: 84, InitialX: int(int16(binary.BigEndian.Uint16(entry[12:]))), InitialWorldY: int(int16(binary.BigEndian.Uint16(entry[14:]))), InitialHeading: int(int16(binary.BigEndian.Uint16(entry[16:]))), Health: int(entry[18]), StrongHealth: entry[19] != 0, Animation: animation})
		}
		groups = append(groups, group)
	}
	if number == 5 {
		const table, count = 0x5633a - levelBase, 10
		if len(level) < table+count*24 {
			return nil, SpriteAtlas{}, fmt.Errorf("middle guardian sprite table is truncated")
		}
		group := GuardianGroup{ID: "middle-guardian", TriggerFixedKind: 5}
		for i := range count {
			entry := level[table+i*24 : table+(i+1)*24]
			part := GuardianComponent{Index: i, ResourceTag: int(binary.BigEndian.Uint16(entry)), InitialX: 112, OffsetX: int(int16(binary.BigEndian.Uint16(entry[18:]))), OffsetY: int(int16(binary.BigEndian.Uint16(entry[20:]))), Health: int(binary.BigEndian.Uint16(entry[22:]))}
			address := int(binary.BigEndian.Uint32(entry[14:]))
			if address != 0 {
				name, err := add(address)
				if err != nil {
					return nil, SpriteAtlas{}, err
				}
				part.Sprite = name
				part.Animation = ActorAnimation{Frames: []AnimationFrame{{Sprite: name}}, Static: true}
			}
			group.Components = append(group.Components, part)
		}
		groups = append(groups, group)
	}
	return groups, packSprites(images), nil
}
