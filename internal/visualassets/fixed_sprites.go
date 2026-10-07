package visualassets

import (
	"encoding/binary"
	"fmt"
)

type FixedSpriteVariant struct {
	ID               int            `json:"id"`
	ResourceTag      int            `json:"resource_tag"`
	OriginOffsetX    int            `json:"origin_offset_x"`
	OriginOffsetY    int            `json:"origin_offset_y"`
	InitialVelocityX int            `json:"initial_velocity_x,omitempty"`
	Animation        ActorAnimation `json:"animation"`
	AttackAnimation  ActorAnimation `json:"attack_animation"`
	ReverseAnimation ActorAnimation `json:"reverse_animation"`
	ShotMode         string         `json:"shot_mode,omitempty"`
	ShotDirections   []int          `json:"shot_directions,omitempty"`
	ShotSpeed        int            `json:"shot_speed,omitempty"`
	ShotSprite       string         `json:"shot_sprite,omitempty"`
	ShotOffsetX      int            `json:"shot_offset_x,omitempty"`
	ShotOffsetY      int            `json:"shot_offset_y,omitempty"`
	ShotMotionBudget int            `json:"shot_motion_budget,omitempty"`
	ShotAnimation    ActorAnimation `json:"shot_animation"`
	Cover            *TilePatch     `json:"cover,omitempty"`
}

type FixedSpriteKind struct {
	Kind              int                  `json:"kind"`
	Health            int                  `json:"health,omitempty"`
	Score             int                  `json:"score,omitempty"`
	VariantSelection  string               `json:"variant_selection"`
	VariantThresholdX int                  `json:"variant_threshold_x,omitempty"`
	Damageable        bool                 `json:"damageable"`
	StrongHealth      bool                 `json:"strong_health"`
	ActorList         string               `json:"actor_list"`
	CollisionMode     string               `json:"collision_mode"`
	ContactDamage     int                  `json:"contact_damage,omitempty"`
	Behavior          string               `json:"behavior"`
	MotionParameters  map[string]int       `json:"motion_parameters,omitempty"`
	MotionTables      map[string][]int     `json:"motion_tables,omitempty"`
	Variants          []FixedSpriteVariant `json:"variants"`
}

