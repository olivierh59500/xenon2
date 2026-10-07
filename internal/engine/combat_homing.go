package engine

// HomingMountState retains its own eighteen-pass launch delay and prevents a
// second volley while any of the original homing missiles remains alive.
type HomingMountState struct{ Cooldown int }

func NewHomingMountState() HomingMountState { return HomingMountState{Cooldown: -1} }

func (s *HomingMountState) Tick(pulse, diving, missilesActive bool) bool {
	if diving {
		return false
	}
	if s.Cooldown >= 0 {
		s.Cooldown--
	}
	if !pulse || s.Cooldown >= 0 || missilesActive {
		return false
	}
	s.Cooldown = 18
	return true
}

// WeaponTarget is an ordered moving-enemy candidate. Bounds are inclusive.
type WeaponTarget struct {
	ID           int
	SlotIdentity int
	ResourceTag  int
	Bounds       CollisionRect
	Active       bool
}

func validHomingTarget(t WeaponTarget) bool {
	return t.ResourceTag >= 200 && visibleWeaponTarget(t)
}

func visibleWeaponTarget(t WeaponTarget) bool {
	return t.Active && t.Bounds.Right >= 0 && t.Bounds.Left < 320 && t.Bounds.Bottom >= 0 && t.Bounds.Top < 192
}

// SelectHomingTarget scans the original moving-list group heads. Acquisition
// can choose a carrier; only reuse of an earlier target requires tag >=200.
func SelectHomingTarget(targets []WeaponTarget, nextRandom func() uint32) (WeaponTarget, bool) {
	if nextRandom == nil {
		return WeaponTarget{}, false
	}
	count := 0
	for _, target := range targets {
		if target.Active && target.ResourceTag != 80 && target.ResourceTag != 84 {
			count++
		}
	}
	if count == 0 {
		count = 1
	}
	choice := int((nextRandom()>>2)&0xffff) % count
	for range 5 {
		for _, target := range targets {
			if target.ResourceTag == 184 || !visibleWeaponTarget(target) {
				continue
			}
			if choice == 0 {
				return target, true
			}
			choice--
		}
	}
	return WeaponTarget{}, false
}

var homingX = [8]int{0, 4, 5, 4, 0, -4, -5, -4}
var homingY = [8]int{-5, -4, 0, 4, 5, 4, 0, -4}

// HomingMissileState holds its launch pause and signed turn timer. A missile
// without an eligible target is removed rather than given a guessed trajectory.
type HomingMissileState struct {
	X, Y                 int
	Direction            uint8
	TurnTimer            int
	TargetID             int
	TargetSlotIdentity   int
	ExpiredWithoutTarget bool
}

func NewHomingMissileState(shipX, shipY int, direction uint8) HomingMissileState {
	return HomingMissileState{X: shipX, Y: shipY, Direction: direction, TurnTimer: 13}
}

// Advance chooses targets in the supplied source order using the shared RNG.
// It returns false if no target exists or its movement leaves the playfield.
func (s *HomingMissileState) Advance(targets []WeaponTarget, nextRandom func() uint32, shipDestroyed bool) bool {
	s.ExpiredWithoutTarget = false
	if shipDestroyed {
		return false
	}
	if s.TurnTimer > 0 {
		s.TurnTimer--
		if s.TurnTimer < 8 {
			return true
		}
	} else {
		var target WeaponTarget
		found := false
		for _, candidate := range targets {
			matches := candidate.ID == s.TargetID
			if s.TargetSlotIdentity != 0 {
				matches = candidate.SlotIdentity == s.TargetSlotIdentity
			}
			if matches && validHomingTarget(candidate) {
				target, found = candidate, true
				break
			}
		}
		if !found {
			target, found = SelectHomingTarget(targets, nextRandom)
			if !found {
				s.ExpiredWithoutTarget = true
				return false
			}
			s.TargetID = target.ID
			s.TargetSlotIdentity = target.SlotIdentity
		}
		if s.TurnTimer != 0 {
			s.TurnTimer++
		}
		if s.TurnTimer == 0 {
			wanted := AimDirection(target.Bounds.Left+2-s.X, target.Bounds.Top+2-s.Y)
			delta := (int(wanted) - int(s.Direction) + 8) & 7
			if delta != 0 {
				if delta >= 4 {
					s.Direction = (s.Direction - 1) & 7
				} else {
					s.Direction = (s.Direction + 1) & 7
				}
				s.TurnTimer = -3
			}
		}
	}
	nextY := s.Y + homingY[s.Direction&7]
	if nextY < 0 || nextY >= 192 {
		return false
	}
	s.Y = nextY
	nextX := s.X + homingX[s.Direction&7]
	if nextX < 0 || nextX >= 320 {
		return false
	}
	s.X = nextX
	return true
}
