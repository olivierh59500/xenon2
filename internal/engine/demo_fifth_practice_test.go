package engine

import "testing"

// The original-resource fixture reproduces the recorded fifth admission, not
// the previous four victories. The app regression separately earns this state
// through the full intro, guardians, real shops and normal READY director.
func fifthPracticeSourceFixture(t testing.TB) *World {
	t.Helper()
	e := Equipment{WeaponLoadout: WeaponLoadout{
		Primary: WeaponSlot{Item: ItemForwardShot, Tier: 1, MaxTier: 2, Serial: 25},
		Mounts:  [4]WeaponSlot{{Item: ItemLaser, MaxTier: 2, Serial: 26}},
		Rear:    WeaponSlot{Item: ItemRearShot, Tier: 1, MaxTier: 2, Serial: 27},
	}, Shield: 39, Lives: 1, SpeedTier: 2, FirePeriod: 8, FireAdvance: 3, NextWeaponSerial: 27}
	data := playableOriginalWorldData(t, 5)
	data.InitialEquipment = &e
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.Score, w.DisplayScore = 176030, 176030
	w.ContinueCredits = 2
	w.Ready = true
	w.SetRandomState(RandomState{A: 1983374973, B: 138844162})
	w.ResetBackgroundStars()
	w.PrimeBackgroundStars(2)
	if err := w.Step(Input{Fire: true}); err != nil {
		t.Fatal(err)
	}
	if err := w.Step(Input{Motion: MotionInput{Up: true}}); err != nil {
		t.Fatal(err)
	}
	if w.Frame != 1 || w.RandomState() != (RandomState{A: 2028701702, B: 777268354}) || fifthPracticeMarker(w) != fifthOpeningMarkers[0] {
		t.Fatalf("captured fifth source fixture differs: F%d P%+v RNG%+v marker%016x want%016x", w.Frame, w.Player, w.RandomState(), fifthPracticeMarker(w), fifthOpeningMarkers[0])
	}
	return w
}

