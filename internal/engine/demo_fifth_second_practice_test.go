package engine

import "testing"

// This fixture earns the intermediate victory and repair from the captured
// fifth-stage admission. Loader/star timing is explicit reference input; this
// does not establish a current carried five-stage frontend playthrough.
func fifthSecondPracticeSourceFixture(t testing.TB) *World {
	t.Helper()
	w := fifthPracticeSourceFixture(t)
	p := PresentationPilot{PALRefreshes: 3}
	for range fifthOpeningControls {
		in, ok := p.fifthPracticedOpeningInput(w)
		if !ok {
			t.Fatal("earned first-half source rejected")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(in); err != nil {
			t.Fatal(err)
		}
	}
	if !w.ShopReady || !w.FifthMiddle.Defeated || w.Money != 500 || w.Equipment.Shield != 7 {
		t.Fatal("source merchant boundary missing")
	}
	rules := ShopRules{Level: 5, StockLimit: 6000}
	if price, err := rules.Buy(&w.Equipment, &w.Money, ItemHealth2); err != nil || price != 500 {
		t.Fatalf("native repair differs: price%d %v", price, err)
	}
	w.SetRandomState(RandomState{A: 4184720610, B: 1806966132})
	w.ResetBackgroundStars()
	w.ResumeShop()
	w.PrimeBackgroundStars(1)
	if w.Frame != fifthSecondPracticeStart || w.Equipment.Shield != 39 || w.Money != 0 || w.RandomState() != (RandomState{A: 1681535540, B: 1568541330}) || fifthPracticeMarker(w) != fifthSecondMarkers[0] {
		t.Fatalf("recorded second-half source differs: F%d RNG%+v marker%016x want%016x", w.Frame, w.RandomState(), fifthPracticeMarker(w), fifthSecondMarkers[0])
	}
	return w
}

func TestFifthSecondPracticeReplaysOriginalFinalApproachOptional(t *testing.T) {
	w := fifthSecondPracticeSourceFixture(t)
	p := PresentationPilot{PALRefreshes: 3}
	minimum := 39
	for i, code := range fifthSecondControls {
		in, ok := p.fifthPracticedOpeningInput(w)
		if !ok || in != fifthPracticeControl(code) {
			t.Fatalf("second-half command rejected at%d F%d", i, w.Frame)
		}
		owner := p.fifthPractice
		if repeated, handled := p.fifthPracticedOpeningInput(w); !handled || repeated != in || p.fifthPractice != owner {
			t.Fatal("repeated second-half control consumed its route")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(in); err != nil {
			t.Fatal(err)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if !w.PlayerAlive || w.Equipment.Lives != 1 || w.ContinueCredits != 2 || w.Rewind.Timer != 0 || fifthPracticeMarker(w) != fifthSecondMarkers[i+1] {
			t.Fatalf("second-half native replay diverged at%d", i)
		}
	}
	if w.Frame != 5402 || w.FifthFinal == nil || w.FifthFinal.OuterRemaining != 18 || w.FifthFinal.CoreHealth != 20 || w.Equipment.Shield != 35 || minimum != 35 || w.Money != 200 || w.Score != 241630 || w.RandomState() != (RandomState{A: 4235635824, B: 2611229262}) {
		t.Fatal("real final admission differs from the original route")
	}
	if _, ok := p.fifthPracticedOpeningInput(w); ok {
		t.Fatal("final-approach route continued past its validated endpoint")
	}
}

func TestFifthSecondPracticeRejectsChangedMerchantReturnOptional(t *testing.T) {
	w := fifthSecondPracticeSourceFixture(t)
	before := forecastIsolationDigest(w)
	p := PresentationPilot{PALRefreshes: 3}
	if _, ok := p.fifthPracticedOpeningInput(w); !ok {
		t.Fatal("earned merchant return was not admitted")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("second segment admission changed live state")
	}
	for _, change := range []func(*World){
		func(q *World) { q.Equipment.Shield-- },
		func(q *World) { q.Money++ },
		func(q *World) { r := q.RandomState(); r.A ^= 1; q.SetRandomState(r) },
		func(q *World) { q.FifthMiddle.Defeated = false },
	} {
		q := fifthSecondPracticeSourceFixture(t)
		change(q)
		pilot := PresentationPilot{PALRefreshes: 3}
		before := forecastIsolationDigest(q)
		if _, ok := pilot.fifthPracticedOpeningInput(q); ok || pilot.fifthPractice != nil {
			t.Fatal("foreign shop state admitted the rehearsed second half")
		}
		if forecastIsolationDigest(q) != before {
			t.Fatal("second-half rejection changed live state")
		}
	}
}
