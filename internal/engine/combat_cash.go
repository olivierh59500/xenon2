package engine

// CashMotion reproduces the shared movement of cash and collectible rewards.
// Positive mode seeks the centre; negative mode spirals; zero mode falls away.
type CashMotion struct {
	X, Y int
	Mode int
	// The original stores the entire word even when only its low three bits
	// select movement. Slot reuse can expose the retained upper bits.
	Direction uint16
}

var cashX = [8]int{0, 2, 2, 2, 0, -2, -2, -2}
var cashY = [8]int{-2, -2, 0, 2, 2, 2, 0, -2}

// Advance returns false when an uncollected reward reaches Y = 200. Collection
// uses the current sprite's ordinary actor rectangle and is handled by the world.
func (s *CashMotion) Advance() bool {
	if s.Mode == 0 {
		s.Y += 8
		return s.Y < 200
	}
	if s.Mode < 0 {
		s.Mode++
		if s.Mode&3 == 0 {
			s.Direction = (s.Direction + 1) & 7
		}
	} else {
		s.Mode &= 7
		if s.Mode == 0 {
			s.Direction = uint16(AimDirection(160-s.X, 100-s.Y))
		}
		s.Mode++
		dx, dy := 160-s.X, 100-s.Y
		if dx*dx+dy*dy < 400 {
			s.Mode = -34
		}
	}
	direction := s.Direction & 7
	s.X += cashX[direction]
	s.Y += cashY[direction]
	return true
}

// CashValue returns the monetary value of a normal or heavy wave reward.
func CashValue(heavy bool) int {
	if heavy {
		return 100
	}
	return 50
}
