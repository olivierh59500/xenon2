package engine

import "testing"

func TestExpertWorldGuardPreservesLiveWorldAndHostCadenceOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	pilot := PresentationPilot{PALRefreshes: 2}
	before := forecastDigest(w)
	input := pilot.NormalInput(w)
	if forecastDigest(w) != before || pilot.PALRefreshes != 2 {
		t.Fatal("expert lookahead changed the live state or lost host cadence")
	}
	valid := false
	for _, motion := range demoDirections {
		valid = valid || input.Motion == motion
	}
	if !valid {
		t.Fatal("expert emitted a movement outside ordinary controls")
	}
	if pilot.forecast.State() == nil {
		t.Fatal("expert did not load its isolated callback forecast")
	}
}

func BenchmarkExpertWorldGuardOpening(b *testing.B) {
	w, err := NewWorld(playableOriginalWorldData(b, 3))
	if err != nil {
		b.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	pilot := PresentationPilot{}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		pilot.forecastOpeningGuard(w, Input{Motion: MotionInput{Right: true}, Fire: true})
	}
}
