package engine

import "testing"

func TestFifthFinalForecastRenewsOriginalReverseBoundOptional(t *testing.T) {
	for _, mode := range []string{"active", "disabled-body"} {
		w := fifthSecondPracticeSourceFixture(t)
		p := PresentationPilot{PALRefreshes: 3}
		for range fifthSecondControls {
			in, ok := p.fifthPracticedOpeningInput(w)
			if !ok {
				t.Fatal("final approach rejected")
			}
			for range 3 {
				w.AdvancePALTick()
			}
			if err := w.Step(in); err != nil {
				t.Fatal(err)
			}
		}
		// Isolate source arena scrolling, without changing native collision data.
		w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 220, 221, 221
		w.Player.X, w.Player.Y, w.Player.Inertia = 14, 176, 0
		w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
		if mode == "disabled-body" {
			w.fifthFinalActors[0].Active = false
			w.storeActorResidue(w.fifthFinalActors[0])
		}

		prediction := newDemoMotionForecast(w)
		for pass := 0; pass < 6; pass++ {
			before := forecastIsolationDigest(w)
			if !prediction.advance(w, MotionInput{Down: true}) {
				t.Fatal("clear original side edge rejected")
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("reverse-bound forecast changed live arena")
			}
			if err := w.Step(Input{Motion: MotionInput{Down: true}}); err != nil {
				t.Fatal(err)
			}
			if !nativeMotionMatches(w, prediction) {
				t.Fatalf("mode%s pass%d: predicted%+v nativeC%d max%d P%+v", mode, pass, prediction.scroll, w.ScrollY, w.MaximumScrollY, w.Player)
			}
		}

	}
}
