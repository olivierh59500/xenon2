package engine

// ElectroBallState follows the oldest ship trail while idle, magnifies ship
// motion while held, then returns to that trail in eight-pixel steps.
type ElectroBallState struct {
	X, Y int
	Mode int
}

// Advance returns whether the ball should query its actor rectangle for a
// four-point hit on this pass. Materialization freezes its interactive state.
func (s *ElectroBallState) Advance(shipX, shipY, previousX, previousY, trailX, trailY int, held, materializing bool) bool {
	if materializing {
		return false
	}
	if s.Mode == 0 {
		if !held {
			s.X, s.Y = trailX, trailY+25
			return false
		}
		s.Mode = 1
		return true
	}
	if s.Mode > 0 {
		if !held {
			s.Mode = -1
			return false
		}
		s.X = clampWeaponCoordinate(s.X+4*(shipX-previousX), 319)
		s.Y = clampWeaponCoordinate(s.Y+4*(shipY-previousY), 191)
		return true
	}
	s.X += clampWeaponStep(trailX - s.X)
	s.Y += clampWeaponStep(trailY + 25 - s.Y)
	if s.X == trailX && s.Y == trailY+25 {
		s.Mode = 0
	}
	return false
}

func clampWeaponCoordinate(value, maximum int) int {
	if value < 0 {
		return 0
	}
	if value > maximum {
		return maximum
	}
	return value
}

func clampWeaponStep(value int) int {
	if value < -8 {
		return -8
	}
	if value > 8 {
		return 8
	}
	return value
}

const ElectroBallDamage = 4
