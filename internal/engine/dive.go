package engine

// DiveState retains the four underwater transition phases and their separate
// duration timer. Input requests occur after the current pass's timer update.
type DiveState struct {
	Phase, Direction, Remaining int
}

func (s *DiveState) Request(charges *int) bool {
	if s.Phase != 0 {
		s.Remaining = 1
		return true
	}
	if charges == nil || *charges <= 0 {
		return false
	}
	*charges--
	s.Phase, s.Direction, s.Remaining = 1, 1, 136
	return true
}

// AdvancePhase runs at the end of the ship update. Finishing a dive starts the
// original negative rewind state so resurfacing inside scenery can be escaped.
func (s *DiveState) AdvancePhase() (resurfaced bool) {
	if s.Phase == 0 {
		return false
	}
	s.Phase += s.Direction
	if s.Phase == 4 || s.Phase == 0 {
		s.Direction = 0
	}
	return s.Phase == 0
}

func (s *DiveState) Tick() {
	if s.Remaining > 0 {
		s.Remaining--
		if s.Remaining == 0 {
			s.Direction = -1
		}
	}
}
