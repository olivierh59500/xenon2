package engine

import (
	"runtime"
	"testing"
)

func TestThirdMiddleWorkersMatchSerialSourceDecisionsOptional(t *testing.T) {
	w := expertMiddleForecastScene(t)
	serial, parallel := PresentationPilot{PALRefreshes: 3}, PresentationPilot{PALRefreshes: 3}
	decisions := 0
	for pass := 0; pass < 64 && w.PlayerAlive && !w.ThirdMiddle.Defeated; pass++ {
		before := forecastIsolationDigest(w)
		want := serial.forecastThirdMiddleSerial(w, Input{})
		got := parallel.forecastThirdMiddleInput(w, Input{})
		if got != want {
			t.Fatalf("frame%d input%+v differs serial%+v", w.Frame, got, want)
		}
		if forecastIsolationDigest(w) != before {
			t.Fatal("candidate evaluation changed the live source")
		}
		if forecastIsolationDigest(parallel.forecast.State()) != forecastIsolationDigest(serial.forecast.State()) {
			t.Fatalf("frame%d final candidate snapshot differs", w.Frame)
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(want); err != nil {
			t.Fatal(err)
		}
		decisions++
	}
	if decisions != 64 {
		t.Fatalf("only%d source decisions were compared", decisions)
	}
	t.Logf("%d actual source input/snapshot comparisons", decisions)
}

func TestThirdMiddleWorkersKeepStatefulContinuationSerialOptional(t *testing.T) {
	w := expertMiddleForecastScene(t)
	for _, index := range []int{3, 4} {
		w.damageThirdMiddle(w.thirdMiddleActors[index], w.ThirdMiddle.EyeHealth[index-3])
	}
	if !w.ThirdMiddle.Defeated {
		t.Fatal("source eye damage callbacks did not defeat the guardian")
	}
	before := forecastIsolationDigest(w)
	for _, independent := range []bool{true, false} {
		var forecast WorldForecast
		var policy DemoPilot
		if err := forecast.Load(w); err != nil {
			t.Fatal(err)
		}
		prepareThirdMiddlePolicy(&policy, forecast.State(), w.ScrollY)
		candidate, err := evaluateThirdMiddleCandidate(&forecast, &policy, MotionInput{}, Input{}, 3, independent)
		if err != nil || candidate.stateful != independent {
			t.Fatalf("independent%v stateful%v err%v", independent, candidate.stateful, err)
		}
		if independent && forecast.State().Frame != w.Frame+3 {
			t.Fatal("worker entered stateful continuation before asking for serial replay")
		}
		if !independent && forecast.State().Frame <= w.Frame+3 {
			t.Fatal("serial evaluator did not continue the ordinary post-defeat route")
		}
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("continuation evaluation changed source defeat rewards or state")
	}
}

func TestThirdMiddleWorkersUseSerialWithOneProcessorOptional(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	w := expertMiddleForecastScene(t)
	serial, single := PresentationPilot{PALRefreshes: 3}, PresentationPilot{PALRefreshes: 3}
	want := serial.forecastThirdMiddleSerial(w, Input{})
	got := single.forecastThirdMiddleInput(w, Input{})
	if got != want || single.middleWorkers != nil || forecastIsolationDigest(single.forecast.State()) != forecastIsolationDigest(serial.forecast.State()) {
		t.Fatal("single-processor path changed the original serial evaluator")
	}
}
