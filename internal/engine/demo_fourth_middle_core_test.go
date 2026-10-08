package engine

import "testing"

// This recorded fourth-stage entry retains its native forest and guardian
// callbacks. It does not claim to replay the preceding three stages here.
func TestFourthMiddleCoreReachesNativeMerchantFromRecordedEntryOptional(t *testing.T) {
	w := fourthMiddleSpecialistRecordedEntry(t)
	p := &PresentationPilot{PALRefreshes: 3}
	observed := WorldForecast{world: w}
	var next WorldForecast
	var gear Equipment
	firstID, hits, primaryHits, cannonHits, samples := 0, 0, 0, 0, 0
	started, defeated := false, false
	for w.Frame < 5000 {
		if w.FourthMiddle != nil && w.FourthMiddle.OuterTargets == 0 && !started {
			if w.Frame != 3983 || w.Equipment.Shield != 27 || w.FourthMiddle.Parts[4].Health != 175 || w.RandomState() != (RandomState{A: 515784913, B: 880257508}) {
				t.Fatal("native all-outer endpoint changed")
			}
			started, firstID, gear = true, w.nextActorID, w.Equipment
		}
		before := ""
		sample := started && (w.Frame == 3983 || w.Frame == 4200 || w.Frame == 4400)
		if sample {
			before = forecastIsolationDigest(w)
		}
		input := p.NormalInput(w)
		if sample {
			core := p.fourthMiddleCore
			if core == nil {
				t.Fatal("exposed core did not acquire its own controller")
			}
			at, plan := core.at, core.plan
			if repeated := p.NormalInput(w); repeated != input || core.at != at || core.plan != plan || forecastIsolationDigest(w) != before {
				t.Fatal("same-frame core sampling changed input, plan or native world")
			}
			if err := next.Load(w); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				next.AdvancePALTick()
			}
			if _, err := next.Advance(input); err != nil {
				t.Fatal(err)
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("core preview mutated its source")
			}
			samples++
		}
		for range 3 {
			w.AdvancePALTick()
		}
		observe := func(owner, id int, area CollisionRect, useful bool) {
			if !started || id <= firstID || owner != 0 && owner != 1 {
				return
			}
			var order [ActorPoolCapacity]*WorldActor
			for _, actor := range w.orderedMovingActors(&order) {
				if !actor.Active || !actor.Collision.Intersects(area) {
					continue
				}
				index := actor.fourthIndex - 1
				if useful && (index == 4 || index == 5) {
					hits++
					if owner == 0 {
						primaryHits++
					} else {
						cannonHits++
					}
				}
				break
			}
		}
		_, err := observed.AdvanceWeaponObserved(input, func(event WeaponPointImpact) {
			if event.Kind != "small-shot" {
				return
			}
			_, useful := presentationFirstPointImpact(w, event.X, event.Y)
			observe(event.OwnerSlot, event.ProjectileID, CollisionRect{Left: event.X, Right: event.X, Top: event.Y, Bottom: event.Y}, useful)
		}, func(event WeaponRectImpact) {
			observe(event.OwnerSlot, event.ProjectileID, event.Area, guardianRectImpactUseful(w, event))
		})
		if err != nil {
			t.Fatal(err)
		}
		if sample && forecastDigest(w) != forecastDigest(next.State()) {
			t.Fatal("three-tick core forecast differs from native World.Step")
		}
		if !started {
			continue
		}
		if !w.PlayerAlive || w.Equipment != gear || w.Equipment.Lives != 1 || w.ContinueCredits != 2 || w.Cheats.Enabled() || fourthAdmissionTerrainUnsafe(w) {
			t.Fatalf("ordinary core route lost strict27 or source clearance at F%d", w.Frame)
		}
		if w.FourthMiddle.Defeated && !defeated {
			// Native signed subtraction retains1-2 as0xffff when it kills.
			if w.Frame != 4496 || w.FourthMiddle.Parts[4].Health != 0xffff || w.PendingExitDrops != 10 || hits != 88 || samples != 3 {
				t.Fatalf("native core/cash boundary differs: F%d health%d pending%d hits%d samples%d", w.Frame, w.FourthMiddle.Parts[4].Health, w.PendingExitDrops, hits, samples)
			}
			defeated = true
			t.Logf("native core defeated F%d shield27 primaryHits%d cannonHits%d pendingCoins%d", w.Frame, primaryHits, cannonHits, w.PendingExitDrops)
		}
		if w.ShopReady {
			if !defeated || int16(w.FourthMiddle.Parts[4].Health) > 0 || w.PendingExitDrops != 0 || w.LevelFinished || w.FourthMiddle.OuterTargets != 0 || primaryHits == 0 || cannonHits == 0 || p.fourthMiddleCore != nil {
				t.Fatal("merchant bypassed native kill, cash or controller retirement")
			}
			t.Logf("real native middle merchant F%d shield27 money%d", w.Frame, w.Money)
			return
		}
	}
	t.Fatal("bounded ordinary core route did not reach the native merchant")
}
