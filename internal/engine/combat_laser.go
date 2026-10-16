package engine

// WeaponMountOffset retains the original four additional mounting positions.
var WeaponMountOffset = [4]struct{ X, Y int }{{-26, 16}, {26, 16}, {-44, 24}, {44, 24}}

// LaserMountState delays successive beam creation independently of the shared
// trigger pulse. Materialization freezes this weapon's own countdown.
type LaserMountState struct {
	Cooldown int
}

func NewLaserMountState() LaserMountState {
	return LaserMountState{Cooldown: -1}
}

func (s *LaserMountState) Tick(pulse, materializing bool) bool {
	if materializing {
		return false
	}
	if s.Cooldown >= 0 {
		s.Cooldown--
	}
	if pulse && s.Cooldown < 0 {
		s.Cooldown = 6
		return true
	}
	return false
}

// LaserBeamState follows its mount during extension, then travels upwards in
// independent thirty-two-pixel steps. Its damage area is wider at higher tiers.
type LaserBeamState struct {
	X, Y   int
	Length int
	Tier   int
}

func NewLaserBeamState(tier int) LaserBeamState {
	return LaserBeamState{Length: -32, Tier: tier}
}

// Advance returns whether the beam remains active, its current collision
// rectangle and damage. An empty rectangle means it cannot yet hit a target.
func (s *LaserBeamState) Advance(mountX, mountY int, shipDestroyed bool) (bool, CollisionRect, uint16) {
	if shipDestroyed {
		return false, CollisionRect{Right: -1, Bottom: -1}, 0
	}
	if s.Length != 65 {
		s.X, s.Y = mountX-7, mountY-24
		s.Length += 32
		if s.Length == 96 {
			s.Length = 65
		}
	}
	if s.Length == 65 {
		s.Y -= 32
		if s.Y < 0 {
			return false, CollisionRect{Right: -1, Bottom: -1}, 0
		}
	}
	left := s.X + 4 - s.Tier*2
	top := s.Y - s.Length
	bottom := top + 32
	if bottom < 0 {
		return true, CollisionRect{Right: -1, Bottom: -1}, 0
	}
	if top < 0 {
		top = 0
	}
	return true, CollisionRect{Left: left, Top: top, Right: left + s.Tier*4 + 7, Bottom: bottom}, uint16((s.Tier + 1) * 3)
}
