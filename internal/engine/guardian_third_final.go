package engine

import (
	"fmt"
	"math/bits"
	"xenon2/internal/visualassets"
)

// ThirdFinalState retains the final stream's separate shared health and rotated
// path-selection seed. The first launch is path0; subsequent choices use1..6.
type ThirdFinalState struct {
	Health      uint16
	Seed        uint32
	LaunchCount int
	Defeated    bool
}

func NewThirdFinalState(health int) ThirdFinalState {
	return ThirdFinalState{Health: uint16(health), Seed: 0xe496aa8a}
}
func (s *ThirdFinalState) SelectLaunch() (int, error) {
	if s.Defeated {
		return -1, fmt.Errorf("final guardian is already defeated")
	}
	if s.LaunchCount == 0 {
		s.LaunchCount = 1
		return 0, nil
	}
	for attempts := 0; attempts < 256; attempts++ {
		s.Seed = bits.RotateLeft32(s.Seed, -5)
		choice := int(s.Seed & 7)
		if choice >= 1 && choice <= 6 {
			return choice, nil
		}
	}
	return -1, fmt.Errorf("final launch seed has no valid candidate")
}
func (s *ThirdFinalState) Strike(amount uint16) bool {
	if s.Defeated {
		return false
	}
	result := ApplyEnemyDamage(s.Health, amount)
	s.Health = result.Health
	s.Defeated = result.Destroyed
	return s.Defeated
}

type ThirdFinalMember struct {
	Motion           PathMotionState
	Animation        AnimationState
	AnimationData    visualassets.ActorAnimation
	Sprite           string
	FireAccumulator  uint8
	ResidualFireRate uint8
	Removed          bool
}

func NewThirdFinalMember(path *visualassets.Path, part visualassets.GuardianComponent, scrollY, requestedStep int) (ThirdFinalMember, error) {
	motion, err := NewPathMotion(path, PathMotionConfig{Budget: part.PathBudget, Delay: -part.InitialDelay})
	if err != nil {
		return ThirdFinalMember{}, err
	}
	motion.Y -= int32(scrollY+requestedStep) << 16
	motion.AngleFixed = 192 << 16
	return ThirdFinalMember{Motion: motion, Animation: NewAnimation(part.Animation), AnimationData: part.Animation, Sprite: part.Sprite}, nil
}

type ThirdFinalEvents struct {
	Updated   bool
	Shots     [9]ThirdGuardianShot
	ShotCount int
}

type ThirdFinalInput struct {
	ScrollDelta, PlayerX, PlayerY int
	FireRate                      uint8
	ShotSpeed                     int
}

func (s *ThirdFinalMember) Advance(path *visualassets.Path, part visualassets.GuardianComponent, sine *[256]int8, input ThirdFinalInput, random *RandomState) (ThirdFinalEvents, error) {
	var event ThirdFinalEvents
	if s.Removed {
		return event, nil
	}
	event.Updated = true
	s.Motion.Y += int32(input.ScrollDelta) << 16
	if part.Behavior == "worm-head" {
		s.Animation.Advance(s.AnimationData)
	}
	previousHeading := int((uint32(s.Motion.AngleFixed)>>16+16)>>5) & 7
	var next func() uint16
	if random != nil {
		next = func() uint16 { return uint16(random.Next()) }
	}
	if err := s.Motion.Advance(path, sine, next); err != nil {
		return event, err
	}
	s.Removed = !s.Motion.Active
	if !s.Removed && s.ResidualFireRate != 0 {
		fire := EnemyFireState{Rate: s.ResidualFireRate, Accumulator: s.FireAccumulator}
		var next32 func() uint32
		if random != nil {
			next32 = random.Next
		}
		shot, fired, err := fire.Tick(next32, input.PlayerX-int(s.Motion.X>>16), input.PlayerY-int(s.Motion.Y>>16))
		if err != nil {
			return event, err
		}
		s.FireAccumulator = fire.Accumulator
		if fired {
			event.Shots[event.ShotCount] = ThirdGuardianShot{X: int(s.Motion.X >> 16), Y: int(s.Motion.Y >> 16), Speed: shot.Speed, Direction: shot.Direction}
			event.ShotCount++
		}
	}
	heading := int((uint32(s.Motion.AngleFixed)>>16+16)>>5) & 7
	if part.Behavior == "worm-head" {
		if previousHeading != heading {
			s.AnimationData = part.HeadingAnimations[heading]
			s.Animation = NewAnimation(s.AnimationData)
		}
		s.Sprite = s.Animation.Sprite(s.AnimationData)
		sum := int(s.FireAccumulator) + int(input.FireRate)
		s.FireAccumulator = uint8(sum)
		if sum >= 256 {
			// This controller deliberately retains the overflow without random reset.
			// Its eight original direction emissions are ordered7 down through0.
			for i := range 8 {
				event.Shots[event.ShotCount] = ThirdGuardianShot{X: int(s.Motion.X >> 16), Y: int(s.Motion.Y >> 16), Speed: input.ShotSpeed, Direction: uint8(7 - i), Animation: "final-worm-shot"}
				event.ShotCount++
			}
		}
	} else {
		s.Sprite = part.HeadingFrames[heading]
	}
	return event, nil
}

// ThirdStageState separates the middle arena's heartbeat from the final stream.
// The boundary consumes one stale heartbeat before releasing minimum scroll.
type ThirdStageState struct {
	MiddleHeartbeat          bool
	FinalCompletionRequested bool
}
type ThirdStageInput struct {
	RequestedStep                              int
	ScrollY, MinimumScrollY, MaximumScrollY    int
	MiddleUpdated, FinalUpdated, FinalDefeated bool
}
type ThirdStageEvents struct {
	ExtraBackgroundStep                       int
	MinimumScrollY, MaximumScrollY            int
	HoldRequestedStep, LaunchFinal, FinalCash bool
}

func (s *ThirdStageState) Advance(input ThirdStageInput) ThirdStageEvents {
	event := ThirdStageEvents{MinimumScrollY: input.MinimumScrollY, MaximumScrollY: input.MaximumScrollY}
	s.MiddleHeartbeat = s.MiddleHeartbeat || input.MiddleUpdated
	if input.MinimumScrollY == 2800 && input.ScrollY == 2800 {
		event.MaximumScrollY = 2800
		event.HoldRequestedStep = true
		event.ExtraBackgroundStep = input.RequestedStep
		if s.MiddleHeartbeat {
			s.MiddleHeartbeat = false
		} else {
			event.MinimumScrollY = 0
		}
	}
	if input.ScrollY > 208 {
		if event.MaximumScrollY < 2608 {
			event.MaximumScrollY = 2608
		}
		return event
	}
	if event.MaximumScrollY < 32 {
		event.MaximumScrollY = 32
	}
	event.LaunchFinal = !input.FinalUpdated && !input.FinalDefeated
	if input.FinalDefeated && !s.FinalCompletionRequested {
		s.FinalCompletionRequested = true
		event.FinalCash = true
	}
	return event
}
