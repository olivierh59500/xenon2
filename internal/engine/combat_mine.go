package engine

// MineState retains the original interactive planting cursor and the release
// countdown of a dropped mine. Negative modes belong to independent mines.
type MineState struct {
	X, Y int
	Mode int
	Tier int
}

// MineContext supplies the shared planting count and the last planting point.
type MineContext struct {
	Count        int
	LastX, LastY int
}

// Advance returns a new dropped mine, an explosion area, and whether this mine
// remains alive. Spawned mines begin moving on the following gameplay pass.
func (s *MineState) Advance(shipX, shipY, trailX, trailY int, input MotionInput, held, materializing, shipDestroyed bool, context *MineContext) (*MineState, CollisionRect, uint16, bool) {
	empty := CollisionRect{Right: -1, Bottom: -1}
	if s.Mode < 0 {
		if shipDestroyed {
			return nil, empty, 0, false
		}
		if held {
			return nil, empty, 0, true
		}
		s.Mode++
		if s.Mode != 0 {
			return nil, empty, 0, true
		}
		if context != nil {
			context.Count--
		}
		radius, rightExtension := 16, 32
		if s.Tier != 0 {
			radius, rightExtension = 24, 47
		}
		left, top := s.X-radius, s.Y-radius
		return nil, CollisionRect{Left: left, Top: top, Right: left + rightExtension, Bottom: top + rightExtension}, uint16(s.Tier*2 + 4), false
	}
	if materializing {
		return nil, empty, 0, true
	}
	switch s.Mode {
	case 0:
		s.X, s.Y = shipX, shipY+25
		if held {
			s.Mode = 1
		}
	case 1:
		if !held {
			s.Mode = 3
		} else if s.Y > 16 {
			s.Y -= 12
		} else {
			s.Mode = 2
		}
	case 2:
		if !held {
			s.Mode = 3
			break
		}
		if input.Left {
			s.X = clampWeaponCoordinate(s.X-8, 319)
		}
		if input.Right {
			s.X = clampWeaponCoordinate(s.X+8, 319)
		}
		if input.Up {
			s.Y = clampWeaponCoordinate(s.Y-8, 191)
		}
		if input.Down {
			s.Y = clampWeaponCoordinate(s.Y+8, 191)
		}
		if context != nil && context.Count != 10 && (absWeapon(s.X-context.LastX) >= 32 || absWeapon(s.Y-context.LastY) >= 32) {
			context.LastX, context.LastY = s.X, s.Y
			context.Count++
			mine := &MineState{X: s.X, Y: s.Y, Mode: -context.Count * 4, Tier: s.Tier}
			return mine, empty, 0, true
		}
	default:
		s.X += clampWeaponStep(shipX - s.X)
		s.Y += clampWeaponStep(shipY + 25 - s.Y)
		if s.X == shipX && s.Y == shipY+25 {
			s.Mode = 0
		}
	}
	return nil, empty, 0, true
}

func absWeapon(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
