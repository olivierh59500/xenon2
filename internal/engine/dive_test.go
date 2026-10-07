package engine

import "testing"

func TestDiveConsumesOneChargeAndCanSurfaceEarly(t *testing.T) {
	s := DiveState{}
	charges := 3
	if !s.Request(&charges) || charges != 2 || s.Phase != 1 || s.Remaining != 136 {
		t.Fatal("dive did not consume exactly one charge")
	}
	for range 3 {
		s.AdvancePhase()
		s.Tick()
	}
	if s.Phase != 4 || s.Direction != 0 {
		t.Fatal("ship should remain submerged at phase four")
	}
	if !s.Request(&charges) || charges != 2 || s.Remaining != 1 {
		t.Fatal("surfacing request must not consume another charge")
	}
	s.AdvancePhase()
	s.Tick()
	for phase := 3; phase >= 0; phase-- {
		resurfaced := s.AdvancePhase()
		if s.Phase != phase || resurfaced != (phase == 0) {
			t.Fatalf("surfacing phase %d: %+v, finished=%v", phase, s, resurfaced)
		}
	}
	charges = 0
	if s.Request(&charges) {
		t.Fatal("empty dive inventory must not start a dive")
	}
}
