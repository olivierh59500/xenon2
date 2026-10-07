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
	PathID    int  `json:"path_id,omitempty"`
}

type GuardianTerrainCell struct {
	Quadrant           int    `json:"quadrant"`
	X                  int    `json:"x"`
	WorldY             int    `json:"world_y"`
	InitiallyDestroyed bool   `json:"initially_destroyed"`
	RestoredTile       uint16 `json:"restored_tile"`
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
			// Stage initialization marks every cell intact, independently of
			// the unused flags stored in the recovered resource file.
			result = append(result, GuardianTerrainCell{Quadrant: quadrant, X: int(int16(binary.BigEndian.Uint16(entry[2:]))), WorldY: int(int16(binary.BigEndian.Uint16(entry[4:]))), InitiallyDestroyed: false, RestoredTile: binary.BigEndian.Uint16(entry[6:])})
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
		headingTable, deathRoot := 0x5681a, 0x5509e
		if index == 0 {
			headingTable, deathRoot = 0x567fa, 0x55062
		}
		if index == 11 {
			headingTable, deathRoot = 0x5683a, 0x550da
		}
		for heading := range 8 {
			frame, err := add(int(binary.BigEndian.Uint32(level[headingTable-levelBase+heading*4:])))
			if err != nil {
				return group, err
			}
			part.HeadingFrames = append(part.HeadingFrames, frame)
		}
		part.DeathAnimation, err = decodeActorAnimation(level, deathRoot-levelBase, add)
		if err != nil {
			return group, err
		}
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

func decodeSecondGuardianDefenseNodes(level []byte) (GuardianGroup, error) {
	group := GuardianGroup{ID: "middle-defense-nodes"}
	if len(level) < 0x5725e-levelBase {
		return group, fmt.Errorf("defense nodes are truncated")
	}
	for i := range 3 {
		// The source allocates node identities in descending order.
		at := 0x57252 - levelBase + (2-i)*4
		column := int(binary.BigEndian.Uint16(level[at:]))
		row := int(binary.BigEndian.Uint16(level[at+2:]))
		part := GuardianComponent{Index: i, Behavior: "defense-node", DamageBehavior: "defense-node-damage", ParentIndex: -1, ResourceTag: 84, InitialX: column * 16, InitialWorldY: row * 16, Health: int(binary.BigEndian.Uint16(level[0x60:])), RenderMode: "terrain-node"}
		flashStart, flashRows := 0x56926, 2
		part.FlashOffsetX, part.FlashOffsetY = -32, -16
		if i == 1 {
			flashStart = 0x5691a
			part.FlashOffsetX = 0
		}
		if i == 2 {
			flashStart, flashRows = 0x56932, 3
			part.FlashOffsetX, part.FlashOffsetY = -16, -32
		}
		flash := readTilePatch(level, flashStart-levelBase, 3, flashRows)
		part.DamageFlash = &flash
		for frame := range 5 {
			part.TileFrames = append(part.TileFrames, readTilePatch(level, 0x56b4e-levelBase+frame*2, 1, 1))
		}
		part.TileFrames = append(part.TileFrames, TilePatch{Columns: 1, Rows: 1, Tiles: []uint16{0x8e45}})
		group.Components = append(group.Components, part)
	}
	for i := range 8 {
		at := 0x56b58 - levelBase + i*18
		cell := int(binary.BigEndian.Uint16(level[at:])) / 2
		gate := GuardianGate{ID: i + 1, Column: cell % 20, Row: cell / 20}
		gate.Frames = []TilePatch{readTilePatch(level, at+2, 2, 2), readTilePatch(level, at+10, 2, 2)}
		group.Gates = append(group.Gates, gate)
	}
	return group, nil
}
