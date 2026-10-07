package visualassets

import (
	"encoding/binary"
	"fmt"
)

func decodeThirdFinalGuardian(level []byte, add func(int) (string, error)) (GuardianGroup, error) {
	const table = 0x55270 - levelBase
	group := GuardianGroup{ID: "final-guardian", MotionParameters: map[string]int{"initial_health": int(binary.BigEndian.Uint16(level[0x6a:])), "path_budget": int(binary.BigEndian.Uint16(level[0x42:])), "fire_rate": int(level[0x6f]), "shot_speed": int(binary.BigEndian.Uint16(level[0x70:]))}}
	shot, err := decodeActorAnimation(level, 0x5549e-levelBase, add)
	if err != nil {
		return group, err
	}
	group.Animations = append(group.Animations, NamedActorAnimation{ID: "final-worm-shot", Ending: shot.Ending, Animation: shot})
	if binary.BigEndian.Uint16(level[table:]) != 10 {
		return group, fmt.Errorf("third final guardian member count differs")
	}
	delay := 0
	for index := range 11 {
		at := table + 2 + index*10
		updater := binary.BigEndian.Uint32(level[at+2:])
		behavior, bank := "worm-link", 0x555d2
		switch updater {
		case 0x55420:
			behavior, bank = "worm-head", 0x55400
		case 0x5556a:
			behavior, bank = "worm-neck", 0x5558e
		case 0x555ae:
		case 0x555f2:
			behavior, bank = "worm-tail", 0x55622
		default:
			return group, fmt.Errorf("unsupported third final member behavior")
		}
		image, err := add(int(binary.BigEndian.Uint32(level[at+6:])))
		if err != nil {
			return group, err
		}
		part := GuardianComponent{Index: index, Behavior: behavior, DamageBehavior: "block-shot", ParentIndex: -1, ResourceTag: 80, InitialDelay: -delay, PathBudget: group.MotionParameters["path_budget"], Sprite: image, RenderMode: "sprite", Animation: ActorAnimation{Frames: []AnimationFrame{{Sprite: image}}, Static: true, Ending: "hold"}}
		if index == 0 {
			part.DamageBehavior = "worm-head-health"
			part.Health = group.MotionParameters["initial_health"]
		}
		for heading := range 8 {
			address := int(binary.BigEndian.Uint32(level[bank-levelBase+heading*4:]))
			if index == 0 {
				animation, err := decodeActorAnimation(level, address-levelBase, add)
				if err != nil {
					return group, err
				}
				part.HeadingAnimations = append(part.HeadingAnimations, animation)
			} else {
				name, err := add(address)
				if err != nil {
					return group, err
				}
				part.HeadingFrames = append(part.HeadingFrames, name)
			}
		}
		group.Components = append(group.Components, part)
		delay += int(binary.BigEndian.Uint16(level[at:]))
	}
	pathTable := int(binary.BigEndian.Uint32(level[0x30:]))
	for index := range 8 {
		group.Launches = append(group.Launches, GuardianLaunch{Index: index, PathID: (0x6ef36-pathTable)/4 + index + 1})
	}
	return group, nil
}
