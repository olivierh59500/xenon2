package engine

import "fmt"

// FlameShot retains the half-pixel launch position and signed random drift.
// Only the first particle of each pair damages enemies, using its actor box.
type FlameShot struct {
	X          int32
	Y          int
	VelocityX  int32
	Tier       int
	Damaging   bool
	SpriteName string
}

// AppendFlamerShots emits two particles on each held-trigger pass. Their random
// vertical jitter is chosen before their signed horizontal drift, in order.
func AppendFlamerShots(dst []FlameShot, tier, shipX, shipY int, nextRandom func() uint32) ([]FlameShot, error) {
	return AppendFlamerShotsWithResidue(dst, tier, shipX, shipY, nextRandom, nil)
}

// AppendFlamerShotsWithResidue preserves the previous slot's low drift word
// when the world reuses an actor. A fresh independent slot supplies zero.
func AppendFlamerShotsWithResidue(dst []FlameShot, tier, shipX, shipY int, nextRandom func() uint32, nextResidue func() uint16) ([]FlameShot, error) {
	if tier < 0 || tier > 2 || nextRandom == nil {
		return dst, fmt.Errorf("flamer needs a valid tier and world random stream")
	}
	for i := 0; i < 2; i++ {
		y := shipY - 16 - i*6 + int(nextRandom()&3)
		velocity := int32(int8(uint8(nextRandom()))) << 10
		if nextResidue != nil {
			velocity += int32(nextResidue() >> 6)
		}
		dst = append(dst, FlameShot{X: int32(shipX)<<16 | 0x8000, Y: y, VelocityX: velocity, Tier: tier, Damaging: i == 0, SpriteName: "flamer-particle"})
	}
	return dst, nil
}

func (s *FlameShot) Advance(shipDestroyed bool) bool {
	if shipDestroyed {
		return false
	}
	s.X += s.VelocityX
	x := s.X >> 16
	if x < 0 || x >= 320 {
		return false
	}
	s.Y -= 12
	return s.Y >= 0
}

func (s FlameShot) Damage() uint16 {
	if !s.Damaging {
		return 0
	}
	return uint16(s.Tier + 1)
}
