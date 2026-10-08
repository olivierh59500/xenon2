package engine

import "testing"

func TestNativeClearTerminalExecutesCapturedExactAlignment(t *testing.T) {
	w := nativeMotionFixture(t, 255, 16, 1836)
	w.MaterializationFrames = 0
	n := demoNavigation{practiced: true}
	n.refresh(w)
	before := forecastIsolationDigest(w)
	var p nativeMotionPlanner
	first, ok := p.commandClearTerminal(w, &n, 256, 1832)
	if !ok || len(p.commands) != 20 || first != (MotionInput{}) || p.expanded != 0 {
		t.Fatalf("captured terminal plan differs: ok%v commands%d first%+v expansions%d", ok, len(p.commands), first, p.expanded)
	}
	for i, input := range p.commands {
		if i < 15 && input != (MotionInput{}) {
			t.Fatal("source camera padding did not precede horizontal alignment")
		}
	}
	if before != forecastIsolationDigest(w) {
		t.Fatal("terminal solver changed live state")
	}
	for index := 0; index < len(p.commands); index++ {
		input := first
		if index != 0 {
			input, ok = p.continueRoute(w, 256, 1832)
			if !ok {
				t.Fatalf("retained source plan invalidated at%d", index)
			}
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(Input{Motion: input}); err != nil {
			t.Fatal(err)
		}
		if !nativeMotionMatches(w, p.states[index+1]) || w.Rewind.Timer != 0 || w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
			t.Fatalf("terminal macro diverged at%d", index)
		}
	}
	if w.Player.X != 256 || w.Player.Y != 16 || w.ScrollY != 1816 {
		t.Fatal("ordinary terminal plan did not reach the exact source firing target")
	}
}

func TestNativeClearTerminalDeclinesBlockedAndOtherSourcePoses(t *testing.T) {
	for _, fixture := range []struct{ x, y, camera, tx, ty int }{{193, 176, 1896, 202, 2077}, {251, 176, 1732, 254, 1908}, {255, 16, 1836, 128, 1832}} {
		w := nativeMotionFixture(t, fixture.x, fixture.y, fixture.camera)
		n := demoNavigation{practiced: true}
		n.refresh(w)
		before := forecastIsolationDigest(w)
		var p nativeMotionPlanner
		if _, ok := p.commandClearTerminal(w, &n, fixture.tx, fixture.ty); ok || p.world != nil {
			t.Fatal("unsupported terminal alignment changed the existing native route")
		}
		if before != forecastIsolationDigest(w) {
			t.Fatal("declined terminal probe changed source")
		}
	}
	w := nativeMotionFixture(t, 255, 16, 1836)
	n := demoNavigation{practiced: true}
	n.refresh(w)
	// The real original map contains the center structure below this upper
	// corridor. A long horizontal target is outside this small alignment scope.
	if n.clearSegment(255, 1852, demoNavPoint{128, 1832}) {
		t.Fatal("original center obstruction no longer blocks the invalid-segment fixture")
	}
}

func TestNativeTerminalCacheRemainsOwnedByMiddleWorker(t *testing.T) {
	w := expertMiddleForecastScene(t)
	var source DemoPilot
	if _, ok := source.nativeMotion.terminalHorizontal.distance(255, 0, 256, 2); !ok {
		t.Fatal("source horizontal table unavailable")
	}
	var worker thirdMiddleForecastWorker
	if _, ok := worker.policy.nativeMotion.terminalHorizontal.distance(251, 0, 254, 2); !ok {
		t.Fatal("worker horizontal table unavailable")
	}
	owned := worker.policy.nativeMotion.terminalHorizontal.speeds[2]
	worker.evaluate(w, MotionInput{}, Input{}, 3, source)
	if worker.err != nil || worker.policy.nativeMotion.terminalHorizontal.speeds[2] != owned || owned == source.nativeMotion.terminalHorizontal.speeds[2] || source.nativeMotion.terminalHorizontal.speeds[2].targetX != 256 {
		t.Fatal("middle worker aliased the source planner's terminal cache")
	}
}
