package visualassets

import (
	"encoding/binary"
	"fmt"
)

type FourthStageVariant struct {
	ID, ResourceTag, X int
	Behavior           string         `json:"behavior"`
	Animation          ActorAnimation `json:"animation"`
	PodTag, PodX       int
	PodAnimation       ActorAnimation `json:"pod_animation"`
}

type FourthStageArt struct {
	Variants                                                                       []FourthStageVariant `json:"variants"`
	Health, PodChildHealth, ChildBudget                                            int
	HatchSpeed, GunSpeed, HatchFireRate, GunFireRate, SecondaryFireRate, ShotSpeed int
	GunShotSpeed                                                                   int
	ShotSprite                                                                     string           `json:"shot_sprite"`
	PodChildAnimation                                                              []ActorAnimation `json:"pod_child_animation"`
	PodChildPaths                                                                  []Path           `json:"pod_child_paths"`
	PodTransformFrames                                                             [2]int           `json:"pod_transform_frames"`
	ChildInitialXOffset                                                            [2]int           `json:"child_initial_x_offset"`
	ChildInitialHeading                                                            [2]int           `json:"child_initial_heading"`
}

func decodeFourthStageArt(level []byte, add func(int) (string, error)) (*FourthStageArt, error) {
	if len(level) < 0x57558-levelBase {
		return nil, fmt.Errorf("fourth stage spawner data is truncated")
	}
	word := func(at int) int { return int(binary.BigEndian.Uint16(level[at:])) }
	a := &FourthStageArt{Health: word(0x42), PodChildHealth: word(0x48), ChildBudget: word(0x4a), HatchSpeed: word(0x44), GunSpeed: word(0x4c), HatchFireRate: int(level[0x47]), GunFireRate: int(level[0x4f]), SecondaryFireRate: int(level[0x67]), ShotSpeed: word(0x86), GunShotSpeed: word(0x50), ChildInitialXOffset: [2]int{10, -10}, ChildInitialHeading: [2]int{0, 128}}
	for i := range 4 {
		at := 0x5515a - levelBase + i*12
		v := FourthStageVariant{ID: i, ResourceTag: word(at), X: word(at + 2), Behavior: "falling-hatch"}
		if i&1 != 0 {
			v.Behavior = "falling-gun"
		}
		root := int(binary.BigEndian.Uint32(level[at+8:]))
		clip, err := decodeActorAnimation(level, root-levelBase, add)
		if err != nil {
			return nil, err
		}
		v.Animation = clip
		if i&1 == 0 {
			podAt := 0x5537e - levelBase
			if i == 2 {
				podAt += 8
			}
			v.PodTag, v.PodX = word(podAt), word(podAt+2)
			podRoot := int(binary.BigEndian.Uint32(level[podAt+4:]))
			v.PodAnimation, err = decodeActorAnimation(level, podRoot-levelBase, add)
			if err != nil {
				return nil, err
			}
			target := 0x5f4aa
			if i == 2 {
				target = 0x5edca
			}
			name, err := add(target)
			if err != nil {
				return nil, err
			}
			for frame, image := range v.PodAnimation.Frames {
				if image.Sprite == name {
					a.PodTransformFrames[i/2] = frame
					break
				}
			}
		}
		a.Variants = append(a.Variants, v)
	}
	for heading := range 8 {
		root := int(binary.BigEndian.Uint32(level[0x55574-levelBase+heading*4:]))
		clip, err := decodeActorAnimation(level, root-levelBase, add)
		if err != nil {
			return nil, err
		}
		a.PodChildAnimation = append(a.PodChildAnimation, clip)
	}
	roots := []int{0x5738e, 0x57434, 0x573e6, 0x57496}
	ends := []int{0x573e0, 0x57490, 0x5742e, 0x574f2}
	for i, root := range roots {
		end := ends[i]
		commands, err := decodePath(level[root-levelBase : end-levelBase])
		if err != nil {
			return nil, err
		}
		a.PodChildPaths = append(a.PodChildPaths, Path{ID: i, Commands: commands})
	}
	choice := word(0x88)
	table := 0x77b70 - levelBase + choice*4
	if table < 0 || table+4 > len(level) {
		return nil, fmt.Errorf("fourth stage shot choice is outside its table")
	}
	root := int(binary.BigEndian.Uint32(level[table:])) - levelBase
	if root < 0 || root+4 > len(level) {
		return nil, fmt.Errorf("fourth stage shot frame is outside its table")
	}
	var err error
	a.ShotSprite, err = add(int(binary.BigEndian.Uint32(level[root:])))
	return a, err
}
