package visualassets

import (
	"encoding/binary"
	"fmt"
	"sort"
)

type PathCommand struct {
	Kind                string `json:"kind"`
	X                   int    `json:"x,omitempty"`
	Y                   int    `json:"y,omitempty"`
	Heading             int    `json:"heading,omitempty"`
	AngularVelocity     int    `json:"angular_velocity,omitempty"`
	AngularAcceleration int    `json:"angular_acceleration,omitempty"`
	Duration            int    `json:"duration,omitempty"`
	Target              int    `json:"target,omitempty"`
	Targets             []int  `json:"targets,omitempty"`
}

type Path struct {
	ID       int           `json:"id"`
	Commands []PathCommand `json:"commands"`
}

type Paths struct {
	Paths     []Path    `json:"paths"`
	SineTable [256]int8 `json:"sine_table"`
}

// DecodePaths replaces native byte offsets and numeric command words with
// named commands and checked command indices. The sine table is ordinary
// signed numeric motion data, not a generated approximation.
func DecodePaths(level, common []byte) (*Paths, error) {
	if len(level) < 0x3c || len(common) < 0x98f6+256 {
		return nil, fmt.Errorf("path tables are truncated")
	}
	table, err := offset(level, 0x30, 4)
	if err != nil {
		return nil, err
	}
	first, err := offset(level, 0x34, 2)
	if err != nil {
		return nil, err
	}
	end, err := offset(level, 0x38, 2)
	if err != nil {
		return nil, err
	}
	if first <= table || (first-table)%4 != 0 || end <= first {
		return nil, fmt.Errorf("invalid path table boundaries")
	}
	result := &Paths{}
	for i := range result.SineTable {
		result.SineTable[i] = int8(common[0x98f6+i])
	}
	roots := make([]int, 0, (first-table)/4)
	unique := map[int]bool{}
	for cursor := table; cursor < first; cursor += 4 {
		address := uint64(binary.BigEndian.Uint32(level[cursor:]))
		if address < levelBase+uint64(first) || address >= levelBase+uint64(end) {
			return nil, fmt.Errorf("path root leaves path data")
		}
		root := int(address - levelBase)
		if root%2 != 0 {
			return nil, fmt.Errorf("unaligned path root")
		}
		roots = append(roots, root)
		unique[root] = true
	}
	sorted := make([]int, 0, len(unique))
	for root := range unique {
		sorted = append(sorted, root)
	}
	sort.Ints(sorted)
	decoded := make(map[int][]PathCommand, len(sorted))
	for i, root := range sorted {
		limit := end
		if i+1 < len(sorted) {
			limit = sorted[i+1]
		}
		commands, err := decodePath(level[root:limit])
		if err != nil {
			return nil, fmt.Errorf("path %d: %w", i+1, err)
		}
		decoded[root] = commands
	}
	for i, root := range roots {
		result.Paths = append(result.Paths, Path{ID: i + 1, Commands: decoded[root]})
	}
	return result, nil
}

func decodePath(data []byte) ([]PathCommand, error) {
	commands := make([]PathCommand, 0)
	positions := map[int]int{}
	cursor := 0
	read := func(position int) int { return int(int16(binary.BigEndian.Uint16(data[position:]))) }
	for cursor < len(data) {
		if cursor+2 > len(data) {
			return nil, fmt.Errorf("truncated path command")
		}
		positions[cursor] = len(commands)
		opcode := read(cursor)
		length := 0
		command := PathCommand{}
		switch opcode {
		case 0:
			command.Kind = "end"
			length = 2
		case 2:
			command.Kind = "curve"
			length = 10
		case 4:
			command.Kind = "pause"
			length = 4
		case 6:
			command.Kind = "random-branch"
			length = 18
		case 8:
			command.Kind = "jump"
			length = 4
		case 10:
			command.Kind = "origin"
			length = 6
		default:
			return nil, fmt.Errorf("unknown path command %d", opcode)
		}
		if cursor+length > len(data) {
			return nil, fmt.Errorf("path command %s exceeds its data", command.Kind)
		}
		switch command.Kind {
		case "curve":
			command.Heading, command.AngularVelocity, command.AngularAcceleration, command.Duration = read(cursor+2), read(cursor+4), read(cursor+6), read(cursor+8)
		case "pause":
			command.Duration = read(cursor + 2)
		case "origin":
			command.X, command.Y = read(cursor+2), read(cursor+4)
		case "jump":
			command.Target = read(cursor + 2)
		case "random-branch":
			for i := range 8 {
				command.Targets = append(command.Targets, read(cursor+2+i*2))
			}
		}
		commands = append(commands, command)
		cursor += length
	}
	for i := range commands {
		command := &commands[i]
		if command.Kind == "jump" {
			target, ok := positions[command.Target]
			if !ok {
				return nil, fmt.Errorf("jump target is not a command boundary")
			}
			command.Target = target
		}
		if command.Kind == "random-branch" {
			available := false
			for i, offset := range command.Targets {
				if offset < 0 {
					command.Targets[i] = -1
					continue
				}
				target, ok := positions[offset]
				if !ok {
					return nil, fmt.Errorf("random target is not a command boundary")
				}
				command.Targets[i] = target
				available = true
			}
			if !available {
				return nil, fmt.Errorf("random branch has no available target")
			}
		}
	}
	return commands, nil
}
