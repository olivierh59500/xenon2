package visualassets

import (
	"encoding/binary"
	"fmt"
)

func decodeFourthGuardianControllerData(level []byte, middle, final *GuardianGroup, add func(int) (string, error)) error {
	if len(level) < 0x568f2-levelBase {
		return fmt.Errorf("fourth guardian controller data is truncated")
	}
	middle.MotionParameters = map[string]int{"head_attack_chance": int(level[0x59]), "satellite_fire_rate": int(level[0x5b]), "satellite_shot_speed": int(binary.BigEndian.Uint16(level[0x5c:])), "companion_fire_rate": int(level[0x73]), "companion_shot_speed": int(binary.BigEndian.Uint16(level[0x74:])), "tail_fire_rate": int(level[0x7f]), "tail_shot_speed": int(binary.BigEndian.Uint16(level[0x80:])), "outer_targets": 5, "core_score": 2000, "reward_cash_pairs": 5, "arena_minimum_scroll": 0, "arena_maximum_scroll": 2480, "defeated_scroll": 2208}
	final.MotionParameters = map[string]int{"alarm_rate": int(level[0x53]), "eye_fire_rate": int(level[0x55]), "eye_shot_speed": int(binary.BigEndian.Uint16(level[0x56:])), "eye_count": 2, "arena_minimum_scroll": 16, "cash_pairs": 10, "explosion_count": 20}
	middle.MotionTables = map[string][]int{}
	final.MotionTables = map[string][]int{}
	readWords := func(address, count int) []int {
		values := make([]int, count)
		for i := range count {
			values[i] = int(int16(binary.BigEndian.Uint16(level[address-levelBase+i*2:])))
		}
		return values
	}
	readBytes := func(address, count int) []int {
		values := make([]int, count)
		for i := range count {
			values[i] = int(level[address-levelBase+i])
		}
		return values
	}
	final.MotionTables["body_vertical_offsets"] = readWords(0x55c54, 16)
	final.MotionTables["left_eye_headings"] = readBytes(0x55ec8, 8)
	final.MotionTables["right_eye_headings"] = readBytes(0x55ed0, 8)
	final.MotionTables["eye_shot_offset_x"] = readWords(0x55e10, 8)
	final.MotionTables["eye_shot_offset_y"] = readWords(0x55e20, 8)
	middle.MotionTables["companion_shot_offset_x"] = readWords(0x566a0, 8)
	middle.MotionTables["companion_shot_offset_y"] = readWords(0x566b0, 8)
	readImages := func(table, count int) ([]string, error) {
		result := make([]string, count)
		for i := range count {
			address := int(binary.BigEndian.Uint32(level[table-levelBase+i*4:]))
			var err error
			result[i], err = add(address)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	readClips := func(table int) ([]ActorAnimation, error) {
		result := make([]ActorAnimation, 8)
		for i := range 8 {
			root := int(binary.BigEndian.Uint32(level[table-levelBase+i*4:]))
			if root == 0 {
				continue
			}
			clip, err := decodeActorAnimation(level, root-levelBase, add)
			if err != nil {
				return nil, err
			}
			result[i] = clip
		}
		return result, nil
	}
	var err error
	headImages, err := readImages(0x56576, 4)
	if err != nil {
		return err
	}
	for i := 0; i < 4; i++ {
		middle.Components[i].HeadingFrames = append([]string(nil), headImages...)
	}
	middle.Components[4].HeadingFrames, err = readImages(0x565da, 8)
	if err != nil {
		return err
	}
	middle.Components[5].HeadingAnimations, err = readClips(0x56680)
	if err != nil {
		return err
	}
	satellites, err := readClips(0x568d2)
	if err != nil {
		return err
	}
	for i := 16; i < 20; i++ {
		middle.Components[i].HeadingAnimations = append([]ActorAnimation(nil), satellites...)
	}
	shot, err := decodeActorAnimation(level, 0x56552-levelBase, add)
	if err != nil {
		return err
	}
	middle.Animations = append(middle.Animations, NamedActorAnimation{ID: "middle-shot", Ending: shot.Ending, Animation: shot})
	satellite, err := add(0x5fda6)
	if err != nil {
		return err
	}
	middle.Animations = append(middle.Animations, NamedActorAnimation{ID: "middle-satellite-shot", Ending: "hold", Animation: ActorAnimation{Frames: []AnimationFrame{{Sprite: satellite}}, Static: true}})
	for heading := 2; heading <= 6; heading++ {
		root := int(binary.BigEndian.Uint32(level[0x55e30-levelBase+heading*4:]))
		clip, err := decodeActorAnimation(level, root-levelBase, add)
		if err != nil {
			return err
		}
		final.Animations = append(final.Animations, NamedActorAnimation{ID: fmt.Sprintf("eye-shot-%d", heading), Ending: clip.Ending, Animation: clip})
	}
	for eye := 1; eye <= 2; eye++ {
		table := 0x55ba0
		if eye == 2 {
			table = 0x55bc4
		}
		root := int(binary.BigEndian.Uint32(level[table-levelBase-4:])) - levelBase
		patch := readTilePatch(level, root, 2, 2)
		final.Components[eye].DestroyedTiles = &patch
	}
	final.Components[3].ParentIndex = 0
	return nil
}
