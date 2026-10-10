package engine

import "testing"

// This captured READY profile belongs to the current four-stage frontend run.
// It does not replay those victories; the frontend regression supplies them.
func capturedCurrentFifthReady(t testing.TB) *World {
	e := Equipment{WeaponLoadout: WeaponLoadout{Primary: WeaponSlot{Item: ItemForwardShot, Tier: 1, MaxTier: 2, Serial: 19}, Mounts: [4]WeaponSlot{{Item: ItemLaser, MaxTier: 2, Serial: 20}}, Rear: WeaponSlot{Item: ItemRearShot, Tier: 2, MaxTier: 2, Serial: 21}}, Shield: 39, Lives: 3, SpeedTier: 2, FirePeriod: 8, FireAdvance: 3, NextWeaponSerial: 21}
	data := playableOriginalWorldData(t, 5)
	data.InitialEquipment = &e
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.Score, w.DisplayScore = 203700, 203700
	w.ContinueCredits = 2
	// The shared-level initializer saves these pre-rebuild weapon serials.
	// Live equipment has newer serials after normal stage initialization.
	w.Checkpoint.Loadout = WeaponLoadout{
		Primary: WeaponSlot{Item: ItemForwardShot, Tier: 1, MaxTier: 2, Serial: 15},
		Mounts:  [4]WeaponSlot{{Item: ItemLaser, MaxTier: 2, Serial: 18}},
		Rear:    WeaponSlot{Item: ItemRearShot, Tier: 2, MaxTier: 2, Serial: 17},
	}
	w.SetRandomState(RandomState{A: 734901088, B: 1533863630})
	w.ResetBackgroundStars()
	w.PrimeBackgroundStars(2)
	w.Ready = true
	if err := w.Step(Input{Fire: true}); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestCurrentFifthPracticeReachesNativeMiddleAdmissionOptional(t *testing.T) {
	w := capturedCurrentFifthReady(t)
	p := PresentationPilot{PALRefreshes: 3, FourthLookahead: 12}
	minimum := 39
	for i, code := range fifthCurrentOpeningControls[:2241] {
		before := ""
		if i == 0 || i == 1185 || i == 2240 {
			before = forecastIsolationDigest(w)
		}
		input, ok := p.fifthPracticedOpeningInput(w)
		if !ok || input != fifthPracticeControl(code) {
			t.Fatalf("current control rejected at%d F%d", i, w.Frame)
		}
		owner := p.fifthPractice
		if repeated, ok := p.fifthPracticedOpeningInput(w); !ok || repeated != input || p.fifthPractice != owner {
			t.Fatal("repeated current control changed input or ownership")
		}
		if before != "" && forecastIsolationDigest(w) != before {
			t.Fatal("current rehearsal changed live source")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if !w.PlayerAlive || w.Equipment.Lives != 3 || w.ContinueCredits != 2 || w.Rewind.Timer != 0 || w.Cheats.Enabled() || fifthPracticeMarker(w) != fifthCurrentOpeningMarkers[i+1] {
			t.Fatalf("current replay diverged at%d F%d", i, w.Frame)
		}
	}
	if w.Frame != 2241 || w.ScrollY != 2367 || w.Checkpoint.ScrollY != 2368 || w.Equipment.Shield != 31 || minimum != 15 || w.Money != 300 || w.Score != 238350 || w.RandomState() != (RandomState{A: 648954619, B: 661526534}) || w.FifthMiddle == nil || w.FifthMiddle.Defeated || w.FifthMiddle.Parts[5].Health != 200 || w.ShopReady || w.LevelFinished {
		t.Fatalf("native current middle admission differs F%d HP%d min%d cash%d score%d", w.Frame, w.Equipment.Shield, minimum, w.Money, w.Score)
	}
	if in, ok := p.fifthPracticedOpeningInput(w); !ok || in != fifthPracticeControl(fifthCurrentOpeningControls[2241]) {
		t.Fatal("verified current middle continuation was rejected")
	}
	t.Log("Current captured fifth READY reaches the original middle admission: three ships, two continues, shield31/min15, cash300, native200-health core.")
}

func TestCurrentFifthPracticeRejectsForeignOwnersAndEntryProfilesOptional(t *testing.T) {
	w := capturedCurrentFifthReady(t)
	p := PresentationPilot{PALRefreshes: 3}
	in, ok := p.fifthPracticedOpeningInput(w)
	if !ok {
		t.Fatal("current READY profile not admitted")
	}
	for range 3 {
		w.AdvancePALTick()
	}
	if err := w.Step(in); err != nil {
		t.Fatal(err)
	}
	var replacement WorldForecast
	if err := replacement.Load(w); err != nil {
		t.Fatal(err)
	}
	before := forecastIsolationDigest(replacement.State())
	if _, ok := p.fifthPracticedOpeningInput(replacement.State()); ok || p.fifthPractice != nil || forecastIsolationDigest(replacement.State()) != before {
		t.Fatal("foreign world borrowed the current opening")
	}
	for _, change := range []func(*World){func(q *World) { q.Equipment.Rear.Tier-- }, func(q *World) { q.Money++ }, func(q *World) { r := q.RandomState(); r.A ^= 1; q.SetRandomState(r) }, func(q *World) { q.Equipment.Shield-- }, func(q *World) { q.Checkpoint.Loadout.Rear.Serial++ }, func(q *World) { q.Ready = true }} {
		q := capturedCurrentFifthReady(t)
		change(q)
		pilot := PresentationPilot{PALRefreshes: 3}
		before := forecastIsolationDigest(q)
		if _, ok := pilot.fifthPracticedOpeningInput(q); ok || pilot.fifthPractice != nil || forecastIsolationDigest(q) != before {
			t.Fatal("foreign entry admitted the current route or changed source")
		}
	}
}

// The captured fixture only supplies the prior READY profile; every hit, core
// death, coin and collection below comes from ordinary full game callbacks.
func TestCurrentFifthPracticeDefeatsMiddleAndCollectsAllExitCashOptional(t *testing.T) {
	w := capturedCurrentFifthReady(t)
	p := PresentationPilot{PALRefreshes: 3}
	minimum := 39
	defeated := false
	for i, c := range fifthCurrentOpeningControls {
		in, ok := p.fifthPracticedOpeningInput(w)
		if !ok || in != fifthPracticeControl(c) {
			t.Fatalf("current middle control rejected at%d", i)
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(in); err != nil {
			t.Fatal(err)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 3 || w.ContinueCredits != 2 || w.Rewind.Timer != 0 || w.Cheats.Enabled() || fifthPracticeMarker(w) != fifthCurrentOpeningMarkers[i+1] {
			t.Fatalf("current middle replay changed reserves or source outcome at%d", i)
		}
		if w.FifthMiddle != nil && w.FifthMiddle.Defeated && !defeated {
			defeated = true
			if w.Frame != 3193 || w.ScrollY != 2313 || w.Equipment.Shield != 3 || w.PendingExitDrops != 10 || w.Score != 248650 || w.Money != 300 || int16(w.FifthMiddle.Parts[5].Health) > 0 || w.LevelFinished || w.ShopReady {
				t.Fatal("current native core/reward boundary differs")
			}
		}
	}
	if !defeated || !w.ShopReady || w.PendingExitDrops != 0 || w.Frame != 3238 || w.ScrollY != 2268 || w.Equipment.Shield != 3 || minimum != 3 || w.Money != 1050 || w.Score != 248650 || w.ExitReady || w.LevelFinished || w.RandomState() != (RandomState{A: 2218123647, B: 3712480616}) {
		t.Fatalf("current intermediate merchant differs F%d HP%d cash%d", w.Frame, w.Equipment.Shield, w.Money)
	}
	if _, ok := p.fifthPracticedOpeningInput(w); ok || p.fifthPractice != nil {
		t.Fatal("current route continued past its verified merchant")
	}
	t.Log("Current source replay defeats the native middle core, collects all750 emitted cash and reaches its real merchant with three ships, two continues and1050 cash.")
}
