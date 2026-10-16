package visualassets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
)

type AnimationFrame struct {
	Sprite   string `json:"sprite"`
	Duration int    `json:"duration"`
	// The second guardian inherits this source animator decision when its
	// preceding path member is still waiting to enter.
	ReverseWhenAdvanced bool `json:"reverse_when_advanced,omitempty"`
}

type ActorAnimation struct {
	Frames   []AnimationFrame `json:"frames"`
	LoopFrom int              `json:"loop_from"`
	Static   bool             `json:"static,omitempty"`
	Ending   string           `json:"ending,omitempty"`
}

type ActorPart struct {
	Atlas           string           `json:"atlas,omitempty"`
	ResourceTag     int              `json:"resource_tag"`
	Score           int              `json:"score"`
	StrongHealth    bool             `json:"strong_health"`
	Linked          bool             `json:"linked"`
	DamageMode      string           `json:"damage_mode"`
	MotionMode      string           `json:"motion_mode"`
	HeadingFrames   []string         `json:"heading_frames,omitempty"`
	HeadingOffset   int              `json:"heading_offset,omitempty"`
	HeadingShift    int              `json:"heading_shift,omitempty"`
	EntryAnimations []ActorAnimation `json:"entry_animations,omitempty"`
	Animation       ActorAnimation   `json:"animation"`
}

type WaveActor struct {
	Kind                          int         `json:"kind"`
	Parts                         []ActorPart `json:"parts"`
	StrongHealthOverride          int         `json:"strong_health_override,omitempty"`
	MotionBudgetOverride          int         `json:"motion_budget_override,omitempty"`
	CarriedRewardFromMotionBudget bool        `json:"carried_reward_from_motion_budget,omitempty"`
}

type Actors struct {
	Kinds []WaveActor `json:"kinds"`
	Atlas SpriteAtlas `json:"atlas"`
}

