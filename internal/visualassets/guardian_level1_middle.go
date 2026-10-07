package visualassets

import (
	"encoding/binary"
	"fmt"
)

func decodeFirstMiddleDefense(level []byte, add func(int) (string, error)) (GuardianGroup, error) {
	group := GuardianGroup{ID: "middle-defense-streams", MotionParameters: map[string]int{"stream_count": 5, "motion_budget": int(binary.BigEndian.Uint16(level[0x62:]))}}
	const table = 0x55ce6 - levelBase
	if len(level) < 0x55e2c-levelBase || binary.BigEndian.Uint16(level[table:]) != 10 {
		return group, fmt.Errorf("first defense descriptors are incomplete")
	}
	for i := range 11 {
		at := table + 2 + i*12
		sprite, err := add(int(binary.BigEndian.Uint32(level[at+6:])))
		if err != nil {
			return group, err
		}
		component := GuardianComponent{Index: i, Behavior: "first-defense-follower", DamageBehavior: "first-defense-fragment", ParentIndex: i - 1,
			ResourceTag: int(binary.BigEndian.Uint16(level[at:])), Health: 1, Score: int(binary.BigEndian.Uint16(level[at+10:])), InitialDelay: -i * 7, Sprite: sprite, RenderMode: "sprite", Animation: ActorAnimation{Static: true, Frames: []AnimationFrame{{Sprite: sprite}}}}
		headingRoot, deathRoot := 0x56194, 0x55250
		if i == 0 {
			headingRoot, deathRoot = 0x56174, 0x55274
		}
		for heading := range 8 {
			name, err := add(int(binary.BigEndian.Uint32(level[headingRoot-levelBase+heading*4:])))
			if err != nil {
				return group, err
			}
			component.HeadingFrames = append(component.HeadingFrames, name)
		}
		clip, err := decodeActorAnimation(level, deathRoot-levelBase, add)
		if err != nil {
			return group, err
		}
		component.DeathAnimation = clip
		group.Components = append(group.Components, component)
	}
	for i := range 16 {
		at := 0x55d6c - levelBase + i*12
		commands, err := decodeTerminatedGuardianPath(level, int(binary.BigEndian.Uint32(level[at+8:]))-levelBase)
		if err != nil {
			return group, err
		}
		group.Launches = append(group.Launches, GuardianLaunch{Index: i, X: int(binary.BigEndian.Uint16(level[at:])), WorldY: int(binary.BigEndian.Uint16(level[at+2:])), GateID: int(binary.BigEndian.Uint16(level[at+4:])), GateState: int(binary.BigEndian.Uint16(level[at+6:])), Path: Path{ID: i + 1, Commands: commands}})
		gateAt := 0x558a2 - levelBase + i*10
		cell := int(binary.BigEndian.Uint16(level[gateAt:])) / 2
		gate := GuardianGate{ID: i, Column: cell % 20, Row: cell / 20}
		for frame := range 4 {
			gate.Frames = append(gate.Frames, readTilePatch(level, gateAt+2+frame*2, 1, 1))
		}
		group.Gates = append(group.Gates, gate)
	}
	return group, nil
}
