package engine

// SecondGuardianState retains the source arena controller's wait, movement,
// hatch and fire clocks. Its three terrain defenses have separate damage rules.
type SecondGuardianState struct {
	BodyWorldY, Velocity       int
	WaitTimer, MotionRemaining int
	HatchCountdown, HatchFrame int
	FireAccumulator            uint8
	BodyCollision              CollisionRect
}

type SecondGuardianInput struct {
	Frame                   uint64
	ScrollY, MaximumScrollY int
	PlayerX, PlayerY        int
	ActorCount              int
	FireRate                uint8
	ShotSpeed               int
	// The crowded branch preserves the high byte of the preceding callback's
	// working value, then replaces its low byte with FireRate. Normal decisions
	// consume a new random value instead.
	CrowdedDecisionValue uint16
}

type SecondGuardianEvents struct {
	MaximumScrollY          int
	ShotCount               int
	ShotDirections          [16]uint8
	ShotX, ShotY, ShotSpeed int
	ShotAnimation           string
	SpawnMinion             bool
	MinionX, MinionY        int
	MinionHeading           uint8
	CenterHatchChanged      bool
	CenterHatchFrame        int
	ThrustersChanged        bool
	ThrusterFrame           int
	MovementSound           bool
}

func NewSecondGuardianState() SecondGuardianState {
	return SecondGuardianState{BodyWorldY: 96, WaitTimer: -1, BodyCollision: CollisionRect{Right: -1, Bottom: -1}}
}

// Advance reproduces the level's controller pass. Starting the first traversal
// returns immediately; collision bounds are refreshed before body travel.
func (s *SecondGuardianState) Advance(input SecondGuardianInput, random *RandomState) SecondGuardianEvents {
	event := SecondGuardianEvents{MaximumScrollY: input.MaximumScrollY}
	if event.MaximumScrollY <= 288 {
		event.MaximumScrollY = 288
	}
	if s.MotionRemaining == 0 {
		if s.WaitTimer < 0 {
			s.BodyCollision = CollisionRect{Right: -1, Bottom: -1}
			if input.PlayerY+input.ScrollY <= 176 && input.PlayerX >= 0 && input.PlayerX <= 319 {
				s.MotionRemaining, s.Velocity, s.WaitTimer = 1000, 3, 340
			}
			return event
		}
		s.WaitTimer--
		if s.WaitTimer < 0 {
			s.HatchCountdown = 0
			s.MotionRemaining, s.Velocity = 1000, -3
		}
	}
	work := uint32(input.CrowdedDecisionValue&0xff00) | uint32(input.FireRate)
	sum := int(s.FireAccumulator) + int(input.FireRate)
	s.FireAccumulator = uint8(sum)
	if sum >= 256 && random != nil {
		work = random.Next() & 63
		s.FireAccumulator = uint8(work)
		work = random.Next() & 3
		event.ShotX, event.ShotY, event.ShotSpeed = 160, s.BodyWorldY-input.ScrollY+112, input.ShotSpeed
		event.ShotAnimation = "guardian-single-shot"
		event.ShotCount = 1
		event.ShotDirections[0] = 4
		if work == 0 {
			event.ShotAnimation = "guardian-radial-shot"
			event.ShotCount = 16
			for i := range 16 {
				event.ShotDirections[i] = uint8(15 - i)
			}
		}
		work = 160 // Both source shot constructors retain their spawn X in D0.
	}
	top := s.BodyWorldY - input.ScrollY + 112
	s.BodyCollision = CollisionRect{Left: 152, Top: top, Right: 167, Bottom: top + 10}
	if input.ScrollY >= 352 {
		return event
	}
	if s.HatchCountdown != 0 {
		s.HatchFrame = (s.HatchFrame + 1) & 3
		event.CenterHatchChanged, event.CenterHatchFrame = true, s.HatchFrame
		s.HatchCountdown--
		if s.HatchCountdown == 0 {
			event.SpawnMinion = true
			event.MinionX, event.MinionY = 160, s.BodyWorldY-input.ScrollY+128
			if random != nil {
				event.MinionHeading = uint8(random.Next() & 7)
			}
		}
		return event
	}
	if s.MotionRemaining != 0 {
		event.MovementSound = true
		s.MotionRemaining--
		s.BodyWorldY += s.Velocity
		if s.BodyWorldY < 96 {
			s.BodyWorldY, s.MotionRemaining = 96, 0
		}
		if s.BodyWorldY > 192 {
			s.BodyWorldY, s.MotionRemaining = 192, 0
		}
		event.ThrustersChanged, event.ThrusterFrame = true, int(input.Frame&1)
		return event
	}
	if input.ActorCount < 16 && random != nil {
		work = random.Next()
		if work&8192 == 0 {
			s.HatchCountdown = int(work&7) + 2
			return event
		}
	}
	s.Velocity = 3
	if work&16384 != 0 {
		s.Velocity = -3
	}
	s.MotionRemaining = int(work&7) + 8
	return event
}