type FixedSprites struct {
	Third          *ThirdFixedArt          `json:"third,omitempty"`
	Kinds          []FixedSpriteKind       `json:"kinds"`
	Atlas          SpriteAtlas             `json:"atlas"`
	Projectile     *FixedProjectileArtwork `json:"projectile,omitempty"`
	HatchCreatures *HatchCreatureArtwork   `json:"hatch_creatures,omitempty"`
	PodCreatures   *PodCreatureArtwork     `json:"pod_creatures,omitempty"`
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
		kind.Damageable = levelNumber != 4
		kind.ActorList = "moving"
		kind.CollisionMode = "sprite-prefix"
		switch levelNumber {
		case 1, 2:
			threshold := 50
			if levelNumber == 2 {
				threshold = 30
			}
			kind.Behavior = "scroll-bounce-attack"
			kind.MotionParameters = map[string]int{"initial_vertical_velocity": -1, "amplitude_state1_scale": 16, "attack_y_minimum": -30, "attack_y_maximum": 10, "attack_random_threshold": threshold, "clip_margin": 208}
			if levelNumber == 1 {
				delete(kind.MotionParameters, "amplitude_state1_scale")
				kind.MotionParameters["amplitude_state2_scale"] = 16
			}
		case 3:
			kind.Behavior = "horizontal-sweeper"
			kind.MotionParameters = map[string]int{"horizontal_speed": 2, "left_edge": -32, "right_edge": 352, "left_reset": 0, "right_reset": 320, "left_attack_x": 136, "right_attack_x": 184, "attack_shot_tick": 11}
		case 4:
			kind.Behavior = "extending-beam"
			kind.MotionParameters = map[string]int{"maximum_phase": 6, "phase_spacing": 16, "contact_y_offset": 8, "contact_height": 17, "clip_bottom": 392, "fire_rate": int(level[0x5f])}
		case 5:
			kind.Behavior = "vertical-oscillator"
			kind.MotionParameters = map[string]int{"vertical_speed": 1, "minimum_world_y": 3632, "maximum_world_y": 3872, "clip_bottom": 400, "fire_rate": int(level[0x4b])}
		}
		if levelNumber == 4 {
			kind.ActorList = "scenery"
			kind.CollisionMode = "extending-beam-rectangle"
			kind.ContactDamage = 6
		}
		kind.StrongHealth = levelNumber == 1 || levelNumber == 3 || levelNumber == 5
		if levelNumber == 3 {
			kind.VariantThresholdX = 160
		}
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
			tags := [5][2]int{{208, 212}, {252, 256}, {208, 208}, {268, 272}, {256, 260}}
			v := FixedSpriteVariant{ID: variant, ResourceTag: tags[levelNumber-1][variant], OriginOffsetX: -8, OriginOffsetY: -8, Animation: animation}
			if (levelNumber == 1 || levelNumber == 5) && variant == 1 {
				v.OriginOffsetX = 8
			}
			if levelNumber == 3 {
				v.OriginOffsetY = 8
				v.InitialVelocityX = 2
				if variant == 1 {
					v.InitialVelocityX = -2
				}
				attackRoot, shotImage := 0x567f2, 0x5b330
				v.ShotOffsetX, v.ShotOffsetY = -5, 4
				v.ShotDirections = []int{3}
				if variant == 1 {
					attackRoot, shotImage = 0x56840, 0x5b386
					v.ShotOffsetX, v.ShotDirections = 5, []int{5}
				}
				v.AttackAnimation, err = decodeActorAnimation(level, attackRoot-levelBase, add)
				if err != nil {
					return nil, err
				}
				v.ShotSprite, err = add(shotImage)
				if err != nil {
					return nil, err
				}
				v.ShotMode = "turning-projectile"
				v.ShotMotionBudget = int(binary.BigEndian.Uint16(level[0x54:]))
			}
			if levelNumber == 5 {
				reverseRoot, shotRoot := 0x5555e, 0x5574e
				v.ShotOffsetX, v.ShotDirections = 32, []int{2}
				if variant == 1 {
					reverseRoot, shotRoot = 0x55582, 0x557ae
					v.ShotOffsetX, v.ShotDirections = -32, []int{6}
				}
				v.ReverseAnimation, err = decodeActorAnimation(level, reverseRoot-levelBase, add)
				if err != nil {
					return nil, err
				}
				v.ShotAnimation, err = decodeActorAnimation(level, shotRoot-levelBase, add)
				if err != nil {
					return nil, err
				}
				v.ShotMode = "animated-aiming-projectile"
				v.ShotSpeed = int(binary.BigEndian.Uint16(level[0x4c:]))
			}
			if levelNumber == 1 || levelNumber == 2 {
				attackRoot := 0x551f8 - levelBase
				if variant == 1 {
					attackRoot = 0x55236 - levelBase
				}
				if levelNumber == 2 {
					attackRoot = 0x55d42 - levelBase
					if variant == 0 {
						attackRoot = 0x55d00 - levelBase
					}
				}
				attack, err := decodeActorAnimation(level, attackRoot, add)
				if err != nil {
					return nil, err
				}
				v.AttackAnimation = attack
				if levelNumber == 1 {
					v.ShotMode = "point-burst"
					v.ShotDirections = []int{3, 2, 1}
					if variant == 1 {
						v.ShotDirections = []int{7, 6, 5}
					}
					v.ShotSpeed = int(binary.BigEndian.Uint16(level[0x56:]))
					v.ShotOffsetY = 7
				} else {
					v.ShotMode = "point"
					v.ShotDirections = []int{2}
					v.ShotOffsetX = 3
					if variant == 1 {
						v.ShotDirections = []int{6}
						v.ShotOffsetX = 11
					}
					v.ShotOffsetY = 7
					v.ShotSpeed = int(binary.BigEndian.Uint16(level[0x4c:]))
					name, err := add(0x5ccf6)
					if err != nil {
						return nil, err
					}
					v.ShotSprite = name
				}
			}
			kind.Variants = append(kind.Variants, v)
		}
		result.Kinds = append(result.Kinds, kind)
	}
	if levelNumber == 4 {
		crawler, err := decodeFourthCrawlerKind(level, add)
		if err != nil {
			return nil, err
		}
		result.Kinds = append(result.Kinds, crawler)
	}
	var err error
	if levelNumber == 2 {
		result.HatchCreatures, err = decodeHatchCreatureArtwork(level, add)
		if err != nil {
			return nil, err
		}
		result.PodCreatures, err = decodePodCreatureArtwork(level, add)
		if err != nil {
			return nil, err
		}
	}
	result.Projectile, err = decodeFixedProjectileArtwork(levelNumber, level, add)
	if err != nil {
		return nil, err
	}
	if levelNumber == 3 {
		var err error
		result.Third, err = decodeThirdFixedArt(level, add)
		if err != nil {
			return nil, err
		}
	}
	result.Atlas = packSprites(images)
	result.Atlas.SourceSpriteNames = names
	return result, nil
}
