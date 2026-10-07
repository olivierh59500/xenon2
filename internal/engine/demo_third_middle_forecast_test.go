package engine

import "testing"

func expertMiddleForecastScene(t testing.TB) *World {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 2832, 168
	w.RestartCheckpoint()
	w.Ready = false
	var policy DemoPilot
	for pass := 0; pass < 120 && w.ThirdMiddle == nil; pass++ {
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(policy.NormalInput(w)); err != nil {
			t.Fatal(err)
		}
		if !w.PlayerAlive {
			t.Fatal("source encounter admission lost the fixture's ship")
		}
	}
	if w.ThirdMiddle == nil {
		t.Fatal("original stream did not admit the middle guardian")
	}
	return w
}

func TestExpertMiddleForecastIsReadOnlyAndKeepsTriggerReleaseOptional(t *testing.T) {
	w := expertMiddleForecastScene(t)
	pilot := PresentationPilot{PALRefreshes: 3}
	before := forecastIsolationDigest(w)
	input := pilot.forecastThirdMiddleInput(w, Input{})
	if forecastIsolationDigest(w) != before {
		t.Fatal("boss lookahead changed the live world")
	}
	ordinary := false
	for _, motion := range demoDirections {
		ordinary = ordinary || input.Motion == motion
	}
	if !ordinary {
		t.Fatal("boss lookahead returned a nonstandard movement")
	}
	w.blockedFireUntilRelease = true
	before = forecastIsolationDigest(w)
	if pilot.forecastThirdMiddleInput(w, Input{Fire: true}).Fire {
		t.Fatal("boss lookahead bypassed READY trigger release")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("trigger-release planning changed live state")
	}
}

func BenchmarkExpertThirdMiddleForecast(b *testing.B) {
	w := expertMiddleForecastScene(b)
	var pilot PresentationPilot
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		pilot.forecastThirdMiddleInput(w, Input{})
	}
}
