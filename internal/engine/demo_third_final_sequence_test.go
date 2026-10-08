package engine

import "testing"

func TestThirdFinalGuardRetainsEntrySequenceAcrossLaunchOptional(t *testing.T) {
	w := thirdFinalEntryPreparationScene(t)
	// Original-art prepared/cannon-destroyed fixture with a valid incoming
	// pose close to the actual launch boundary, not a carried-game replay.
	w.ScrollY, w.PreviousScrollY, w.RenderScrollY = 209, 209, 209
	w.Player = PlayerMotionState{X: 88, Y: 131, Inertia: 1, SpeedTier: 2, ScrollStep: 1}
	w.PreviousPlayer = w.Player
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("original boundary pose covered")
	}
	p := PresentationPilot{PALRefreshes: 3, planner: DemoPilot{practicedRoute: true}}
	motion, handled := p.thirdFinalEntryPreparation(w)
	if !handled {
		t.Fatal("original final-entry gates declined fixture")
	}
	sequence, owned := p.planner.nativeMotion.guardSequence(w, motion)
	t.Logf("ENTRY C%d motion%+v handled%v owned%v commands%+v", w.ScrollY, motion, handled, owned, p.planner.nativeMotion.commands)
	if !owned {
		t.Fatal("fixture did not create a source-native committed macro")
	}
	for pass := 0; pass < 2; pass++ {
		input := Input{Motion: sequence[pass]}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		// Consume the same retained successor through the real preparer.
		if pass == 0 {
			motion, handled = p.thirdFinalEntryPreparation(w)
			if !handled {
				t.Fatal("entry failed to continue before launch")
			}
		}
	}
	motion, handled = p.thirdFinalEntryPreparation(w)
	if !handled {
		t.Fatal("entry failed to continue after launch")
	}
	sequence, owned = p.planner.nativeMotion.guardSequence(w, motion)
	t.Logf("AFTER C%d launch%d motion%+v owned%v sequence%+v", w.ScrollY, w.ThirdFinal.LaunchCount, motion, owned, sequence)
	if w.ScrollY > 208 || w.ThirdFinal.LaunchCount != 1 || !owned {
		t.Fatal("source fixture did not retain a macro across208")
	}
	before := forecastIsolationDigest(w)
	planned := Input{Motion: motion}
	var native, held WorldForecast
	if err := native.Load(w); err != nil {
		t.Fatal(err)
	}
	if err := held.Load(w); err != nil {
		t.Fatal(err)
	}
	mixed := false
	for pass := 0; pass < 6; pass++ {
		mixed = mixed || sequence[pass] != motion
		for range 3 {
			native.AdvancePALTick()
			held.AdvancePALTick()
		}
		nr, err := native.Advance(Input{Motion: sequence[pass]})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := held.Advance(planned); err != nil {
			t.Fatal(err)
		}
		if !nr.Alive || nr.Shield != w.Equipment.Shield || native.State().Rewind.Timer != 0 {
			t.Fatalf("source retained macro unsafe at%d", pass)
		}
	}
	if !mixed || forecastDigest(native.State()) == forecastDigest(held.State()) {
		t.Fatal("fixture does not distinguish mixed macro from held motion")
	}
	got := p.forecastOpeningGuard(w, planned)
	if got != planned || forecastDigest(p.forecast.State()) != forecastDigest(native.State()) {
		t.Fatalf("guard did not preview the retained native macro: got%+v planned%+v", got, planned)
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("guard changed live source")
	}
	t.Logf("retained source preview%+v differs from held preview%+v", native.State().Player, held.State().Player)
}

func TestThirdFinalGuardUnownedRoutesMatchHeldForecastOptional(t *testing.T) {
	w := thirdFinalEntryPreparationScene(t)
	w.ScrollY, w.PreviousScrollY, w.RenderScrollY = 208, 208, 208
	w.Player = PlayerMotionState{X: 88, Y: 131, Inertia: 1, SpeedTier: 2, ScrollStep: 1}
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.ScrollY != 207 || w.ThirdFinal.LaunchCount != 1 {
		t.Fatal("native final launch fixture missing")
	}
	for _, foreign := range []bool{false, true} {
		p := PresentationPilot{PALRefreshes: 3}
		planned := Input{Motion: MotionInput{Down: true, Right: true}}
		var owner WorldForecast
		if foreign {
			if err := owner.Load(w); err != nil {
				t.Fatal(err)
			}
			p.planner.nativeMotion = nativeMotionPlanner{world: owner.State(), at: 1, commands: []MotionInput{planned.Motion}, states: []demoMotionForecast{newDemoMotionForecast(w)}}
		}
		var held WorldForecast
		if err := held.Load(w); err != nil {
			t.Fatal(err)
		}
		for range 6 {
			for range 3 {
				held.AdvancePALTick()
			}
			result, err := held.Advance(planned)
			if err != nil || !result.Alive || result.Shield != w.Equipment.Shield {
				t.Fatal("unowned source held fixture is unsafe")
			}
		}
		before := forecastIsolationDigest(w)
		current := p.forecastOpeningGuard(w, planned)
		if current != planned || forecastDigest(p.forecast.State()) != forecastDigest(held.State()) {
			t.Fatalf("unowned final case changed: foreign%v current%+v planned%+v", foreign, current, planned)
		}
		if forecastIsolationDigest(w) != before {
			t.Fatal("unowned guard changed live source")
		}
	}
}
