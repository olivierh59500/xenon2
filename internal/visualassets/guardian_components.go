package visualassets

import (
	"encoding/binary"
	"fmt"
)

type GuardianComponent struct {
	Index                int               `json:"index"`
	Behavior             string            `json:"behavior"`
	DamageBehavior       string            `json:"damage_behavior"`
	ParentIndex          int               `json:"parent_index"`
	PathBudget           int               `json:"path_budget,omitempty"`
	ResourceTag          int               `json:"resource_tag"`
	InitialX             int               `json:"initial_x"`
	InitialWorldY        int               `json:"initial_world_y"`
	InitialScreenY       int               `json:"initial_screen_y,omitempty"`
	RenderMode           string            `json:"render_mode,omitempty"`
	OffsetX              int               `json:"offset_x"`
	OffsetY              int               `json:"offset_y"`
	Health               int               `json:"health"`
	Score                int               `json:"score,omitempty"`
	InitialDelay         int               `json:"initial_delay,omitempty"`
	StrongHealth         bool              `json:"strong_health"`
	InitialHeading       int               `json:"initial_heading,omitempty"`
	AngularVelocityFixed int32             `json:"angular_velocity_fixed,omitempty"`
	Sprite               string            `json:"sprite,omitempty"`
	Animation            ActorAnimation    `json:"animation"`
	HeadingFrames        []string          `json:"heading_frames,omitempty"`
	HeadingAnimations    []ActorAnimation  `json:"heading_animations,omitempty"`
	DeathAnimation       ActorAnimation    `json:"death_animation"`
	ActiveAnimation      ActorAnimation    `json:"active_animation"`
	DestroyedSprite      string            `json:"destroyed_sprite,omitempty"`
	TileFrames           []TilePatch       `json:"tile_frames,omitempty"`
	DamageFlash          *TilePatch        `json:"damage_flash,omitempty"`
	FlashOffsetX         int               `json:"flash_offset_x,omitempty"`
	FlashOffsetY         int               `json:"flash_offset_y,omitempty"`
	Overlays             []GuardianOverlay `json:"overlays,omitempty"`
}

type GuardianOverlay struct {
	ID              string   `json:"id"`
	OffsetX         int      `json:"offset_x"`
	PositiveOffsetY int      `json:"positive_offset_y"`
	NegativeOffsetY int      `json:"negative_offset_y"`
	CounterStride   int      `json:"counter_stride"`
	CounterMinimum  int      `json:"counter_minimum,omitempty"`
	CounterMaximum  int      `json:"counter_maximum,omitempty"`
	Frames          []string `json:"frames"`
}

type GuardianGroup struct {
	ID                string                `json:"id"`
	TriggerFixedKind  int                   `json:"trigger_fixed_kind"`
	Components        []GuardianComponent   `json:"components"`
	Path              *Path                 `json:"path,omitempty"`
	PathOffsetX       int                   `json:"path_offset_x,omitempty"`
	PathOffsetY       int                   `json:"path_offset_y,omitempty"`
	Launches          []GuardianLaunch      `json:"launches,omitempty"`
	DestructibleCells []GuardianTerrainCell `json:"destructible_cells,omitempty"`
	Gates             []GuardianGate        `json:"gates,omitempty"`
	MotionParameters  map[string]int        `json:"motion_parameters,omitempty"`
	Animations        []NamedActorAnimation `json:"animations,omitempty"`
}

