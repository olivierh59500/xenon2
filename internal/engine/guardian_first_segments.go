package engine

// FirstGuardianSegments retains the eight articulated pieces independently of
// their artwork. The final piece selects heading-specific images and fires.
type FirstGuardianSegments struct {
	Pieces    [8]GuardianArcState
	TailFrame uint8
	TailFire  EnemyFireState
	Alive     bool
}

func NewFirstGuardianSegments() FirstGuardianSegments {
	s := FirstGuardianSegments{Alive: true, TailFire: EnemyFireState{Rate: 115}}
	for i := range s.Pieces {
		s.Pieces[i].Y = -100 << 16
	}
	return s
}

// Advance runs before the guardian body controller, matching the native list
// order. Each piece copies its predecessor's newly advanced curve state.
func (s *FirstGuardianSegments) Advance(controller FirstGuardianState, scrollY, playerX, playerY int, sine *[256]int8, nextRandom func() uint32) (shot EnemyShot, fired bool, err error) {
	if !controller.Active || controller.Defeated {
		s.Alive = false
		return
	}
	previous := GuardianArcState{X: 160 << 16, Y: int32(controller.BodyWorldY+96-scrollY) << 16,
		AngleFixed: int32(controller.Heading) << 16, AngularVelocity: controller.AngularVelocity,
		AngularAcceleration: controller.AngularAcceleration, Budget: controller.SegmentBudget}
	for i := range s.Pieces {
		s.Pieces[i] = previous
		s.Pieces[i].Advance(sine)
		previous = s.Pieces[i]
	}
	tail := &s.Pieces[7]
	if tail.Budget == 0 {
		return
	}
	dx, dy := playerX-int(tail.X>>16), playerY-int(tail.Y>>16)
	s.TailFrame = AimDirection(dx, dy)
	// The heading selector also writes the low fractional word of the tail's
	// swapped native angle. Preserve that fraction in independent fixed point.
	tail.AngleFixed = tail.AngleFixed&^0xffff | int32(s.TailFrame)
	if tail.Budget >= 11 {
		shot, fired, err = s.TailFire.Tick(nextRandom, dx, dy)
		shot.Speed = 4
	}
	return
}
