package engine

import "testing"

// This fixture supplies the captured merchant presentation timing. All prior
// combat and shop transactions run normally; the frontend test connects them
// to the default intro and the four preceding stage victories.
func capturedCurrentFifthMerchantReturn(t testing.TB) *World {
	t.Helper()
	w := capturedCurrentFifthReady(t)
	for _, code := range fifthCurrentOpeningControls {
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(fifthPracticeControl(code)); err != nil {
			t.Fatal(err)
		}
	}
	if !w.ShopReady || w.PendingExitDrops != 0 || w.Money != 1050 || w.Equipment.Shield != 3 || !w.FifthMiddle.Defeated {
		t.Fatal("current captured combat did not earn the native merchant")
	}
	rules := ShopRules{Level: 5, StockLimit: 6000}
	for _, position := range []SalePosition{SaleRear, SaleSide} {
		if _, err := rules.Sell(&w.Equipment, &w.Money, position); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []Item{ItemHealth2, ItemSideShot, ItemProtection, ItemPowerup, ItemPowerup} {
		if _, err := rules.Buy(&w.Equipment, &w.Money, item); err != nil {
			t.Fatal(err)
		}
	}
	rules.Leave(&w.Equipment)
	w.SetRandomState(RandomState{A: 3209480124, B: 3923103820})
	w.ResetBackgroundStars()
	w.ResumeShop()
	w.PrimeBackgroundStars(1)
	if w.Frame != fifthCurrentSecondPracticeStart || w.Equipment.Shield != 39 || w.Money != 550 || w.RandomState() != (RandomState{A: 4115466107, B: 4167859090}) || fifthPracticeMarker(w) != fifthCurrentSecondMarkers[0] {
		t.Fatal("captured current merchant return differs")
	}
	return w
}

func TestCurrentFifthSecondPracticeReachesFinalWithFullShieldOptional(t *testing.T) {
	w := capturedCurrentFifthMerchantReturn(t)
	p := PresentationPilot{PALRefreshes: 3}
	for i, code := range fifthCurrentSecondControls {
		before := ""
		if i == 0 || i == 1000 || i == len(fifthCurrentSecondControls)-1 {
			before = forecastIsolationDigest(w)
		}
		in, ok := p.fifthPracticedOpeningInput(w)
		if !ok || in != fifthPracticeControl(code) {
			t.Fatalf("current second-half input rejected at%d", i)
		}
		owner := p.fifthPractice
		if again, ok := p.fifthPracticedOpeningInput(w); !ok || again != in || p.fifthPractice != owner {
			t.Fatal("repeated current continuation changed route ownership")
		}
		if before != "" && forecastIsolationDigest(w) != before {
			t.Fatal("current continuation rehearsal changed the live world")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(in); err != nil {
			t.Fatal(err)
		}
		if !w.PlayerAlive || w.Equipment.Shield != 39 || w.Equipment.Lives != 3 || w.ContinueCredits != 2 || w.Rewind.Timer != 0 || w.Cheats.Enabled() || fifthPracticeMarker(w) != fifthCurrentSecondMarkers[i+1] {
			t.Fatalf("current second-half replay lost source outcome at%d", i)
		}
	}
	if w.Frame != 5507 || w.ScrollY != 415 || w.Checkpoint.ScrollY != 416 || w.FifthFinal == nil || w.FifthFinal.OuterRemaining != 18 || w.FifthFinal.CoreHealth != 20 || w.FifthFinal.Defeated || w.LevelFinished || w.Money != 850 || w.Score != 280050 || w.RandomState() != (RandomState{A: 1906578826, B: 683871768}) {
		t.Fatal("current original final admission differs")
	}
	if _, ok := p.fifthPracticedOpeningInput(w); ok || p.fifthPractice != nil {
		t.Fatal("current continuation exceeded its verified final admission")
	}
}

func TestCurrentFifthSecondPracticeRejectsForeignMerchantProfilesOptional(t *testing.T) {
	for _, change := range []func(*World){
		func(q *World) { q.Equipment.Protection = false },
		func(q *World) { q.Equipment.Side.Tier++ },
		func(q *World) { q.Equipment.Mounts[0].Serial++ },
		func(q *World) { q.Equipment.Shield-- },
		func(q *World) { q.Money++ },
		func(q *World) { r := q.RandomState(); r.A ^= 1; q.SetRandomState(r) },
		func(q *World) { q.FifthMiddle.Defeated = false },
		func(q *World) { q.Checkpoint.Loadout.Side.Serial++ },
	} {
		w := capturedCurrentFifthMerchantReturn(t)
		change(w)
		p := PresentationPilot{PALRefreshes: 3}
		before := forecastIsolationDigest(w)
		if _, ok := p.fifthPracticedOpeningInput(w); ok || p.fifthPractice != nil || forecastIsolationDigest(w) != before {
			t.Fatal("foreign current merchant admitted a route or changed live state")
		}
	}
}