type GuardianGate struct {
	ID     int         `json:"id"`
	Column int         `json:"column"`
	Row    int         `json:"row"`
	Frames []TilePatch `json:"frames"`
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
	if number == 2 {
		group, err := decodeSecondLevelDefense(level, add)
		if err != nil {
			return nil, SpriteAtlas{}, err
		}
		groups = append(groups, group)
		nodes, err := decodeSecondGuardianDefenseNodes(level)
		if err != nil {
			return nil, SpriteAtlas{}, err
		}
		groups = append(groups, nodes)
		cells, err := decodeSecondGuardianCells(level)
		if err != nil {
			return nil, SpriteAtlas{}, err
		}
		groups = append(groups, GuardianGroup{ID: "final-terrain-guardian", DestructibleCells: cells})
	}
	if number == 4 {
		const table, count = 0x5613a - levelBase, 20
		if len(level) < table+count*20 {
			return nil, SpriteAtlas{}, fmt.Errorf("middle guardian component table is truncated")
		}
		group := GuardianGroup{ID: "middle-guardian", TriggerFixedKind: 2}
		middleBehaviors := map[uint32]string{0x56410: "middle-head", 0x56530: "middle-follow-head", 0x56586: "middle-core", 0x565fa: "middle-core-companion", 0x567e0: "middle-tail-base", 0x56854: "middle-tail-link", 0x56864: "middle-tail-tip", 0x568f2: "middle-satellite"}
		middleDamages := map[uint32]string{0xe48: "none", 0x56354: "middle-core-damage", 0x56350: "middle-companion-damage", 0x562ca: "middle-tail-damage", 0x562fe: "middle-satellite-damage"}
		for i := range count {
			entry := level[table+i*20 : table+(i+1)*20]
			root := int(binary.BigEndian.Uint32(entry[8:])) - levelBase
			animation, err := decodeActorAnimation(level, root, add)
			if err != nil {
				return nil, SpriteAtlas{}, err
			}
			behavior, ok := middleBehaviors[binary.BigEndian.Uint32(entry)]
			if !ok {
				return nil, SpriteAtlas{}, fmt.Errorf("unsupported middle guardian updater")
			}
			damage, ok := middleDamages[binary.BigEndian.Uint32(entry[4:])]
			if !ok {
				return nil, SpriteAtlas{}, fmt.Errorf("unsupported middle guardian damage")
			}
			parent := 0
			if i == 0 {
				parent = -1
			}
			if behavior == "middle-tail-link" || behavior == "middle-tail-tip" {
				parent = i - 1
			}
			group.Components = append(group.Components, GuardianComponent{Index: i, Behavior: behavior, DamageBehavior: damage, ParentIndex: parent, PathBudget: 3, RenderMode: "sprite", ResourceTag: 84, InitialX: int(int16(binary.BigEndian.Uint16(entry[12:]))), InitialWorldY: int(int16(binary.BigEndian.Uint16(entry[14:]))), InitialHeading: int(int16(binary.BigEndian.Uint16(entry[16:]))), Health: int(entry[18]), StrongHealth: entry[19] != 0, Animation: animation})
		}
		groups = append(groups, group)
		const finalTable, finalCount = 0x55880 - levelBase, 19
		if len(level) < finalTable+finalCount*18 {
			return nil, SpriteAtlas{}, fmt.Errorf("fourth final guardian table is truncated")
		}
		final := GuardianGroup{ID: "final-guardian", TriggerFixedKind: 4}
		behaviorNames := map[uint32]string{0x55c74: "final-body-controller", 0x55d3c: "eye-turret", 0x55ed8: "arm-base", 0x55f26: "arm-link", 0x55f52: "arm-tip"}
		damageNames := map[uint32]string{0x559d6: "final-core-damage", 0x55a1c: "eye-health-counter", 0xf2c: "arm-contact"}
		for index := range finalCount {
			entry := level[finalTable+index*18 : finalTable+(index+1)*18]
			behavior, ok := behaviorNames[binary.BigEndian.Uint32(entry)]
			if !ok {
				return nil, SpriteAtlas{}, fmt.Errorf("unsupported fourth guardian updater")
			}
			damage, ok := damageNames[binary.BigEndian.Uint32(entry[4:])]
			if !ok {
				return nil, SpriteAtlas{}, fmt.Errorf("unsupported fourth guardian damage")
			}
			part := GuardianComponent{Index: index, ResourceTag: 80, Behavior: behavior, DamageBehavior: damage, ParentIndex: 0, InitialX: int(int16(binary.BigEndian.Uint16(entry[12:]))), InitialWorldY: int(int16(binary.BigEndian.Uint16(entry[14:]))), Health: int(binary.BigEndian.Uint16(entry[16:])), StrongHealth: true, PathBudget: 3}
			if index == 0 {
				part.ParentIndex = -1
				part.RenderMode = "tiled-body"
				part.TileFrames = []TilePatch{readTilePatch(level, 0x55aac-levelBase, 6, 5), readTilePatch(level, 0x55ae8-levelBase, 6, 6)}
			} else if index == 1 || index == 2 {
				part.RenderMode = "tiled-eye"
				table := 0x55ba0 - levelBase
				if index == 2 {
					table = 0x55bc4 - levelBase
				}
				for frame := range 8 {
					start, err := offset(level, table+frame*4, 8)
					if err != nil {
						return nil, SpriteAtlas{}, err
					}
					part.TileFrames = append(part.TileFrames, readTilePatch(level, start, 2, 2))
				}
			} else {
				part.RenderMode = "sprite"
				address := 0x5f8ca
				if index == 18 {
					address = 0x5f81a
				}
				name, err := add(address)
				if err != nil {
					return nil, SpriteAtlas{}, err
				}
				part.Sprite = name
				part.Animation = ActorAnimation{Frames: []AnimationFrame{{Sprite: name}}, Static: true}
				part.ParentIndex = index - 1
			}
			final.Components = append(final.Components, part)
		}
		groups = append(groups, final)
	}
	if number == 3 {
		const table = 0x558f0 - levelBase
		group := GuardianGroup{ID: "middle-guardian", TriggerFixedKind: 3, MotionParameters: map[string]int{"head_fire_rate": int(level[0x75]), "head_shot_speed": int(binary.BigEndian.Uint16(level[0x76:])), "eye_fire_rate": int(level[0x79])}}
		shot, err := decodeActorAnimation(level, 0x55b14-levelBase, add)
		if err != nil {
			return nil, SpriteAtlas{}, err
		}
		group.Animations = append(group.Animations, NamedActorAnimation{ID: "middle-head-shot", Ending: shot.Ending, Animation: shot})
		const pathStart = 0x570c2 - levelBase
		if len(level) < pathStart+90 {
			return nil, SpriteAtlas{}, fmt.Errorf("third guardian flight path is truncated")
		}
		commands, err := decodePath(level[pathStart : pathStart+90])
		if err != nil {
			return nil, SpriteAtlas{}, err
		}
		group.Path = &Path{ID: 0, Commands: commands}
		group.PathOffsetX = -32
		group.PathOffsetY = -40
		for index := 0; index < 64; index++ {
			start := table + index*16
			if start+16 > len(level) {
				return nil, SpriteAtlas{}, fmt.Errorf("third guardian component table is truncated")
			}
			if binary.BigEndian.Uint32(level[start:]) == 0 {
				break
			}
			root := int(binary.BigEndian.Uint32(level[start+8:])) - levelBase
			animation, err := decodeActorAnimation(level, root, add)
			if err != nil {
				return nil, SpriteAtlas{}, err
			}
			behaviorNames := map[uint32]string{0x55a88: "head-flight-fire", 0x55af0: "follow-head", 0x55b38: "eye-turret", 0x55c52: "arm-base", 0x55d12: "arm-link", 0x55d62: "arm-tip"}
			behavior, ok := behaviorNames[binary.BigEndian.Uint32(level[start:])]
			if !ok {
				return nil, SpriteAtlas{}, fmt.Errorf("unsupported third guardian behavior")
			}
			part := GuardianComponent{Index: index, Behavior: behavior, DamageBehavior: "none", ParentIndex: 0, PathBudget: int(binary.BigEndian.Uint16(level[0x46:])), ResourceTag: 84, InitialX: 149, InitialWorldY: -128, StrongHealth: true, AngularVelocityFixed: int32(binary.BigEndian.Uint32(level[start+12:])), Animation: animation}
			part.OffsetX = int(int16(binary.BigEndian.Uint16(level[start+12:])))
			part.OffsetY = int(int16(binary.BigEndian.Uint16(level[start+14:])))
			if behavior == "head-flight-fire" {
				part.ParentIndex = -1
			}
			if behavior == "arm-link" || behavior == "arm-tip" {
				part.ParentIndex = index - 1
			}
			if behavior == "eye-turret" {
				part.DamageBehavior = "eye-health-counter"
				part.Health = int(binary.BigEndian.Uint16(level[0x6c:]))
				part.ActiveAnimation, err = decodeActorAnimation(level, 0x55a40-levelBase, add)
				if err != nil {
					return nil, SpriteAtlas{}, err
				}
				closed := 0x5f05e
				if index == 4 {
					closed = 0x5f0f0
				}
				part.DestroyedSprite, err = add(closed)
				if err != nil {
					return nil, SpriteAtlas{}, err
				}
			}
			if behavior == "arm-tip" {
				for frame := range 16 {
					name, err := add(int(binary.BigEndian.Uint32(level[0x55dac-levelBase+frame*4:])))
					if err != nil {
						return nil, SpriteAtlas{}, err
					}
					part.HeadingFrames = append(part.HeadingFrames, name)
				}
			}
			group.Components = append(group.Components, part)
			if index == 63 {
				return nil, SpriteAtlas{}, fmt.Errorf("third guardian table lacks a terminator")
			}
		}
		if len(group.Components) != 17 {
			return nil, SpriteAtlas{}, fmt.Errorf("unsupported third guardian component count")
		}
		groups = append(groups, group)
		final, err := decodeThirdFinalGuardian(level, add)
		if err != nil {
			return nil, SpriteAtlas{}, err
		}
		groups = append(groups, final)
	}
	if number == 5 {
		const table, count = 0x5633a - levelBase, 10
		if len(level) < table+count*24 {
			return nil, SpriteAtlas{}, fmt.Errorf("middle guardian sprite table is truncated")
		}
		group := GuardianGroup{ID: "middle-guardian", TriggerFixedKind: 5}
		middleBehaviors := map[uint32]string{0x566e8: "middle-body-controller", 0x565e2: "follow-middle-body", 0x564e2: "middle-central-cannon", 0x56594: "middle-corner"}
		middleDamages := map[uint32]string{0xe48: "none", 0x56678: "middle-mount-damage", 0x56526: "middle-cannon-damage"}
		for i := range count {
			entry := level[table+i*24 : table+(i+1)*24]
			part := GuardianComponent{Index: i, ResourceTag: int(binary.BigEndian.Uint16(entry)), InitialX: 112, OffsetX: int(int16(binary.BigEndian.Uint16(entry[18:]))), OffsetY: int(int16(binary.BigEndian.Uint16(entry[20:]))), Health: int(binary.BigEndian.Uint16(entry[22:]))}
			behavior, ok := middleBehaviors[binary.BigEndian.Uint32(entry[2:])]
			if !ok {
				return nil, SpriteAtlas{}, fmt.Errorf("unsupported fifth middle updater")
			}
			part.Behavior = behavior
			damage, ok := middleDamages[binary.BigEndian.Uint32(entry[10:])]
			if !ok {
				return nil, SpriteAtlas{}, fmt.Errorf("unsupported fifth middle damage")
			}
			part.DamageBehavior = damage
			if i == 0 {
				part.ParentIndex = -1
				part.RenderMode = "tiled-middle-body"
				base := readTilePatch(level, 0x568f8-levelBase, 6, 6)
				for state := range 9 {
					patch := TilePatch{Columns: 6, Rows: 6, Tiles: append([]uint16(nil), base.Tiles...)}
					for row := range 3 {
						for column := range 2 {
							patch.Tiles[(row+1)*6+column+2] = binary.BigEndian.Uint16(level[0x5688c-levelBase+state*12+row*4+column*2:])
						}
					}
					part.TileFrames = append(part.TileFrames, patch)
				}
			} else {
				part.RenderMode = "sprite"
			}
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
		const finalTable, finalCount = 0x5594e - levelBase, 22
		if len(level) < finalTable+finalCount*24 {
			return nil, SpriteAtlas{}, fmt.Errorf("final guardian table is truncated")
		}
		final := GuardianGroup{ID: "final-guardian", TriggerFixedKind: 6}
		behaviors := map[uint32]string{0x55d7e: "final-body-controller", 0x55d3c: "barrier-band", 0x565e2: "final-mount", 0x5612a: "final-side-turret", 0x55d62: "follow-final-body", 0x55d6e: "final-weak-point"}
		damages := map[uint32]string{0xe48: "none", 0xf2c: "barrier-contact", 0x561d2: "final-part-damage", 0x562a6: "final-core-damage"}
		for index := range finalCount {
			entry := level[finalTable+index*24 : finalTable+(index+1)*24]
			behavior, ok := behaviors[binary.BigEndian.Uint32(entry[2:])]
			if !ok {
				return nil, SpriteAtlas{}, fmt.Errorf("unsupported final guardian updater")
			}
			damage, ok := damages[binary.BigEndian.Uint32(entry[10:])]
			if !ok {
				return nil, SpriteAtlas{}, fmt.Errorf("unsupported final guardian damage")
			}
			part := GuardianComponent{Index: index, ResourceTag: int(binary.BigEndian.Uint16(entry)), Behavior: behavior, DamageBehavior: damage, ParentIndex: 0, InitialX: 48, InitialScreenY: -320, OffsetX: int(int16(binary.BigEndian.Uint16(entry[18:]))), OffsetY: int(int16(binary.BigEndian.Uint16(entry[20:]))), Health: int(entry[22]), StrongHealth: entry[23] != 0, RenderMode: "sprite"}
			if index == 0 {
				part.ParentIndex = -1
			}
			renderer := binary.BigEndian.Uint32(entry[6:])
			if renderer == 0xe48 {
				part.RenderMode = "none"
			} else if renderer == 0x55eda {
				part.RenderMode = "terrain-body-with-mouth"
				mouth := GuardianOverlay{ID: "mouth", OffsetX: 105, PositiveOffsetY: 85, NegativeOffsetY: 229, CounterStride: 4}
				for frame := range 15 {
					address := int(binary.BigEndian.Uint32(level[0x55f42-levelBase+frame*4:]))
					name, err := add(address)
					if err != nil {
						return nil, SpriteAtlas{}, err
					}
					mouth.Frames = append(mouth.Frames, name)
				}
				inner := GuardianOverlay{ID: "mouth-inner", OffsetX: 119, PositiveOffsetY: 101, NegativeOffsetY: 245, CounterStride: 4, CounterMinimum: 20, CounterMaximum: 40}
				for frame := range 5 {
					address := int(binary.BigEndian.Uint32(level[0x55f2e-levelBase+frame*4:]))
					name, err := add(address)
					if err != nil {
						return nil, SpriteAtlas{}, err
					}
					inner.Frames = append(inner.Frames, name)
				}
				part.Overlays = []GuardianOverlay{mouth, inner}
			} else if renderer != 0xe12 {
				return nil, SpriteAtlas{}, fmt.Errorf("unsupported final guardian renderer")
			}
			if part.RenderMode == "sprite" {
				root := int(binary.BigEndian.Uint32(entry[14:])) - levelBase
				animation, err := decodeActorAnimation(level, root, add)
				if err != nil {
					return nil, SpriteAtlas{}, err
				}
				part.Animation = animation
			}
			final.Components = append(final.Components, part)
		}
		groups = append(groups, final)
	}
	atlas := packSprites(images)
	atlas.SourceSpriteNames = names
	return groups, atlas, nil
}

func GuardianGroupTileCodes(groups []GuardianGroup) []uint16 {
	var codes []uint16
	for _, group := range groups {
		for _, gate := range group.Gates {
			for _, patch := range gate.Frames {
				codes = append(codes, patch.Tiles...)
			}
		}
		for _, cell := range group.DestructibleCells {
			codes = append(codes, cell.RestoredTile)
		}
		for _, part := range group.Components {
			if part.DamageFlash != nil {
				codes = append(codes, part.DamageFlash.Tiles...)
			}
			for _, patch := range part.TileFrames {
				codes = append(codes, patch.Tiles...)
			}
		}
	}
	return codes
}

func RemapGuardianGroupTiles(groups []GuardianGroup, ids map[uint16]uint16) error {
	for i := range groups {
		for j := range groups[i].Gates {
			for k := range groups[i].Gates[j].Frames {
				for n, code := range groups[i].Gates[j].Frames[k].Tiles {
					id, ok := ids[code]
					if !ok && code != 0 {
						return fmt.Errorf("guardian gate tile is missing")
					}
					groups[i].Gates[j].Frames[k].Tiles[n] = id
				}
			}
		}
		for j, cell := range groups[i].DestructibleCells {
			id, ok := ids[cell.RestoredTile]
			if !ok && cell.RestoredTile != 0 {
				return fmt.Errorf("guardian destroyed tile is missing")
			}
			groups[i].DestructibleCells[j].RestoredTile = id
		}
		for j := range groups[i].Components {
			if patch := groups[i].Components[j].DamageFlash; patch != nil {
				for n, code := range patch.Tiles {
					id, ok := ids[code]
					if !ok && code != 0 {
						return fmt.Errorf("guardian damage tile is missing")
					}
					patch.Tiles[n] = id
				}
			}
			for k := range groups[i].Components[j].TileFrames {
				patch := &groups[i].Components[j].TileFrames[k]
				for n, code := range patch.Tiles {
					id, ok := ids[code]
					if !ok && code != 0 {
						return fmt.Errorf("compound guardian tile is missing from the atlas")
					}
					patch.Tiles[n] = id
				}
			}
		}
	}
	return nil
}
