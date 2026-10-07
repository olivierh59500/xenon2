package engine

// MotionInput describes directional controls independently of their device.
type MotionInput struct {
	Up, Down, Left, Right bool
}

// MotionContext supplies the current scroll limits and ordinary scroll step.
// VisitedScrollY is the furthest backwards position allowed by the level.
type MotionContext struct {
	ScrollY, VisitedScrollY int
	BaseScrollStep          int
}

// PlayerMotionState retains the original whole-pixel ship coordinates and
// horizontal inertia. Rendering may interpolate previous and current positions.
type PlayerMotionState struct {
	X, Y                   int
	Inertia                int
	SpeedTier              int
	ScrollStep             int
	ScrollReverseRequested bool
}

// Advance applies one original gameplay pass. The held horizontal input uses
// the full movement step while the stored inertia controls release drift and
// the banked ship image. Left input takes precedence when both sides are held.
func (s *PlayerMotionState) Advance(input MotionInput, context MotionContext) {
	s.ScrollStep = context.BaseScrollStep
	s.ScrollReverseRequested = false
	if input.Left {
		if s.Inertia > 0 {
			s.Inertia = 0
		}
		if s.Inertia > -6 {
			s.Inertia--
		}
	} else if input.Right {
		if s.Inertia < 0 {
			s.Inertia = 0
		}
		if s.Inertia < 6 {
			s.Inertia++
		}
	} else if s.Inertia > 0 {
		s.Inertia--
	} else if s.Inertia < 0 {
		s.Inertia++
	}

	// The original checks a vertical limit before movement. A step that crosses
	// the limit is clamped on the following pass, preserving its edge behavior.
	checkDown := true
	if input.Up {
		if s.Y > 16 {
			s.Y -= 3 + s.SpeedTier
		} else {
			s.Y = 16
			s.ScrollStep = 1
			checkDown = false
		}
	}
	if checkDown && input.Down {
		if s.Y < 176 {
			s.Y += 3 + s.SpeedTier
		} else {
			if context.ScrollY != context.VisitedScrollY {
				s.ScrollStep = -1
				s.ScrollReverseRequested = true
			}
			s.Y = 176
		}
	}

	step := s.Inertia
	if step > 0 && input.Right {
		step = 6
	} else if step < 0 && input.Left {
		step = -6
	}
	if s.SpeedTier == 0 {
		step >>= 1
	} else if s.SpeedTier >= 2 {
		step += step >> 1
	}
	s.X += step
	if s.X < 14 {
		s.X = 14
	} else if s.X > 304 {
		s.X = 304
	}
}

// BankFrame selects one of the five unique ship images from its inertia.
func (s PlayerMotionState) BankFrame() int {
	switch {
	case s.Inertia <= -4:
		return 0
	case s.Inertia < 0:
		return 1
	case s.Inertia == 0:
		return 2
	case s.Inertia <= 3:
		return 3
	default:
		return 4
	}
}
