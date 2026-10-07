package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

type FourthGuardianPart struct {
	Arc                      GuardianArcState
	Health                   uint16
	Counter                  int16
	Direction                int16
	FireAccumulator          uint8
	Animation                AnimationState
	Pose                     int
	Collision                CollisionRect
	Visible, Disabled, Flash bool
}

type FourthGuardianShot struct {
	X, Y, Speed int
	Direction   uint8
	Animation   string
}

type FourthGuardianEvents struct {
	MaximumScrollY                      int
	ShotCount                           int
	Shots                               [8]FourthGuardianShot
	WarningSound                        bool
	Defeated, ClearMoving, AdvanceLevel bool
	ReleaseMoving                       bool
	MinimumScrollY, ForcedScrollY       int
	ClearTerrainRow, ClearTerrainRows   int
	CashPairs, ExplosionCount, Score    int
	ExplosionRectangle                  CollisionRect
	Explosion                           bool
	ExplosionX, ExplosionY              int
	RemoveParts                         [20]bool
}

type FourthGuardianInput struct {
	Frame                                uint64
	ScrollY, ScrollDelta, MaximumScrollY int
	PlayerX, PlayerY                     int
}

// FourthFinalGuardian retains the nineteen original parts. The body advances
// before its two tiled eyes and the sixteen independently drawn arm pieces.
type FourthFinalGuardian struct {
	Parts                          [19]FourthGuardianPart
	EyesRemaining, BodyScrollDelta int
	Defeated                       bool
}

func NewFourthFinalGuardian(art *visualassets.GuardianGroup, scrollY int, residues []ActorResidue) (FourthFinalGuardian, error) {
	var state FourthFinalGuardian
	if art == nil || art.ID != "final-guardian" || len(art.Components) != 19 {
		return state, fmt.Errorf("invalid fourth final guardian artwork")
	}
	state.EyesRemaining = 2
	for i, component := range art.Components {
		part := &state.Parts[i]
		part.Arc = GuardianArcState{X: int32(component.InitialX) << 16, Y: int32(component.InitialWorldY-scrollY) << 16, Budget: 3}
		if i < len(residues) {
			part.Arc.X |= int32(residues[i].XFraction)
			part.Arc.Y |= int32(residues[i].YFraction)
			part.FireAccumulator = residues[i].FireAccumulator()
		}
		part.Health, part.Visible, part.Collision = uint16(component.Health), true, CollisionRect{Left: 1000, Right: 1000}
	}
	return state, nil
}

func fourthTailCurve(frame uint64) (heading uint8, velocity, acceleration int32) {
	h := int(frame * 4 & 127)
	if h&64 != 0 {
		h = 127 - h
	}
	heading = uint8(h + 160)
	v := int((frame*2 + 80) & 255)
	if v&128 != 0 {
		v = 255 - v
	}
	velocity = int32(v-64) << 8
	a := int((frame + 160) & 511)
	if a&256 != 0 {
		a ^= 511
	}
	acceleration = int32(int16((a - 128) << 2))
	return
}

func fourthFire(accumulator *uint8, rate int, nextRandom func() uint32) bool {
	sum := int(*accumulator) + rate
	*accumulator = uint8(sum)
	if sum < 256 {
		return false
	}
	if nextRandom != nil {
		*accumulator = uint8(nextRandom() & 63)
	}
	return true
}

