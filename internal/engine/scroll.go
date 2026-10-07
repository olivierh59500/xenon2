package engine

// ScrollState keeps the original moving upper bound separately from the stage's
// lower bound. The upper bound can expand again when a guardian locks an area.
type ScrollState struct {
	Y, Minimum, Maximum int
	DeviationPasses     int
	ActualStep          int
}

// Advance applies the original end-of-pass limits and sustained reverse speed.
// Holding down at the upper bound deliberately omits the sixteen-pixel buffer.
func (s *ScrollState) Advance(requestedStep, baseStep int, downHeld bool) {
	previous := s.Y
	if requestedStep != baseStep {
		s.DeviationPasses++
		if s.DeviationPasses >= 35 {
			requestedStep *= 2
		}
	} else {
		s.DeviationPasses = 0
	}
	s.Y -= requestedStep
	if s.Y < s.Minimum {
		s.Y = s.Minimum
	} else if s.Y > s.Maximum {
		s.Y = s.Maximum
	}
	s.ActualStep = previous - s.Y
	limit := s.Y
	if previous != s.Maximum || !downHeld {
		limit += 16
	}
	if limit <= s.Maximum {
		s.Maximum = limit
	}
}