// DecodeWaveActors recognizes the verified offline selector wrappers, then
// exports only ordinary actor descriptors and their original image animations.
// Selector instructions and routine addresses do not reach the JSON resource.
func DecodeWaveActors(level []byte, palette [16][4]uint8, encounters *Encounters, commonArt *SpriteAtlas) (*Actors, error) {
	if len(level) < 0x14 || binary.BigEndian.Uint16(level[0x10:]) != 0x6000 {
		return nil, fmt.Errorf("unsupported actor selector header")
	}
	selector := 0x12 + int(int16(binary.BigEndian.Uint16(level[0x12:])))
	pattern := []byte{0xd0, 0x40, 0xd0, 0x40, 0x24, 0x7b, 0, 4, 0x4e, 0xd2}
	if selector < 0 || selector+10 > len(level) || !bytes.Equal(level[selector:selector+10], pattern) {
		return nil, fmt.Errorf("unsupported actor selector layout")
	}
	unique := map[int]bool{}
	for _, wave := range encounters.Moving {
		unique[wave.EnemyKind] = true
	}
	kinds := make([]int, 0, len(unique))
	for kind := range unique {
		kinds = append(kinds, kind)
	}
	sort.Ints(kinds)
	actors := &Actors{}
	images := make([]*Sprite, 0)
	imageNames := map[int]string{}
	addImage := func(address int) (string, error) {
		if name, ok := imageNames[address]; ok {
			return name, nil
		}
		name := fmt.Sprintf("enemy-image-%03d", len(images))
		picture, err := DecodeActorSprite(level, address-levelBase, name, palette)
		if err != nil {
			return "", err
		}
		images = append(images, picture)
		imageNames[address] = name
		return name, nil
	}
	for _, kind := range kinds {
		field := selector + 10 + kind*4
		if kind < 0 || field+4 > len(level) {
			return nil, fmt.Errorf("actor kind %d leaves its selector", kind)
		}
		address := int(binary.BigEndian.Uint32(level[field:]))
		if address == 0xe5a {
			if commonArt == nil {
				return nil, fmt.Errorf("carrier requires the common animation bank")
			}
			var carrier *NamedActorAnimation
			for i := range commonArt.Animations {
				if commonArt.Animations[i].ID == "power-up-carrier" {
					carrier = &commonArt.Animations[i]
					break
				}
			}
			if carrier == nil {
				return nil, fmt.Errorf("carrier common animation is missing")
			}
			part := ActorPart{Atlas: "common", ResourceTag: 100, DamageMode: "drop-equipment", MotionMode: "path", Animation: carrier.Animation}
			actors.Kinds = append(actors.Kinds, WaveActor{Kind: kind, Parts: []ActorPart{part}, MotionBudgetOverride: 3, CarriedRewardFromMotionBudget: true})
			continue
		}
		wrapper := address - levelBase
		if wrapper < 0 || wrapper+10 > len(level) {
			return nil, fmt.Errorf("actor kind %d wrapper is outside level", kind)
		}
		actor := WaveActor{Kind: kind}
		descriptorField := wrapper + 2
		if bytes.Equal(level[wrapper:wrapper+4], []byte{0x3f, 0x38, 4, 0x26}) {
			if wrapper+28 > len(level) || !bytes.Equal(level[wrapper+4:wrapper+6], []byte{0x31, 0xf9}) || !bytes.Equal(level[wrapper+12:wrapper+14], []byte{0x45, 0xf9}) || !bytes.Equal(level[wrapper+18:wrapper+28], []byte{0x4e, 0xb8, 0x0e, 0x60, 0x31, 0xdf, 4, 0x26, 0x4e, 0x75}) {
				return nil, fmt.Errorf("unsupported actor health override")
			}
			health, err := offset(level, wrapper+6, 2)
			if err != nil {
				return nil, err
			}
			actor.StrongHealthOverride = int(binary.BigEndian.Uint16(level[health:]))
			descriptorField = wrapper + 14
		} else if !bytes.Equal(level[wrapper:wrapper+2], []byte{0x45, 0xf9}) || !bytes.Equal(level[wrapper+6:wrapper+10], []byte{0x4e, 0xf8, 0x0e, 0x60}) {
			return nil, fmt.Errorf("unsupported actor kind %d wrapper", kind)
		}
		descriptor, err := offset(level, descriptorField, 24)
		if err != nil {
			return nil, err
		}
		for partIndex := 0; partIndex < 64; partIndex++ {
			if descriptor+26 > len(level) {
				return nil, fmt.Errorf("actor descriptor is truncated")
			}
			resource := int(binary.BigEndian.Uint16(level[descriptor:]))
			if resource == 0 {
				break
			}
			updater := binary.BigEndian.Uint32(level[descriptor+4:])
			renderer := binary.BigEndian.Uint32(level[descriptor+8:])
			damage := binary.BigEndian.Uint32(level[descriptor+12:])
			headingTable := 0
			switch updater {
			case 0x55044:
				headingTable = 0x55078 - levelBase
			case 0x5504e:
				headingTable = 0x55098 - levelBase
			case 0x55058:
				headingTable = 0x550b8 - levelBase
			}
			if updater != 0xe0c && updater != 0xe24 && updater != 0x550d8 && headingTable == 0 || renderer != 0xe12 || damage != 0xe18 && damage != 0xe1e {
				return nil, fmt.Errorf("unsupported actor kind %d behavior", kind)
			}
			part := ActorPart{ResourceTag: resource, Score: int(binary.BigEndian.Uint16(level[descriptor+2:])), StrongHealth: level[descriptor+22] != 0, Linked: level[descriptor+23] != 0, DamageMode: "individual", MotionMode: "path"}
			if updater == 0xe24 {
				part.MotionMode = "follow-leader"
			}
			if headingTable != 0 {
				if headingTable+32 > len(level) {
					return nil, fmt.Errorf("heading image table is truncated")
				}
				part.MotionMode = "path-heading-frames"
				part.HeadingOffset = 16
				part.HeadingShift = 5
				for i := range 8 {
					name, err := addImage(int(binary.BigEndian.Uint32(level[headingTable+i*4:])))
					if err != nil {
						return nil, err
					}
					part.HeadingFrames = append(part.HeadingFrames, name)
				}
			}
			if updater == 0x550d8 {
				const edgeTable = 0x5513e - levelBase
				if edgeTable+16 > len(level) {
					return nil, fmt.Errorf("entry-edge animation table is truncated")
				}
				part.MotionMode = "path-entry-edge-frames"
				for edge := range 4 {
					start, err := offset(level, edgeTable+edge*4, 6)
					if err != nil {
						return nil, err
					}
					animation, err := decodeActorAnimation(level, start, addImage)
					if err != nil {
						return nil, err
					}
					part.EntryAnimations = append(part.EntryAnimations, animation)
				}
			}
			if damage == 0xe1e {
				part.DamageMode = "group"
			}
			animation, err := offset(level, descriptor+16, 6)
			if err != nil {
				return nil, err
			}
			part.Animation, err = decodeActorAnimation(level, animation, addImage)
			if err != nil {
				return nil, fmt.Errorf("actor kind %d animation: %w", kind, err)
			}
			actor.Parts = append(actor.Parts, part)
			descriptor += 24
			if partIndex == 63 {
				return nil, fmt.Errorf("actor has too many linked parts")
			}
		}
		actors.Kinds = append(actors.Kinds, actor)
	}
	actors.Atlas = packSprites(images)
	actors.Atlas.SourceSpriteNames = imageNames
	return actors, nil
}