func TestFifthPracticeReplaysOriginalCallbacksAndDoesNotConsumeRepeatedInputsOptional(t *testing.T) {
	w := fifthPracticeSourceFixture(t)
	p := PresentationPilot{PALRefreshes: 3, world: w, frame: w.Frame}
	minimum := 39
	for index := range fifthOpeningControls {
		var before string
		if index == 0 || index == 800 || index == len(fifthOpeningControls)-1 {
			before = forecastIsolationDigest(w)
		}
		in, ok := p.fifthPracticedOpeningInput(w)
		if !ok || in != fifthPracticeControl(fifthOpeningControls[index]) {
			t.Fatalf("source command rejected at%d", index)
		}
		owner := p.fifthPractice
		if repeated, handled := p.fifthPracticedOpeningInput(w); !handled || repeated != in || p.fifthPractice != owner {
			t.Fatal("same-frame lookup consumed or changed its ordinary input")
		}
		if before != "" && forecastIsolationDigest(w) != before {
			t.Fatal("practice validation changed the source graph")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(in); err != nil {
			t.Fatal(err)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if !w.PlayerAlive || w.Equipment.Lives != 1 || w.ContinueCredits != 2 || w.Rewind.Timer != 0 || fifthPracticeMarker(w) != fifthOpeningMarkers[index+1] {
			t.Fatalf("native route diverged at%d", index)
		}
	}
	if w.Frame != 3195 || w.Checkpoint.ScrollY != 2368 || w.Equipment.Shield != 7 || minimum != 7 || !w.FifthMiddle.Defeated || !w.ShopReady || w.PendingExitDrops != 0 || w.Money != 500 {
		t.Fatal("source route omitted its real checkpoint or native health pickup")
	}
	if _, ok := p.fifthPracticedOpeningInput(w); ok || p.fifthPractice != nil {
		t.Fatal("completed practice route extended beyond its validated endpoint")
	}
}

func TestFifthPracticeRejectsChangedLiveProjectileBeyondItsMarkerOptional(t *testing.T) {
	w := fifthPracticeSourceFixture(t)
	p := PresentationPilot{PALRefreshes: 3}
	for len(w.Projectiles) == 0 && w.Frame < 500 {
		in, ok := p.fifthPracticedOpeningInput(w)
		if !ok {
			t.Fatal("practice rejected before native shot birth")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(in); err != nil {
			t.Fatal(err)
		}
	}
	if len(w.Projectiles) == 0 || w.InvulnerableFrames != 0 {
		t.Fatal("original route did not expose an ordinary enemy shot")
	}
	key := fifthPracticeMarker(w)
	shot := w.Projectiles[0]
	shot.Motion.X, shot.Motion.Y = int32(w.Player.X)<<16, int32(w.Player.Y-7)<<16
	shot.Motion.Direction, shot.Motion.Speed = 4, 6
	shot.X, shot.Y = float64(w.Player.X), float64(w.Player.Y-7)
	if fifthPracticeMarker(w) != key {
		t.Fatal("entity-only mutation unexpectedly changed the compact marker")
	}
	var actual WorldForecast
	if err := actual.Load(w); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		actual.AdvancePALTick()
	}
	r, err := actual.Advance(fifthPracticeControl(fifthOpeningControls[w.Frame-1]))
	if err != nil || r.Shield >= w.Equipment.Shield {
		t.Fatal("changed native shot does not threaten the proposed control")
	}
	before := forecastIsolationDigest(w)
	if _, ok := p.fifthPracticedOpeningInput(w); ok || p.fifthPractice != nil {
		t.Fatal("full callback validation accepted a changed projectile hidden from the marker")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("rejected control changed the live source")
	}
}

func TestFifthPracticeDropsChangedOwnersAndForeignEntryStatesOptional(t *testing.T) {
	w := fifthPracticeSourceFixture(t)
	p := PresentationPilot{PALRefreshes: 3}
	in, ok := p.fifthPracticedOpeningInput(w)
	if !ok {
		t.Fatal("canonical entry not admitted")
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
	if _, ok := p.fifthPracticedOpeningInput(replacement.State()); ok || p.fifthPractice != nil {
		t.Fatal("mid-route clone borrowed another world's practice")
	}
	if forecastIsolationDigest(replacement.State()) != before {
		t.Fatal("owner rejection modified its replacement world")
	}
	for _, change := range []func(*World, *PresentationPilot){
		func(q *World, p *PresentationPilot) { q.Level.Number = 4 },
		func(q *World, p *PresentationPilot) { q.Ready = true },
		func(q *World, p *PresentationPilot) { q.PlayerAlive = false },
		func(q *World, p *PresentationPilot) { q.ShopReady = true },
		func(q *World, p *PresentationPilot) { q.ScreenClearFrames = 1 },
		func(q *World, p *PresentationPilot) { q.stepContinuation.active = true },
		func(q *World, p *PresentationPilot) { q.Dive.Phase = 1 },
		func(q *World, p *PresentationPilot) { q.Equipment.Mounts[0].Tier++ },
		func(q *World, p *PresentationPilot) { r := q.RandomState(); r.B ^= 1; q.SetRandomState(r) },
		func(q *World, p *PresentationPilot) { q.blockedFireUntilRelease = true },
		func(q *World, p *PresentationPilot) { p.PALRefreshes = 2 },
		func(q *World, p *PresentationPilot) { q.Frame = uint64(len(fifthOpeningControls) + 1) },
	} {
		q := fifthPracticeSourceFixture(t)
		pilot := PresentationPilot{PALRefreshes: 3}
		change(q, &pilot)
		before := forecastIsolationDigest(q)
		if _, ok := pilot.fifthPracticedOpeningInput(q); ok || pilot.fifthPractice != nil {
			t.Fatal("foreign entry or lifecycle admitted rehearsed inputs")
		}
		if forecastIsolationDigest(q) != before {
			t.Fatal("scope rejection changed live simulation")
		}
	}
}

func BenchmarkFifthPracticeActiveMissileSceneOriginal(b *testing.B) {
	w := fifthPracticeSourceFixture(b)
	p := PresentationPilot{PALRefreshes: 3}
	for w.Frame < 800 {
		in, ok := p.fifthPracticedOpeningInput(w)
		if !ok {
			b.Fatal("practice rejected before benchmark scene")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(in); err != nil {
			b.Fatal(err)
		}
	}
	active := 0
	for _, a := range w.Actors {
		if a.Active && a.fixedAiming != nil {
			active++
		}
	}
	if active == 0 {
		b.Fatal("benchmark omitted native guided missiles")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, ok := p.fifthPracticedOpeningInput(w); !ok {
			b.Fatal("native next-step rehearsal validation failed")
		}
	}
}

func TestFifthPracticeRejectsChangedGuardianDuringAdmissionOptional(t *testing.T) {
	w := fifthPracticeSourceFixture(t)
	p := PresentationPilot{PALRefreshes: 3}
	for w.FifthMiddle == nil {
		in, ok := p.fifthPracticedOpeningInput(w)
		if !ok {
			t.Fatal("original route rejected before guardian birth")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(in); err != nil {
			t.Fatal(err)
		}
	}
	key := fifthPracticeMarker(w)
	w.FifthMiddle.Parts[5].Health--
	if fifthPracticeMarker(w) == key {
		t.Fatal("guardian health is absent from admission marker")
	}
	before := forecastIsolationDigest(w)
	if _, ok := p.fifthPracticedOpeningInput(w); ok || p.fifthPractice != nil {
		t.Fatal("changed guardian retained the rehearsed admission")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("guardian rejection altered live state")
	}
}
