package visualassets

import (
	"encoding/binary"
	"fmt"
)

// FifthFormationArt describes the ten-member path formations encountered in
// the last level. The members share a path but retain independent delays.
type FifthFormationArt struct {
	Paths                                                    []Path           `json:"paths"`
	HeadingAnimations                                        []ActorAnimation `json:"heading_animations"`
	InitialSprite                                            string           `json:"initial_sprite"`
	ShotSprite                                               string           `json:"shot_sprite"`
	Health, MotionBudget, MemberSpacing, FireRate, ShotSpeed int
}

func decodeFifthFormationArt(level []byte, add func(int) (string, error)) (*FifthFormationArt, error) {
	if len(level) < 0x72ee0-levelBase {
		return nil, fmt.Errorf("fifth formation resources are truncated")
	}
	word := func(at int) int { return int(binary.BigEndian.Uint16(level[at:])) }
	a := &FifthFormationArt{Health: word(0x42), MemberSpacing: word(0x44), MotionBudget: word(0x46), FireRate: int(level[0x79]), ShotSpeed: word(0x7a)}
	for i, end := range []int{0x72bc6, 0x72d88, 0x72ec8} {
		root := int(binary.BigEndian.Uint32(level[0x551dc-levelBase+i*4:]))
		commands, err := decodePath(level[root-levelBase : end-levelBase])
		if err != nil {
			return nil, err
		}
		a.Paths = append(a.Paths, Path{ID: i, Commands: commands})
	}
	for i := 0; i < 8; i++ {
		root := int(binary.BigEndian.Uint32(level[0x552ba-levelBase+i*4:]))
		animation, err := decodeActorAnimation(level, root-levelBase, add)
		if err != nil {
			return nil, err
		}
		a.HeadingAnimations = append(a.HeadingAnimations, animation)
	}
	var err error
	a.InitialSprite, err = add(0x5c1ea)
	if err != nil {
		return nil, err
	}
	choice := word(0x7c)
	root := int(binary.BigEndian.Uint32(level[0x72ec8-levelBase+choice*4:])) - levelBase
	if root < 0 || root+4 > len(level) {
		return nil, fmt.Errorf("fifth formation shot choice leaves its bank")
	}
	a.ShotSprite, err = add(int(binary.BigEndian.Uint32(level[root:])))
	return a, err
}