func decodeActorAnimation(level []byte, start int, addImage func(int) (string, error)) (ActorAnimation, error) {
	animation := ActorAnimation{}
	if start < 0 || start+6 > len(level) {
		return animation, fmt.Errorf("animation root is outside its source")
	}
	offsets := map[int]int{}
	cursor := start
	joined := false
	for len(animation.Frames) < 256 {
		if cursor+6 > len(level) {
			return animation, fmt.Errorf("animation is truncated")
		}
		address := int(binary.BigEndian.Uint32(level[cursor:]))
		if address == 0 {
			if cursor+8 > len(level) {
				return animation, fmt.Errorf("animation ending is truncated")
			}
			if binary.BigEndian.Uint32(level[cursor+4:]) == 0xe30 {
				animation.Ending = "remove"
				return animation, nil
			}
			if cursor+12 > len(level) || binary.BigEndian.Uint32(level[cursor+4:]) != 0xe2a {
				return animation, fmt.Errorf("unsupported animation ending")
			}
			loop := int(binary.BigEndian.Uint32(level[cursor+8:])) - levelBase
			index, ok := offsets[loop]
			if !ok {
				if loop < 0 || loop+6 > len(level) {
					return animation, fmt.Errorf("animation loop leaves its source")
				}
				// Some directed lists join a shared animation tail. Flatten
				// that tail into named frames until its loop or held ending.
				cursor = loop
				joined = true
				continue
			}
			animation.LoopFrom = index
			animation.Ending = "loop"
			return animation, nil
		}
		name, err := addImage(address)
		if err != nil {
			return animation, err
		}
		duration := int(binary.BigEndian.Uint16(level[cursor+4:]))
		offsets[cursor] = len(animation.Frames)
		animation.Frames = append(animation.Frames, AnimationFrame{Sprite: name, Duration: duration, ReverseWhenAdvanced: !joined && address&0x4000 != 0})
		joined = false
		cursor += 6
		if duration == 0 {
			animation.Static = len(animation.Frames) == 1
			animation.Ending = "hold"
			return animation, nil
		}
	}
	return animation, fmt.Errorf("animation has too many frames")
}
