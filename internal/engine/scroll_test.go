package engine

import "testing"

func TestScrollMovingUpperBoundAndReverseClamp(t *testing.T) {
	s := ScrollState{Y: 100, Minimum: 0, Maximum: 100}
	for range 25 {
		s.Advance(1, 1, false)
	}
	if s.Y != 75 || s.Maximum != 91 {
		t.Fatalf("forward scrolling upper bound: %+v", s)
	}
	for range 25 {
		s.Advance(-1, 1, false)
	}
	if s.Y != 91 || s.Maximum != 91 || s.ActualStep != 0 {
		t.Fatalf("reverse must stop at the current upper bound: %+v", s)
	}
}

func TestScrollSustainedReverseAndDownAtLimit(t *testing.T) {
	s := ScrollState{Y: 100, Minimum: 0, Maximum: 200, DeviationPasses: 33}
	s.Advance(-1, 1, false)
	if s.ActualStep != -1 {
		t.Fatal("reverse accelerated before thirty-five differing passes")
	}
	// A guardian can release a larger reverse range between updates.
	s.Maximum = 200
	s.Advance(-1, 1, false)
	if s.ActualStep != -2 {
		t.Fatal("sustained reverse did not double its step")
	}
	s = ScrollState{Y: 100, Maximum: 100}
	s.Advance(1, 1, true)
	if s.Y != 99 || s.Maximum != 99 {
		t.Fatal("holding down at the limit must omit the reverse buffer")
	}
}
