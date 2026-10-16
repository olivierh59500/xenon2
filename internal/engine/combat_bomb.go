package engine

// BombMountState waits for a trigger pulse and retains the original cooldown.
type BombMountState struct {
	Cooldown int
}

func NewBombMountState() BombMountState { return BombMountState{Cooldown: -1} }

func (s *BombMountState) Tick(pulse, diving, bombActive bool) bool {
	if diving {
		return false
	}
	if s.Cooldown >= 0 {
		s.Cooldown--
	}
	if !pulse || s.Cooldown >= 0 || bombActive {
		return false
	}
	s.Cooldown = 15
	return true
}

// BombState rises for a position-dependent countdown, pauses for three passes,
// then damages its inclusive 64-pixel explosion area.
type BombState struct {
	X, Y  int
	Timer int
}

func NewBombState(shipX, shipY int) BombState {
	y := shipY - 12
	return BombState{X: shipX, Y: y, Timer: -((y >> 4) + 4)}
}

func (s *BombState) Advance(shipDestroyed bool) (bool, CollisionRect, uint16) {
	empty := CollisionRect{Right: -1, Bottom: -1}
	if shipDestroyed {
		return false, empty, 0
	}
	if s.Timer < 0 {
		s.Timer++
		s.Y -= 6
		return true, empty, 0
	}
	if s.Timer == 0 {
		s.Timer = 3
		return true, empty, 0
	}
	s.Timer--
	if s.Timer != 0 {
		return true, empty, 0
	}
	return false, CollisionRect{Left: s.X - 32, Top: s.Y - 32, Right: s.X + 31, Bottom: s.Y + 31}, 8
}
