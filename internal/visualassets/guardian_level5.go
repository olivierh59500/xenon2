package visualassets

import (
	"encoding/binary"
	"fmt"
)

func enrichFifthGuardianArt(level []byte, group *GuardianGroup, add func(int) (string, error)) error {
	const base = 0x54e00
	group.MotionParameters = map[string]int{"initial_move_remaining": 4, "body_fire_rate": int(level[0x54e5b-base]), "body_laser_rate": int(level[0x54e7f-base]), "mount_fire_rate": int(level[0x54e57-base]), "mount_shot_speed": int(binary.BigEndian.Uint16(level[0x54e58-base:]))}
	if group.ID == "middle-guardian" {
		group.MotionParameters["body_laser_speed"] = int(binary.BigEndian.Uint16(level[0x54e5c-base:]))
		group.MotionParameters["maximum_scroll"] = 0x920
		group.MotionParameters["minimum_world_y"] = 0x8d0
		group.MotionParameters["maximum_world_y"] = 0x910
		group.MotionParameters["initial_x"] = 112
	} else {
		group.MotionParameters["maximum_scroll"] = 416
		group.Components[0].TileFrames = []TilePatch{readTilePatch(level, 0x57666-base, 15, 18)}
		group.MotionParameters["side_laser_frame"] = 1
		group.MotionParameters["mouth_creature_speed"] = int(binary.BigEndian.Uint16(level[0x54e52-base:]))
		group.MotionParameters["body_laser_speed"] = int(binary.BigEndian.Uint16(level[0x54e5c-base:]))
		group.MotionParameters["mouth_fire_rate"] = int(level[0x54e51-base])
		group.MotionParameters["side_fire_rate"] = int(level[0x54e55-base])
		group.MotionParameters["core_health"] = int(level[0x5594e-base+22])
	}
	for i := range group.Components {
		part := &group.Components[i]
		if part.Behavior == "final-side-turret" {
			clip, err := decodeFifthPausedAnimation(level, 0x55b70-base, add)
			if err != nil {
				return err
			}
			part.Animation = clip
		}
		table, count := 0, 0
		switch part.Behavior {
		case "follow-middle-body", "final-mount":
			table, count = 0x56658, 8
		case "middle-central-cannon":
			table, count = 0x5650e, 6
		case "middle-corner":
			table, count = 0x565d2, 4
			if part.ResourceTag == 0x118 {
				table = 0x565c2
			}
		}
		if table != 0 {
			for frame := 0; frame < count; frame++ {
				address := int(binary.BigEndian.Uint32(level[table-base+frame*4:]))
				name, err := add(address)
				if err != nil {
					return err
				}
				part.HeadingFrames = append(part.HeadingFrames, name)
			}
			part.Sprite = part.HeadingFrames[0]
		}
	}
	for _, entry := range []struct {
		id   string
		root int
	}{{"fifth-mouth-creature", 0x55f7e}, {"fifth-side-shot", 0x5571e}} {
		clip, err := decodeActorAnimation(level, entry.root-base, add)
		if err != nil {
			return err
		}
		group.Animations = append(group.Animations, NamedActorAnimation{ID: entry.id, Animation: clip})
	}
	group.MotionTables = make(map[string][]int)
	for _, table := range []struct {
		id      string
		address int
	}{{"seeking_x", 0x557fe}, {"seeking_y", 0x5580e}} {
		for i := 0; i < 8; i++ {
			group.MotionTables[table.id] = append(group.MotionTables[table.id], int(int16(binary.BigEndian.Uint16(level[table.address-base+i*2:]))))
		}
	}
	for _, entry := range []struct {
		name    string
		address int
	}{{"fifth-laser-top", 0x5f2fe}, {"fifth-laser-bottom", 0x5f340}} {
		name, err := add(entry.address)
		if err != nil {
			return err
		}
		group.MotionTables[entry.name] = []int{}
		group.Animations = append(group.Animations, NamedActorAnimation{ID: entry.name, Animation: ActorAnimation{Frames: []AnimationFrame{{Sprite: name}}}})
	}
	group.MotionParameters["side_health"] = int(binary.BigEndian.Uint16(level[0x54e4c-base:]))
	group.MotionParameters["mouth_creature_health"] = int(binary.BigEndian.Uint16(level[0x54e52-base:]))
	group.MotionParameters["side_lifetime"] = int(binary.BigEndian.Uint16(level[0x54e4e-base:]))
	table := 0x557de
	if group.ID == "final-guardian" {
		table = 0x5603e
	}
	for i := 0; i < 8; i++ {
		root := int(binary.BigEndian.Uint32(level[table-base+i*4:])) - base
		clip, err := decodeActorAnimation(level, root, add)
		if err != nil {
			return err
		}
		group.Components[0].HeadingAnimations = append(group.Components[0].HeadingAnimations, clip)
	}
	destroyed, err := add(0x5df96)
	if err != nil {
		return err
	}
	for i := range group.Components {
		if group.Components[i].DamageBehavior == "middle-mount-damage" || group.Components[i].ResourceTag == 0x114 {
			group.Components[i].DestroyedSprite = destroyed
		}
	}
	if group.ID != "middle-guardian" && group.ID != "final-guardian" {
		return fmt.Errorf("unknown fifth guardian group")
	}
	return nil
}

func decodeFifthPausedAnimation(level []byte, root int, add func(int) (string, error)) (ActorAnimation, error) {
	const base = 0x54e00
	var result ActorAnimation
	positions := make(map[int]int)
	cursor := root
	for len(result.Frames) < 256 {
		if cursor < 0 || cursor+6 > len(level) {
			return result, fmt.Errorf("fifth paused animation truncated")
		}
		positions[cursor] = len(result.Frames)
		address := int(binary.BigEndian.Uint32(level[cursor:]))
		cursor += 4
		if address == 0 {
			if cursor+4 > len(level) {
				return result, fmt.Errorf("fifth paused animation loop truncated")
			}
			target := int(binary.BigEndian.Uint32(level[cursor:]))
			if target == 0xe2a {
				cursor += 4
				if cursor+4 > len(level) {
					return result, fmt.Errorf("fifth animation callback loop truncated")
				}
				target = int(binary.BigEndian.Uint32(level[cursor:]))
			}
			target -= base
			loop, ok := positions[target]
			if !ok {
				return result, fmt.Errorf("fifth paused animation loop leaves its sequence")
			}
			result.LoopFrom = loop
			return result, nil
		}
		name, err := add(address)
		if err != nil {
			return result, err
		}
		duration := int(binary.BigEndian.Uint16(level[cursor:]))
		cursor += 2
		result.Frames = append(result.Frames, AnimationFrame{Sprite: name, Duration: duration})
	}
	return result, fmt.Errorf("fifth paused animation lacks its loop")
}
