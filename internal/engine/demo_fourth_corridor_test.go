package engine

import "testing"

func TestFourthCorridorOriginalPointLegsExecuteNativeCommandsOptional(t *testing.T) {
	for _, fixture := range []struct {
		name                                 string
		x, y, camera, nextX, nextY, commands int
	}{
		{"before-fork", 147, 151, 4128, 123, 4255, 9},
		{"last-shared-branch", 211, 176, 4087, 187, 4239, 8},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			w := fourthOpeningSourceFixture(t, fixture.camera, fixture.x, fixture.y, 0)
			before := forecastIsolationDigest(w)
			navigation := demoNavigation{practiced: true}
			x, y, found := navigation.pointWaypoint(w, fourthForkTargetX, fourthForkTargetY)
			if !found || x != fixture.nextX || y != fixture.nextY {
				t.Fatalf("original fork leg changed: %d,%d found%v", x, y, found)
			}
			for index, point := range navigation.path {
				if navigation.touching(point.x, point.y) || index > 0 && !navigation.clearSegment(navigation.path[index-1].x, navigation.path[index-1].y, point) {
					t.Fatal("fork route crossed the original full ship stencil")
				}
			}
			var planner nativeMotionPlanner
			if !planner.search(w, x, y) || len(planner.commands) != fixture.commands {
				t.Fatalf("native fork commitment changed: commands%d expanded%d", len(planner.commands), planner.expanded)
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("point or native fork planning changed the live source state")
			}
			for index, motion := range planner.commands {
				for range 3 {
					w.AdvancePALTick()
				}
				if err := w.Step(Input{Motion: motion}); err != nil {
					t.Fatal(err)
				}
				if !nativeMotionMatches(w, planner.states[index+1]) || w.Rewind.Timer != 0 || w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) || !w.PlayerAlive {
					t.Fatalf("native fork pass%d differs or touches terrain: player%+v camera%d", index, w.Player, w.ScrollY)
				}
			}
			if w.Player.X != fixture.nextX || w.Player.Y+w.ScrollY != fixture.nextY {
				t.Fatal("ordinary commands did not reach the original clear fork leg")
			}
		})
	}
}

func TestFourthCorridorKeepsTheCommittedLegOptional(t *testing.T) {
	w := fourthOpeningSourceFixture(t, 4087, 211, 176, 0)
	pilot := DemoPilot{practicedRoute: true}
	before := forecastIsolationDigest(w)
	first, handled := pilot.FourthCorridorInput(w)
	if !handled || len(pilot.nativeMotion.commands) != 8 || first.Motion != pilot.nativeMotion.commands[0] {
		t.Fatal("practiced fourth corridor did not admit its original native commitment")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("admitting the fourth corridor changed source state")
	}
	commands := append([]MotionInput(nil), pilot.nativeMotion.commands...)
	for index, command := range commands {
		if index > 0 {
			input, handled := pilot.FourthCorridorInput(w)
			if !handled || input.Motion != command || pilot.nativeMotion.at != index+1 {
				t.Fatal("next native command was replaced by a moving geometric goal")
			}
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(Input{Motion: command}); err != nil {
			t.Fatal(err)
		}
		if !nativeMotionMatches(w, pilot.nativeMotion.states[index+1]) {
			t.Fatal("committed fourth leg differs from the source player/camera rules")
		}
	}
}

func TestFourthCorridorDoesNotInventALatePocketExitOptional(t *testing.T) {
	w := fourthOpeningSourceFixture(t, 4023, 254, 170, 0)
	pilot := DemoPilot{practicedRoute: true}
	before := forecastIsolationDigest(w)
	if _, handled := pilot.FourthCorridorInput(w); handled || len(pilot.navigation.path) != 0 {
		t.Fatal("late right pocket was reported as an ordinary geometric exit")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("failed late-pocket search changed the live source world")
	}
}

func TestFourthCorridorRetainsItsNarrowPracticedScopeOptional(t *testing.T) {
	w := fourthOpeningSourceFixture(t, 4087, 211, 176, 0)
	for _, fixture := range []struct {
		name      string
		practiced bool
		modify    func(*World)
	}{
		{"unpracticed", false, func(*World) {}},
		{"before-fork", true, func(w *World) { w.ScrollY = fourthForkStartCamera + 1 }},
		{"after-fork", true, func(w *World) { w.ScrollY = fourthForkEndCamera }},
		{"middle-admitted", true, func(w *World) { w.FourthMiddle = &FourthMiddleGuardian{} }},
		{"ready", true, func(w *World) { w.Ready = true }},
		{"rewinding", true, func(w *World) { w.Rewind.Timer = 1 }},
		{"destroyed", true, func(w *World) { w.PlayerAlive = false }},
		{"another-level", true, func(w *World) { w.Level.Number = 2 }},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			copy := *w
			fixture.modify(&copy)
			before := forecastIsolationDigest(&copy)
			pilot := DemoPilot{practicedRoute: fixture.practiced}
			if _, handled := pilot.FourthCorridorInput(&copy); handled || pilot.navigation != nil || pilot.nativeMotion.world != nil {
				t.Fatal("practiced fork route entered another source scope")
			}
			if forecastIsolationDigest(&copy) != before {
				t.Fatal("ineligible fork route changed source state")
			}
		})
	}
}
