// Package presentation implements the original menu's independent visual state.
package presentation

import "xenon2/internal/engine"

type Star struct {
	X, Y                    int16
	Depth                   uint16
	ScreenX, ScreenY, Color int
}

type Starfield struct {
	Stars  [48]Star
	Random *engine.RandomState
	Colors [8]uint8
}

func NewStarfield(random *engine.RandomState, colors []uint8) *Starfield {
	s := &Starfield{Random: random}
	copy(s.Colors[:], colors)
	for i := range s.Stars {
		s.Stars[i] = Star{X: int16(random.Next()), Y: int16(random.Next()), Depth: uint16(random.Next()) & 8191}
	}
	return s
}

// Advance preserves the original signed-word projection and depth decrement.
// Occupancy is optional; callers can record a covered star for next-pass reset.
func (s *Starfield) Advance() {
	var occupied [320 * 200]bool
	for i := range s.Stars {
		star := &s.Stars[i]
		depth := int16((star.Depth & 32767) - 512)
		for attempts := 0; attempts < 1000; attempts++ {
			if depth <= 512 {
				star.Y = int16(s.Random.Next())
				star.X = int16(s.Random.Next())
				depth = 8191
			}
			x, y := int32(star.X)*16/int32(depth)+160, int32(star.Y)*16/int32(depth)+100
			if x < 0 || x >= 320 || y < 0 || y >= 200 {
				depth = 0
				continue
			}
			star.Depth = uint16(depth)
			star.ScreenX, star.ScreenY = int(x), int(y)
			rotated := uint16(depth)<<6 | uint16(depth)>>10
			star.Color = int(s.Colors[rotated&7])
			pixel := star.ScreenY*320 + star.ScreenX
			if occupied[pixel] {
				star.Depth |= 32768
			} else {
				occupied[pixel] = true
			}
			break
		}
	}
}

func (s *Starfield) Covered(index int) { s.Stars[index].Depth |= 32768 }
