package visualassets

import (
	"encoding/binary"
	"fmt"
)

// FixedProjectileArtwork stores specialized enemy-shot artwork and gameplay
// data independently of the level program that originally selected it.
type FixedProjectileArtwork struct {
	Kind              string           `json:"kind"`
	ActorList         string           `json:"actor_list"`
	ResourceTag       int              `json:"resource_tag"`
	Health            int              `json:"health,omitempty"`
	Score             int              `json:"score,omitempty"`
	Lifetime          int              `json:"lifetime,omitempty"`
	MotionBudget      int              `json:"motion_budget,omitempty"`
	ContactDamage     int              `json:"contact_damage"`
	TurningSprites    [2]string        `json:"turning_sprites,omitempty"`
	HeadingAnimations []ActorAnimation `json:"heading_animations,omitempty"`
}

func decodeFixedProjectileArtwork(number int, level []byte, add func(int) (string, error)) (*FixedProjectileArtwork, error) {
	if (number == 3 || number == 5) && len(level) < 0x56 {
		return nil, fmt.Errorf("fixed projectile level header is truncated")
	}
	switch number {
	case 3:
		a := &FixedProjectileArtwork{Kind: "turning-projectile", ActorList: "projectile", ResourceTag: 20, MotionBudget: int(binary.BigEndian.Uint16(level[0x54:])), ContactDamage: 4}
		var err error
		a.TurningSprites[0], err = add(0x5b330)
		if err != nil {
			return nil, err
		}
		a.TurningSprites[1], err = add(0x5b386)
		return a, err
	case 5:
		if len(level) < 0x557de-levelBase+32 {
			return nil, fmt.Errorf("fixed projectile heading table is truncated")
		}
		a := &FixedProjectileArtwork{Kind: "animated-aiming-projectile", ActorList: "moving", ResourceTag: 228, Health: int(binary.BigEndian.Uint16(level[0x4c:])), Score: 100, Lifetime: int(binary.BigEndian.Uint16(level[0x4e:])), ContactDamage: 4}
		for heading := range 8 {
			root := int(binary.BigEndian.Uint32(level[0x557de-levelBase+heading*4:])) - levelBase
			animation, err := decodeActorAnimation(level, root, add)
			if err != nil {
				return nil, err
			}
			a.HeadingAnimations = append(a.HeadingAnimations, animation)
		}
		return a, nil
	}
	return nil, nil
}
