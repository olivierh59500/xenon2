package engine

// FirstGuardianState retains the original final guardian's timed extension,
// body movement and articulated path parameters. Artwork is supplied separately.
type FirstGuardianState struct {
	Health                        uint16
	BodyWorldY                    int
	BodyVelocity, BodyTimer       int
	ExtensionPhase, SegmentBudget int
	Heading                       uint8
	AngularVelocity               int32
	AngularAcceleration           int32
	EyeClock                      uint8
	Active, Defeated              bool
	Flash                         bool
	BodyCollision                 CollisionRect
}

func NewFirstGuardianState(health int) FirstGuardianState {
	return FirstGuardianState{Health: uint16(health), BodyWorldY: 16, BodyCollision: CollisionRect{Right: -1, Bottom: -1}}
}

// Advance applies the controller pass after its segments' previous-state
// updates. The returned upper bound preserves the original locked arena.
func (s *FirstGuardianState) Advance(frame uint64, scrollY, maximumScrollY int) (maximum int, activateSegments bool, warningSound bool) {
	maximum = maximumScrollY
	s.Flash = false
	if s.Defeated || scrollY > 448 {
		s.BodyCollision = CollisionRect{Right: -1, Bottom: -1}
		return
	}
	if !s.Active {
		s.Active = true
		activateSegments = true
		s.Heading, s.AngularVelocity, s.AngularAcceleration = 0, 0, 0
	}
	if maximum <= 448 {
		maximum = 448
	}
	top := s.BodyWorldY - scrollY + 4
	s.BodyCollision = CollisionRect{Left: 128, Top: top, Right: 200, Bottom: top + 80}
	if s.ExtensionPhase == 0 {
		if scrollY > 48 {
			s.BodyCollision = CollisionRect{Right: -1, Bottom: -1}
			return
		}
		s.ExtensionPhase, s.BodyVelocity, s.BodyTimer = 1, 0, 170
	}
	if s.ExtensionPhase >= 0 {
		if s.SegmentBudget < 11 {
			warningSound = s.SegmentBudget == 0
			s.SegmentBudget++
		}
	} else if s.SegmentBudget > 0 {
		s.SegmentBudget--
	}
	s.ExtensionPhase++
	if s.ExtensionPhase == 0 {
		s.ExtensionPhase++
	}
	if s.ExtensionPhase >= 100 {
		s.ExtensionPhase = -40
	}
	heading := int(frame & 127)
	if heading&64 != 0 {
		heading = 127 - heading
	}
	s.Heading = uint8(heading + 32)
	angularVelocity := int((frame + 80) & 255)
	if angularVelocity&128 != 0 {
		angularVelocity = int(uint8(^uint8(angularVelocity)))
	}
	s.AngularVelocity = int32(angularVelocity-64) << 10
	angularAcceleration := int((frame + 160) & 511)
	if angularAcceleration&256 != 0 {
		angularAcceleration ^= 511
	}
	s.AngularAcceleration = int32(angularAcceleration-128) << 5
	s.BodyWorldY += s.BodyVelocity
	if s.BodyWorldY < 16 {
		s.BodyWorldY, s.BodyTimer = 16, 0
	} else if s.BodyWorldY > 480 {
		s.BodyWorldY, s.BodyTimer = 480, 0
	}
	s.BodyTimer--
	if s.BodyTimer < 0 {
		switch s.BodyVelocity {
		case -1:
			s.BodyVelocity, s.BodyTimer = 0, 170
		case 1:
			s.BodyVelocity, s.BodyTimer = 0, 60
		default:
			s.BodyTimer = 50
			s.BodyVelocity = -1
			if s.BodyWorldY == 16 {
				s.BodyVelocity = 1
			}
		}
	}
	s.EyeClock += uint8(30 - s.Health)
	return
}

func (s FirstGuardianState) WeakPoint(scrollY int) CollisionRect {
	if !s.Active || s.Defeated {
		return CollisionRect{Right: -1, Bottom: -1}
	}
	top := s.BodyWorldY - scrollY + 68
	return CollisionRect{Left: 152, Top: top, Right: 167, Bottom: top + 14}
}

// Strike is called only after the broad body collision has selected this
// guardian. Hits outside its eye can consume a bullet without damaging it.
func (s *FirstGuardianState) Strike(hit CollisionRect, amount uint16, scrollY int) (damaged, destroyed bool) {
	if !s.WeakPoint(scrollY).Intersects(hit) {
		return false, false
	}
	s.Flash = true
	damage := ApplyEnemyDamage(s.Health, amount)
	s.Health = damage.Health
	s.Defeated = damage.Destroyed
	if s.Defeated {
		s.Active = false
		s.BodyCollision = CollisionRect{Right: -1, Bottom: -1}
	}
	return true, damage.Destroyed
}

// GuardianArcState uses the same fixed-point curve substeps as ordinary paths,
// with the predecessor's complete state copied anew before each segment pass.
type GuardianArcState struct {
	X, Y, AngleFixed, AngularVelocity, AngularAcceleration int32
	Budget                                                 int
}

func (s *GuardianArcState) Advance(sine *[256]int8) {
	for range s.Budget {
		heading := uint8(uint32(s.AngleFixed) >> 16)
		s.X += int32(sine[uint8(heading+64)]) << 10
		s.Y += int32(sine[heading]) << 10
		s.AngleFixed = (s.AngleFixed + s.AngularVelocity) & 0x00ffffff
		s.AngularVelocity += s.AngularAcceleration
	}
}
