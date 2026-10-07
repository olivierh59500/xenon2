package engine

import "xenon2/internal/visualassets"

// FifthFormationState keeps path state and the independent staggered emitter
// clock. Direction changes replace the image without advancing its animation.
type FifthFormationState struct {
	Motion          PathMotionState
	FireAccumulator uint8
	Sprite          string
	Duration        int
}

func NewFifthFormation(art visualassets.FifthFormationArt, variant, member, scroll int) (FifthFormationState, error) {
	motion, err := NewPathMotion(&art.Paths[variant], PathMotionConfig{Budget: art.MotionBudget, Delay: member * art.MemberSpacing})
	if err != nil {
		return FifthFormationState{}, err
	}
	motion.Y -= int32(scroll) << 16
	return FifthFormationState{Motion: motion, FireAccumulator: uint8((9 - member) * 25), Sprite: art.InitialSprite}, nil
}

// Advance returns a radial burst request. The original carry-triggered clock
// resets to zero, unlike ordinary aimed emitters that seed it from randomness.
func (s *FifthFormationState) Advance(art visualassets.FifthFormationArt, scrollDelta int, sine *[256]int8, random func() uint16) (bool, error) {
	if !s.Motion.Active {
		return false, nil
	}
	s.Motion.Y += int32(scrollDelta) << 16
	if int16(s.Motion.Y>>16) >= 400 {
		s.Motion.Active = false
		return false, nil
	}
	previous := ((uint32(s.Motion.AngleFixed) >> 16) + 16) >> 5 & 7
	if err := s.Motion.Advance(&art.Paths[s.Motion.PathID], sine, random); err != nil {
		return false, err
	}
	heading := ((uint32(s.Motion.AngleFixed) >> 16) + 16) >> 5 & 7
	if heading != previous {
		clip := art.HeadingAnimations[heading]
		s.Sprite, s.Duration = clip.Frames[0].Sprite, clip.Frames[0].Duration
	}
	sum := int(s.FireAccumulator) + art.FireRate
	s.FireAccumulator = uint8(sum)
	if sum >= 256 {
		s.FireAccumulator = 0
		return true, nil
	}
	return false, nil
}
