package engine

import "testing"

// Recorded source entry; ordinary callbacks preserve the earned forest, tail,
// equipment changes and the specialist's subsequent native guardian hits.
func fourthMiddleSpecialistRecordedEntry(t testing.TB) *World {
	t.Helper()
	w := fourthGuardRecordedEntry(t, false)
	w.Score, w.DisplayScore = 130480, 130480
	w.SetRandomState(RandomState{A: 344191127, B: 4035761264})
	w.ResetBackgroundStars()
	w.PrimeBackgroundStars(2)
	return w
}

func TestFourthMiddleSpecialistClearsUpperAndRightFromRecordedEntryOptional(t *testing.T) {
	entry := fourthMiddleSpecialistRecordedEntry(t)
	before := forecastIsolationDigest(entry)
	var actual WorldForecast
	if err := actual.Load(entry); err != nil {
		t.Fatal(err)
	}
	w := actual.State()
	pilot := PresentationPilot{PALRefreshes: 3}
	upperFrame, rearHits, rightHits := uint64(0), 0, 0
	for pass := 0; pass < 4500; pass++ {
		input := pilot.NormalInput(w)
		for range 3 {
			actual.AdvancePALTick()
		}
		result, err := actual.AdvanceWeaponObserved(input, func(event WeaponPointImpact) {
			if event.OwnerSlot != 5 || event.Kind != "small-shot" {
				return
			}
			var storage [ActorPoolCapacity]*WorldActor
			for _, actor := range w.orderedMovingActors(&storage) {
				if !actor.Active || !actor.Collision.Contains(event.X, event.Y) {
					continue
				}
				_, useful := presentationFirstPointImpact(w, event.X, event.Y)
				if useful && (actor.fourthIndex == 18 || actor.fourthIndex == 20) {
					rearHits++
				}
				break
			}
		}, func(event WeaponRectImpact) {
			if upperFrame != 0 && event.OwnerSlot == 1 && event.Kind == "cannon-ball" && guardianRectImpactUseful(w, event) && event.Area.Intersects(w.fourthMiddleActors[18].Collision) {
				rightHits++
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.Alive || w.GameOver || w.Cheats.Enabled() || w.Equipment.Lives != 1 || w.ContinueCredits != 2 {
			t.Fatal("ordinary recorded lifetime changed")
		}
		if w.FourthMiddle == nil {
			continue
		}
		m := w.FourthMiddle
		if m.Parts[15].Disabled && w.Equipment.Shield != 27 {
			t.Fatalf("tail/specialist reserve changed at%d: %d", w.Frame, w.Equipment.Shield)
		}
		if m.Parts[17].Disabled && m.Parts[19].Disabled && upperFrame == 0 {
			upperFrame = w.Frame
			if upperFrame != 3555 || rearHits != 40 || m.Parts[18].Health != 4 || m.OuterTargets != 2 {
				t.Fatalf("native upper boundary differs: frame%d hits%d right%d outer%d", upperFrame, rearHits, m.Parts[18].Health, m.OuterTargets)
			}
			t.Logf("native upper gates F%d shield%d RearHits%d", w.Frame, w.Equipment.Shield, rearHits)
		}
		if upperFrame != 0 && m.Parts[18].Disabled {
			if w.Frame != 3673 || rightHits != 2 || m.Parts[16].Health != 20 || m.Parts[4].Health != 175 || m.OuterTargets != 1 || w.PendingExitDrops != 0 || w.ShopReady {
				t.Fatalf("native right boundary differs: frame%d hits%d left%d core%d outer%d", w.Frame, rightHits, m.Parts[16].Health, m.Parts[4].Health, m.OuterTargets)
			}
			if forecastIsolationDigest(entry) != before {
				t.Fatal("recorded controller replay changed original source")
			}
			t.Logf("native right gate F%d shield%d CannonHits%d sameShip1 credits2", w.Frame, w.Equipment.Shield, rightHits)
			return
		}
	}
	t.Fatal("bounded recorded specialist did not clear upper and right native gates")
}

func fourthMiddleSpecialistRecordedTail(t testing.TB) (*World, *PresentationPilot) {
	t.Helper()
	w := fourthMiddleSpecialistRecordedEntry(t)
	p := &PresentationPilot{PALRefreshes: 3}
	for pass := 0; pass < 4500; pass++ {
		input := p.NormalInput(w)
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		if !w.PlayerAlive || w.Equipment.Lives != 1 || w.ContinueCredits != 2 {
			t.Fatal("recorded tail lifetime changed")
		}
		if w.FourthMiddle != nil && w.FourthMiddle.Parts[15].Disabled {
			if w.Frame != 2542 || w.Equipment.Shield != 27 {
				t.Fatal("recorded native tail differs")
			}
			return w, p
		}
	}
	t.Fatal("recorded source did not reach native tail")
	return nil, nil
}

func TestFourthMiddleSpecialistScopeAndResourceValidationOptional(t *testing.T) {
	w, _ := fourthMiddleSpecialistRecordedTail(t)
	if !fourthMiddleSpecialistEligible(w, 3) {
		t.Fatal("earned native tail profile rejected")
	}
	for _, tier := range []int{-1, 0, 1, 2, 3} {
		var branch WorldForecast
		if err := branch.Load(w); err != nil {
			t.Fatal(err)
		}
		branch.State().Equipment.Rear.Tier = tier
		if got := fourthMiddleSpecialistEligible(branch.State(), 3); got != (tier >= 0 && tier <= 2) {
			t.Fatalf("native Rear Shot tier %d eligibility %v", tier, got)
		}
	}
	for _, mode := range []string{"level", "tail", "dead", "ready", "middle-defeated", "strobe", "partial", "dive", "shop", "drops", "PAL", "primary", "rear", "extra-mount", "missing-art", "missing-boxes", "missing-ships", "missing-weapons"} {
		t.Run(mode, func(t *testing.T) {
			var copy WorldForecast
			if err := copy.Load(w); err != nil {
				t.Fatal(err)
			}
			q, pal := copy.State(), 3
			switch mode {
			case "level":
				q.Level.Number = 3
			case "tail":
				q.FourthMiddle.Parts[15].Disabled = false
			case "dead":
				q.PlayerAlive = false
			case "ready":
				q.Ready = true
			case "middle-defeated":
				q.FourthMiddle.Defeated = true
			case "strobe":
				q.ScreenClearFrames = 1
			case "partial":
				q.stepContinuation.active = true
			case "dive":
				q.Dive.Phase = 1
			case "shop":
				q.ShopReady = true
			case "drops":
				q.PendingExitDrops = 1
			case "PAL":
				pal = 1
			case "primary":
				q.Equipment.Primary.Item = ItemDoubleShot
			case "rear":
				q.Equipment.Rear = WeaponSlot{}
			case "extra-mount":
				q.Equipment.Mounts[1].Item = ItemCannon
			case "missing-art":
				q.fourthMiddleArt = nil
			case "missing-boxes":
				q.movingSpriteBoxes = nil
			case "missing-ships":
				q.Level.Ships = nil
			case "missing-weapons":
				q.Weapons = nil
			}
			if fourthMiddleSpecialistEligible(q, pal) {
				t.Fatal("specialist escaped native gate/profile/resource scope")
			}
		})
	}
	if fourthMiddleSpecialistEligible(nil, 3) {
		t.Fatal("nil world eligible")
	}
}

func TestFourthMiddleSpecialistIdempotenceSourceMotionAndWorldOwnershipOptional(t *testing.T) {
	w, p := fourthMiddleSpecialistRecordedTail(t)
	for pass := 0; pass < 6; pass++ {
		before := forecastIsolationDigest(w)
		input := p.NormalInput(w)
		controller := p.fourthMiddleUpper
		if controller == nil {
			t.Fatal("native upper specialist not admitted")
		}
		at, phase := controller.at, controller.phase
		if repeated := p.NormalInput(w); repeated != input || controller.at != at || controller.phase != phase {
			t.Fatal("same-frame sample consumed a plan or phase")
		}
		if forecastIsolationDigest(w) != before {
			t.Fatal("specialist prediction changed live source")
		}
		want := controller.plan.after[at-1]
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		if fourthMiddleUpperState(w) != want {
			t.Fatal("committed ordinary input differs from complete source forecast")
		}
	}
	old := p.fourthMiddleUpper
	before := forecastIsolationDigest(w)
	var other WorldForecast
	if err := other.Load(w); err != nil {
		t.Fatal(err)
	}
	p.NormalInput(other.State())
	if p.fourthMiddleUpper == nil || p.fourthMiddleUpper == old || p.fourthMiddleUpper.world != other.State() {
		t.Fatal("world change retained another controller's ownership")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("new-world sampling changed the previous source world")
	}
}

func TestFourthMiddleCoastAlignmentConvergesAllNativeSpeedTwoStates(t *testing.T) {
	for _, target := range []int{80, 124, 204} {
		maximum, errorMaximum := 0, 0
		for x := 14; x <= 304; x++ {
			for inertia := -6; inertia <= 6; inertia++ {
				state := PlayerMotionState{X: x, Y: 100, Inertia: inertia, SpeedTier: 2}
				seen, settled := map[[2]int]bool{}, false
				for pass := 0; pass <= 64; pass++ {
					input, done := fourthMiddleCoastMotion(state, target)
					if done {
						if state.Inertia != 0 || target == 80 && state.X > 80 {
							t.Fatal("native coast stopped outside its legal flank")
						}
						maximum = max(maximum, pass)
						errorMaximum = max(errorMaximum, absDemo(state.X-target))
						settled = true
						break
					}
					key := [2]int{state.X, state.Inertia}
					if seen[key] {
						t.Fatalf("native coast cycle at target%d initial%d/%d", target, x, inertia)
					}
					seen[key] = true
					state.Advance(input, MotionContext{BaseScrollStep: 1})
				}
				if !settled {
					t.Fatal("native coast exceeded finite64-step contract")
				}
			}
		}
		t.Logf("target%d all3783 states settle max%d passes/error%d", target, maximum, errorMaximum)
	}
}
