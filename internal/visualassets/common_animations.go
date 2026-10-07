package visualassets

import (
	"encoding/binary"
	"fmt"
)

type NamedActorAnimation struct {
	ID          string         `json:"id"`
	ResourceTag int            `json:"resource_tag,omitempty"`
	Action      string         `json:"action,omitempty"`
	SoundEffect string         `json:"sound_effect,omitempty"`
	Ending      string         `json:"ending"`
	Animation   ActorAnimation `json:"animation"`
}

// DecodeCommonAnimations exports named common effects and all nineteen reward
// image sequences. Native loop/removal callbacks become ordinary ending names.
func DecodeCommonAnimations(common []byte, resolve func(int) (string, error)) ([]NamedActorAnimation, error) {
	if len(common) < 0x3728+19*6 {
		return nil, fmt.Errorf("common reward table is truncated")
	}
	definitions := []struct {
		id        string
		tag, root int
	}{{"power-up-carrier", 100, 0x2564}, {"player-invulnerability", 0, 0x25a0}, {"cash-small", 0, 0x26ae}, {"cash-large", 0, 0x264e}, {"player-death", 0, 0x2760},
		{"cannon-idle", 0, 0x280a}, {"cannon-fire", 0, 0x27f2}, {"cannon-ball", 0, 0x288e}, {"launcher-idle", 0, 0x2c24}, {"launcher-flight", 0, 0x2918}, {"laser-active", 0, 0x2612}, {"flamer-active", 0, 0x2540}, {"drone-active", 0, 0x25be}, {"electro-ball-active", 0, 0x2a02}, {"mine-small-active", 0, 0x28b2}, {"mine-large-active", 0, 0x28d0},
		{"flamer-particle", 0, 0x2796},
		{"bomb-flight", 0, 0x25e2}, {"rear-shot-active", 0, 0x283a},
		{"explosion-small", 0, 0x270e}, {"explosion-large", 0, 0x273a},
		{"launcher-fire", 0, 0x2c2a}, {"rear-shot-fire", 0, 0x2840}, {"cannon-support", 0, 0x28ee},
	}
	if len(common) < 0x3aa2+32 {
		return nil, fmt.Errorf("homing animation table is truncated")
	}
	for heading := range 8 {
		definitions = append(definitions, struct {
			id        string
			tag, root int
		}{fmt.Sprintf("homing-%d", heading), 0, int(binary.BigEndian.Uint32(common[0x3aa2+heading*4:]))})
	}
	baseCount := len(definitions)
	for i := range 19 {
		field := 0x3728 + i*6
		definitions = append(definitions, struct {
			id        string
			tag, root int
		}{fmt.Sprintf("pickup-%d", i), int(binary.BigEndian.Uint16(common[field:])), int(binary.BigEndian.Uint32(common[field+2:]))})
	}
	result := make([]NamedActorAnimation, 0, len(definitions))
	actions := []string{"speedup", "autofire", "health-power-1", "health-power-2", "rear-shot", "side-shot", "powerup", "cannon", "drone", "laser", "electro-ball", "mine-small", "missile-launcher", "homing-missile", "flamer", "bomb", "invulnerability", "dive", "screen-clear"}
	callbacks := []uint32{0x4906, 0x48f6, 0x4894, 0x48b4, 0x5206, 0x4920, 0x48bc, 0x543a, 0x4f66, 0x538c, 0x5014, 0x50ac, 0x52e0, 0x4dc6, 0x4e74, 0x4d28, 0x4a94, 0x491a, 0x4926}
	for definitionIndex, definition := range definitions {
		animation, ending, err := decodeCommonAnimation(common, definition.root, resolve)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", definition.id, err)
		}
		entry := NamedActorAnimation{ID: definition.id, ResourceTag: definition.tag, Ending: ending, Animation: animation}
		if definitionIndex >= baseCount {
			pickupIndex := definitionIndex - baseCount
			if len(common) < 0x47d6+(pickupIndex+1)*4 {
				return nil, fmt.Errorf("pickup action table is truncated")
			}
			if binary.BigEndian.Uint32(common[0x47d6+pickupIndex*4:])&0x00ffffff != callbacks[pickupIndex] {
				return nil, fmt.Errorf("unsupported pickup %d initializer", pickupIndex)
			}
			entry.Action = actions[pickupIndex]
			entry.SoundEffect = "sampled-effect-01"
			if common[0x47d6+pickupIndex*4] != 0 {
				entry.SoundEffect = "synthesized-effect-04"
			}
			// Health and screen clearing replace the selector's queued sound.
			if pickupIndex == 2 || pickupIndex == 3 {
				entry.SoundEffect = "synthesized-effect-07"
			}
			if pickupIndex == 18 {
				entry.SoundEffect = "synthesized-effect-02"
			}
		}
		result = append(result, entry)
	}
	static, err := resolve(0x192ba)
	if err != nil {
		return nil, err
	}
	result = append(result, NamedActorAnimation{ID: "double-shot-active", Ending: "hold", Animation: ActorAnimation{Frames: []AnimationFrame{{Sprite: static}}, Static: true}})
	return result, nil
}

func decodeCommonAnimation(common []byte, start int, resolve func(int) (string, error)) (ActorAnimation, string, error) {
	animation := ActorAnimation{}
	offsets := map[int]int{}
	cursor := start
	if start < 0 || start+6 > len(common) {
		return animation, "", fmt.Errorf("common animation root is outside source")
	}
	for len(animation.Frames) < 256 {
		if cursor+6 > len(common) {
			return animation, "", fmt.Errorf("common animation is truncated")
		}
		image := int(binary.BigEndian.Uint32(common[cursor:]))
		if image == 0 {
			if cursor+8 > len(common) {
				return animation, "", fmt.Errorf("common animation ending is truncated")
			}
			callback := binary.BigEndian.Uint32(common[cursor+4:])
			switch callback {
			case 0x2c82, 0xe2a:
				if cursor+12 > len(common) {
					return animation, "", fmt.Errorf("common animation loop is truncated")
				}
				loop := int(binary.BigEndian.Uint32(common[cursor+8:]))
				index, ok := offsets[loop]
				if !ok {
					if loop < 0 || loop+6 > len(common) || binary.BigEndian.Uint16(common[loop+4:]) != 0 {
						return animation, "", fmt.Errorf("common animation loop leaves its frame list")
					}
					name, err := resolve(int(binary.BigEndian.Uint32(common[loop:])))
					if err != nil {
						return animation, "", err
					}
					animation.Frames = append(animation.Frames, AnimationFrame{Sprite: name})
					return animation, "hold", nil
				}
				animation.LoopFrom = index
				return animation, "loop", nil
			case 0x2c92, 0xe30:
				return animation, "remove", nil
			default:
				return animation, "", fmt.Errorf("unsupported common animation ending")
			}
		}
		name, err := resolve(image)
		if err != nil {
			return animation, "", err
		}
		duration := int(binary.BigEndian.Uint16(common[cursor+4:]))
		offsets[cursor] = len(animation.Frames)
		animation.Frames = append(animation.Frames, AnimationFrame{Sprite: name, Duration: duration})
		cursor += 6
		if duration == 0 {
			animation.Static = len(animation.Frames) == 1
			return animation, "hold", nil
		}
	}
	return animation, "", fmt.Errorf("common animation has too many frames")
}
