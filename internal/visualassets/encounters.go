package visualassets

import (
	"encoding/binary"
	"fmt"
)

type Wave struct {
	TriggerY     int `json:"trigger_y"`
	EnemyKind    int `json:"enemy_kind"`
	Count        int `json:"count"`
	PathID       int `json:"path_id"`
	Spacing      int `json:"spacing"`
	FireRate     int `json:"fire_rate"`
	MotionBudget int `json:"motion_budget"`
}

// FixedEncounter retains the placement and state parameters from the fixed
// scenery stream. EnemyKind is a level-specific resource identifier, not code.
type FixedEncounter struct {
	TriggerY  int `json:"trigger_y"`
	EnemyKind int `json:"enemy_kind"`
	State1    int `json:"state_1"`
	State2    int `json:"state_2"`
	X         int `json:"x"`
	Y         int `json:"y"`
	Variant   int `json:"variant"`
}

type Encounters struct {
	Moving []Wave           `json:"moving"`
	Fixed  []FixedEncounter `json:"fixed"`
}

// DecodeEncounters preserves source order and checks each fourteen-byte record.
// It does not invent behavior for the level-specific enemy kinds.
func DecodeEncounters(level []byte, paths *Paths) (*Encounters, error) {
	result := &Encounters{}
	readStream := func(field int, visit func([7]int) error) error {
		start, err := offset(level, field, 2)
		if err != nil {
			return err
		}
		for records := 0; records < 4096; records++ {
			if start+2 > len(level) {
				return fmt.Errorf("encounter stream lacks its terminator")
			}
			if int16(binary.BigEndian.Uint16(level[start:])) < 0 {
				return nil
			}
			if start+14 > len(level) {
				return fmt.Errorf("encounter record is truncated")
			}
			var values [7]int
			for i := range values {
				values[i] = int(int16(binary.BigEndian.Uint16(level[start+i*2:])))
			}
			if err := visit(values); err != nil {
				return err
			}
			start += 14
		}
		return fmt.Errorf("encounter stream exceeds its record limit")
	}
	if err := readStream(0x38, func(v [7]int) error {
		if v[2] < 0 || v[3] < 1 || v[3] > len(paths.Paths) || v[6] < 0 {
			return fmt.Errorf("moving encounter has invalid count, path or motion budget")
		}
		result.Moving = append(result.Moving, Wave{TriggerY: v[0], EnemyKind: v[1], Count: v[2], PathID: v[3], Spacing: v[4], FireRate: v[5], MotionBudget: v[6]})
		return nil
	}); err != nil {
		return nil, err
	}
	if err := readStream(0x14, func(v [7]int) error {
		result.Fixed = append(result.Fixed, FixedEncounter{TriggerY: v[0], EnemyKind: v[1], State1: v[2], State2: v[3], X: v[4], Y: v[5], Variant: v[6]})
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}