// AdvancePart is called once in the source list order. Source lookup tables
// supply the wobble, eye-facing clamps and the asymmetric shot offsets.
func (s *FourthFinalGuardian) AdvancePart(index int, art *visualassets.GuardianGroup, input FourthGuardianInput, sine *[256]int8, box func(string) visualassets.CollisionBox, nextRandom func() uint32) (FourthGuardianEvents, error) {
	event := FourthGuardianEvents{MaximumScrollY: input.MaximumScrollY}
	if index < 0 || index >= 19 || len(art.Components) != 19 {
		return event, fmt.Errorf("invalid fourth guardian component")
	}
	if s.Defeated {
		return event, nil
	}
	part := &s.Parts[index]
	part.Flash = false
	switch index {
	case 0:
		if event.MaximumScrollY < 16 {
			event.MaximumScrollY = 16
		}
		offsets := art.MotionTables["body_vertical_offsets"]
		if len(offsets) != 16 {
			return event, fmt.Errorf("fourth guardian body offsets are missing")
		}
		s.BodyScrollDelta = int(int16(input.ScrollDelta + offsets[input.Frame&15]))
		part.Arc.Y += int32(s.BodyScrollDelta) << 16
		part.Collision.Left, part.Collision.Right = 1000, 1000
		if s.EyesRemaining == 0 {
			top := int(part.Arc.Y>>16) + 32
			part.Collision = CollisionRect{Left: 148, Top: top, Right: 172, Bottom: top + 8}
		}
		advance := part.Counter != 0
		if !advance {
			advance = fourthFire(&part.FireAccumulator, art.MotionParameters["alarm_rate"], nextRandom)
			event.WarningSound = advance
		}
		if advance {
			phase := uint8(part.Counter) + 1
			if int8(phase) < 0 {
				part.Counter = 0
			} else {
				part.Counter = int16(phase)
				event.WarningSound = event.WarningSound || phase == 120
			}
		}
		part.Pose = 0
		if part.Counter != 0 {
			part.Pose = 1
		}
	case 1, 2:
		part.Arc.Y += int32(s.BodyScrollDelta) << 16
		x, y := int(part.Arc.X>>16), int(part.Arc.Y>>16)
		part.Collision = CollisionRect{Left: x + 6, Top: y + 6, Right: x + 26, Bottom: y + 26}
		if part.Disabled {
			return event, nil
		}
		wanted := AimDirection(input.PlayerX-x-16, input.PlayerY-y-16)
		table := art.MotionTables["left_eye_headings"]
		if index == 2 {
			table = art.MotionTables["right_eye_headings"]
		}
		if len(table) != 8 {
			return event, fmt.Errorf("fourth eye heading table is missing")
		}
		part.Pose = table[wanted]
		part.Direction = int16(part.Pose)
		part.Arc.AngleFixed = part.Arc.AngleFixed&^0xffff | int32(uint16(part.Pose))
		if fourthFire(&part.FireAccumulator, art.MotionParameters["eye_fire_rate"], nextRandom) {
			xs, ys := art.MotionTables["eye_shot_offset_x"], art.MotionTables["eye_shot_offset_y"]
			if len(xs) != 8 || len(ys) != 8 {
				return event, fmt.Errorf("fourth eye shot offsets are missing")
			}
			event.ShotCount = 1
			event.Shots[0] = FourthGuardianShot{X: x + 16 + xs[part.Pose], Y: y + 16 + ys[part.Pose], Speed: art.MotionParameters["eye_shot_speed"], Direction: uint8(part.Pose), Animation: fmt.Sprintf("eye-shot-%d", part.Pose)}
		}
	case 3:
		part.Arc.Y += int32(s.BodyScrollDelta) << 16
		heading, velocity, acceleration := fourthTailCurve(input.Frame)
		part.Arc.AngleFixed = int32(heading^128) << 16
		part.Arc.AngularVelocity, part.Arc.AngularAcceleration = velocity, acceleration
		budget := int(s.Parts[0].Counter)
		if budget > 8 {
			budget = 128 - budget
			if budget > 8 {
				budget = 8
			}
		}
		part.Arc.Budget = budget
		part.Visible = budget != 0
		part.Collision = ActorCollisionRect(box(art.Components[index].Sprite), int(part.Arc.X>>16), int(part.Arc.Y>>16))
		if !part.Visible {
			part.Collision.Left, part.Collision.Right = 1000, 1000
		}
	default:
		part.Arc = s.Parts[index-1].Arc
		part.Arc.Advance(sine)
		part.Counter = int16(100 - part.Arc.Budget)
		part.Visible = part.Arc.Budget != 0
		part.Collision = ActorCollisionRect(box(art.Components[index].Sprite), int(part.Arc.X>>16), int(part.Arc.Y>>16))
		if !part.Visible {
			part.Collision.Left, part.Collision.Right = 1000, 1000
		}
	}
	return event, nil
}

// Strike leaves destroyed eyes present with their closed artwork. Both eyes
// must be disabled before the body's small central weakness accepts damage.
func (s *FourthFinalGuardian) Strike(index int, damage uint16, scrollY int) FourthGuardianEvents {
	var event FourthGuardianEvents
	if s.Defeated || index < 0 || index > 2 {
		return event
	}
	part := &s.Parts[index]
	if part.Disabled || index == 0 && s.EyesRemaining != 0 {
		return event
	}
	part.Flash = true
	hit := ApplyEnemyDamage(part.Health, damage)
	part.Health = hit.Health
	if !hit.Destroyed {
		return event
	}
	if index != 0 {
		part.Disabled, part.Pose, part.Direction = true, -1, -1
		part.Arc.AngleFixed = part.Arc.AngleFixed&^0xffff | 0xffff
		s.EyesRemaining--
		event.Explosion = true
		event.ExplosionX = int(part.Arc.X>>16) + 16
		event.ExplosionY = int(part.Arc.Y>>16) - scrollY + 32
		return event
	}
	s.Defeated = true
	event.Defeated, event.ClearMoving, event.AdvanceLevel = true, true, true
	event.ReleaseMoving = true
	event.CashPairs, event.ExplosionCount = 10, 20
	top := int(part.Arc.Y >> 16)
	event.ExplosionRectangle = CollisionRect{Left: 112, Top: top, Right: 207, Bottom: top + 79}
	return event
}
