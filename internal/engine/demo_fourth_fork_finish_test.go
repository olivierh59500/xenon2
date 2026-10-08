package engine

import "testing"

func (p *DemoPilot) FourthCorridorBeforeFinishOrder(w *World) (Input, bool) {
	if !p.practicedRoute || !fourthCorridorActive(w) || w.Ready || !w.PlayerAlive || w.GameOver || w.Rewind.Timer != 0 || w.Coverage == nil || w.Level.PlayerStencil == nil {
		return Input{}, false
	}
	if w.Player.X <= 140 && w.Player.Y+w.ScrollY <= fourthForkTargetY {
		return Input{}, false
	}
	if motion, ok := p.nativeMotion.continueRoute(w, fourthForkTargetX, fourthForkTargetY); ok {
		return Input{Motion: motion}, true
	}
	if p.navigation == nil {
		p.navigation = &demoNavigation{}
	}
	p.navigation.practiced = true
	x, y, found := p.navigation.pointWaypoint(w, fourthForkTargetX, fourthForkTargetY)
	if !found {
		return Input{}, false
	}
	motion, found := p.nativeMotion.command(w, x, y, fourthForkTargetX, fourthForkTargetY)
	if !found {
		return Input{}, false
	}
	return Input{Motion: motion}, true
}

func TestFourthForkFinishGateKeepsRetainedSourceCommandsOptional(t *testing.T) {
	w := fourthOpeningSourceFixture(t, 4016, 85, 126, 0)
	w.cursor = EncounterCursor{MovingHighWater: 0, FixedHighWater: 0}
	p := DemoPilot{practicedRoute: true}
	before := forecastIsolationDigest(w)
	input, ok := p.FourthCorridorInput(w)
	if !ok || len(p.nativeMotion.commands) < 2 {
		t.Fatalf("source finish-crossing plan missing: accepted%v commands%d", ok, len(p.nativeMotion.commands))
	}
	t.Logf("admitted commands%d first%+v expanded%d", len(p.nativeMotion.commands), input, p.nativeMotion.expanded)
	if forecastIsolationDigest(w) != before {
		t.Fatal("fork planning changed source world")
	}
	commands := append([]MotionInput(nil), p.nativeMotion.commands...)
	crossed := false
	for index, want := range commands {
		if index > 0 {
			fulfills := w.Player.X <= 140 && w.Player.Y+w.ScrollY <= fourthForkTargetY
			if fulfills {
				// The old helper declines before consulting a still-matching
				// route. Its guard preview nevertheless predicts the next input.
				old := p
				if _, accepted := old.FourthCorridorBeforeFinishOrder(w); accepted {
					t.Fatal("old finish gate did not reproduce early abandonment")
				}

				crossed = true
			}
			input, ok = p.FourthCorridorInput(w)
		}
		if !ok || input.Motion != want {
			t.Fatalf("retained input abandoned at%d: accepted%v input%+v want%+v", index, ok, input.Motion, want)
		}
		sequence, valid := p.nativeMotion.guardSequence(w, input.Motion)
		if !valid || sequence[0] != want {
			t.Fatalf("current six-command preview missing at%d", index)
		}
		for offset := 0; offset < 6 && index+offset < len(commands); offset++ {
			if sequence[offset] != commands[index+offset] {
				t.Fatal("guard preview differs from retained source route")
			}
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		expected := p.nativeMotion.states[index+1]
		if !nativeMotionMatches(w, expected) || w.Rewind != expected.rewind || w.Rewind.Timer != 0 || w.Equipment.Shield != 39 || !w.PlayerAlive {
			t.Fatalf("source route differs after command%d", index)
		}
	}
	if !crossed {
		t.Fatal("source plan did not cross finish condition before its final input")
	}
	if _, accepted := p.FourthCorridorInput(w); accepted {
		t.Fatal("completed finish gate admitted a fresh route")
	}
	t.Logf("retained route finished at%d,%d camera%d without damage", w.Player.X, w.Player.Y, w.ScrollY)
}
