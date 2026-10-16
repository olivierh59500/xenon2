package engine

// FlamerSoundState preserves the held-trigger flag independently of particles.
// A completed sound restarts while held; release stops all effect voices once.
type FlamerSoundState struct {
	Started bool
	Counter int16
}

func (s *FlamerSoundState) Advance(held, materializing, effectActive bool) (start, stop bool) {
	if materializing {
		return false, false
	}
	if held {
		start = !s.Started || !effectActive
		if start {
			s.Counter = 1
		}
		s.Started = true
		return start, false
	}
	stop, s.Started = s.Started, false
	s.Counter = 0
	return false, stop
}
