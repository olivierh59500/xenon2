package visualassets

import (
	"encoding/binary"
	"fmt"
)

type GuardianLaunch struct {
	Index     int  `json:"index"`
	X         int  `json:"x"`
	WorldY    int  `json:"world_y"`
	GateID    int  `json:"gate_id"`
	GateState int  `json:"gate_state"`
	Path      Path `json:"path"`
}

type GuardianTerrainCell struct {
	Quadrant           int    `json:"quadrant"`
	X                  int    `json:"x"`
	WorldY             int    `json:"world_y"`
	InitiallyDestroyed bool   `json:"initially_destroyed"`
	DestroyedTile      uint16 `json:"destroyed_tile"`
}

func decodeSecondGuardianCells(level []byte) ([]GuardianTerrainCell, error) {
	mapStart, err := offset(level, 4, 12000)
	if err != nil {
		return nil, err
	}
	start := mapStart - 360
	if start < 0 {
		return nil, fmt.Errorf("guardian collision cells are truncated")
	}
	var result []GuardianTerrainCell
	for quadrant, relative := range []int{98, 278, 0, 196} {
		cursor := start + relative
		count := int(binary.BigEndian.Uint16(level[cursor:]))
		cursor += 2
		if count > 32 || cursor+count*8 > mapStart {
			return nil, fmt.Errorf("guardian collision quadrant is invalid")
		}
		for i := range count {
			entry := level[cursor+i*8 : cursor+(i+1)*8]
			result = append(result, GuardianTerrainCell{Quadrant: quadrant, X: int(int16(binary.BigEndian.Uint16(entry[2:]))), WorldY: int(int16(binary.BigEndian.Uint16(entry[4:]))), InitiallyDestroyed: binary.BigEndian.Uint16(entry) != 0, DestroyedTile: binary.BigEndian.Uint16(entry[6:])})
		}
	}
	return result, nil
}

func decodeSecondLevelDefense(level []byte, add func(int) (string, error)) (GuardianGroup, error) {
	const table = 0x55ec4 - levelBase
	group := GuardianGroup{ID: "middle-defense-wave"}
	if len(level) < table+146 || binary.BigEndian.Uint16(level[table:]) != 11 {
		return group, fmt.Errorf("middle defense part table is invalid")
	}
	for index := range 12 {
		entry := level[table+2+index*12 : table+2+(index+1)*12]
		name, err := add(int(binary.BigEndian.Uint32(entry[6:])))
		if err != nil {
			return group, err
		}
		behavior := "defense-link"
		if index == 0 {
			behavior = "defense-head"
		}
		if index == 11 {
			behavior = "defense-tail"
		}
		part := GuardianComponent{Index: index, ResourceTag: int(binary.BigEndian.Uint16(entry)), Behavior: behavior, DamageBehavior: "defense-part-damage", ParentIndex: index - 1, PathBudget: int(binary.BigEndian.Uint16(level[0x54:])), Health: int(binary.BigEndian.Uint16(level[0x52:])), Score: int(binary.BigEndian.Uint16(entry[10:])), InitialDelay: -index * 10, Sprite: name, RenderMode: "sprite", Animation: ActorAnimation{Frames: []AnimationFrame{{Sprite: name}}, Static: true}}
		group.Components = append(group.Components, part)
	}
	const launches = 0x55f56 - levelBase
	for index := range 16 {
		entry := level[launches+index*12 : launches+(index+1)*12]
		root := int(binary.BigEndian.Uint32(entry[8:])) - levelBase
		commands, err := decodeTerminatedGuardianPath(level, root)
		if err != nil {
			return group, err
		}
		group.Launches = append(group.Launches, GuardianLaunch{Index: index, X: int(int16(binary.BigEndian.Uint16(entry))), WorldY: int(int16(binary.BigEndian.Uint16(entry[2:]))), GateID: int(binary.BigEndian.Uint16(entry[4:])), GateState: int(binary.BigEndian.Uint16(entry[6:])), Path: Path{ID: index + 1, Commands: commands}})
	}
	return group, nil
}

func decodeTerminatedGuardianPath(data []byte, start int) ([]PathCommand, error) {
	if start < 0 || start+2 > len(data) {
		return nil, fmt.Errorf("guardian path root is invalid")
	}
	cursor := start
	for count := 0; count < 1024; count++ {
		if cursor+2 > len(data) {
			return nil, fmt.Errorf("guardian path is truncated")
		}
		opcode := binary.BigEndian.Uint16(data[cursor:])
		length := 0
		switch opcode {
		case 0:
			return decodePath(data[start : cursor+2])
		case 2:
			length = 10
		case 4:
			length = 4
		case 6:
			length = 18
		case 8:
			length = 4
		case 10:
			length = 6
		default:
			return nil, fmt.Errorf("unsupported guardian path command")
		}
		cursor += length
	}
	return nil, fmt.Errorf("guardian path has no terminator")
}
