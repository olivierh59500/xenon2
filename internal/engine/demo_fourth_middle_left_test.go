package engine

import "testing"

func fourthMiddleLeftRecordedStart(t testing.TB) (*World, *PresentationPilot) {
	t.Helper()
	w := fourthMiddleSpecialistRecordedEntry(t)
	p := &PresentationPilot{PALRefreshes: 3}
	for w.Frame < 4000 {
		input := p.NormalInput(w)
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		if !w.PlayerAlive || w.Cheats.Enabled() {
			t.Fatal("recorded lifetime changed")
		}
		if w.FourthMiddle != nil && w.FourthMiddle.Parts[18].Disabled {
			if w.Frame != 3673 || w.Equipment.Shield != 27 || w.FourthMiddle.Parts[16].Health != 20 || w.FourthMiddle.OuterTargets != 1 {
				t.Fatal("native left admission differs")
			}
			return w, p
		}
	}
	t.Fatal("recorded native right target was not defeated")
	return nil, nil
}

func TestFourthMiddleLeftClearsAllOuterTargetsFromRecordedEntryOptional(t *testing.T) {
	w, p := fourthMiddleLeftRecordedStart(t)
	gear := w.Equipment
	firstID, hits, samples := w.nextActorID, 0, 0
	// Observe the owned native World without changing its identity or physics.
	observed := WorldForecast{world: w}
	var forecast WorldForecast
	for w.Frame < 4500 {
		before := forecastIsolationDigest(w)
		input := p.NormalInput(w)
		sample := w.Frame == 3673 || w.Frame == 3800 || w.Frame == 3950
		if sample {
			left := p.fourthMiddleLeft
			phase, at := left.phase, left.at
			if repeated := p.NormalInput(w); repeated != input || left.phase != phase || left.at != at || forecastIsolationDigest(w) != before {
				t.Fatal("same-state input sampling consumed phase or native state")
			}
			if err := forecast.Load(w); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				forecast.AdvancePALTick()
			}
			if _, err := forecast.Advance(input); err != nil {
				t.Fatal(err)
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("prediction changed its source")
			}
			samples++
		}
		for range 3 {
			w.AdvancePALTick()
		}
		_, err := observed.AdvanceObserved(input, func(event WeaponPointImpact) {
			if event.OwnerSlot != 0 || event.Kind != "small-shot" || event.ProjectileID <= firstID {
				return
			}
			var order [ActorPoolCapacity]*WorldActor
			for _, actor := range w.orderedMovingActors(&order) {
				if !actor.Active || !actor.Collision.Contains(event.X, event.Y) {
					continue
				}
				_, useful := presentationFirstPointImpact(w, event.X, event.Y)
				if actor.fourthIndex == 17 && useful {
					hits++
				}
				break
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if sample && forecastDigest(w) != forecastDigest(forecast.State()) {
			t.Fatal("three-tick forecast differs from native World.Step")
		}
		if !w.PlayerAlive || w.Equipment != gear || w.Equipment.Lives != 1 || w.ContinueCredits != 2 || w.Cheats.Enabled() || fourthAdmissionTerrainUnsafe(w) {
			t.Fatal("ordinary left route lost its strict27 reserve or terrain clearance")
		}
		if w.FourthMiddle.Parts[16].Disabled {
			if w.Frame != 3983 || w.ScrollY != 2465 || w.Player.X != 116 || w.Player.Y != 176 || w.Player.Inertia != -1 || hits != 10 || samples != 3 || w.RandomState() != (RandomState{A: 515784913, B: 880257508}) || w.FourthMiddle.OuterTargets != 0 || w.FourthMiddle.Parts[4].Health != 175 || w.PendingExitDrops != 0 || w.ShopReady {
				t.Fatalf("native outer boundary differs: F%d hits%d samples%d rng%+v", w.Frame, hits, samples, w.RandomState())
			}
			t.Logf("native all outer targets defeated F%d shield27 primaryHits%d threeTickSamples%d", w.Frame, hits, samples)
			return
		}
	}
	t.Fatal("bounded ordinary left route did not clear its native target")
}

func TestFourthMiddleLeftDeclinesOtherScopesOptional(t *testing.T) {
	source, _ := fourthMiddleLeftRecordedStart(t)
	for _, name := range []string{"level3", "ready", "dead", "defeated", "shop", "pending", "dive", "screen-clear", "no-ships", "wrong-cadence"} {
		t.Run(name, func(t *testing.T) {
			var copy WorldForecast
			if err := copy.Load(source); err != nil {
				t.Fatal(err)
			}
			w := copy.State()
			p := PresentationPilot{PALRefreshes: 3, fourthMiddleLeft: &fourthMiddleLeftPilot{world: source}}
			switch name {
			case "level3":
				w.Level.Number = 3
			case "ready":
				w.Ready = true
			case "dead":
				w.PlayerAlive = false
			case "defeated":
				w.FourthMiddle.Defeated = true
			case "shop":
				w.ShopReady = true
			case "pending":
				w.PendingExitDrops = 1
			case "dive":
				w.Dive.Phase = 1
			case "screen-clear":
				w.ScreenClearFrames = 1
			case "no-ships":
				w.Level.Ships = nil
			case "wrong-cadence":
				p.PALRefreshes = 2
			}
			before := forecastIsolationDigest(w)
			if _, handled := p.fourthMiddleSpecialistInput(w); handled || p.fourthMiddleLeft != nil {
				t.Fatal("unsupported specialist retained a left phase")
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("declined scope changed native state")
			}
		})
	}
}

func TestFourthMiddleLeftResetsWorldAndFrameOptional(t *testing.T) {
	w, _ := fourthMiddleLeftRecordedStart(t)
	var another WorldForecast
	if err := another.Load(w); err != nil {
		t.Fatal(err)
	}
	fresh := fourthMiddleLeftPilot{}
	want := fresh.Input(w)
	p := fourthMiddleLeftPilot{world: another.State(), phase: fourthMiddleLeftPhase{stage: 2, goal: 1000, fireGoal: 1000, crossing: true}}
	if got := p.Input(w); got != want || p.phase != fresh.phase {
		t.Fatal("new world retained an old left phase")
	}
	p.phase = fourthMiddleLeftPhase{stage: 2, goal: 1000, fireGoal: 1000, crossing: true}
	p.plan.before[0].frame = w.Frame + 1
	if got := p.Input(w); got != want || p.phase != fresh.phase {
		t.Fatal("frame rollback retained an old left phase")
	}
}
