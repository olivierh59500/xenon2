package engine

import "testing"

// This original-terrain fixture isolates retained commands. Its loadout and
// pose are arranged; the complete idle-start app regression earns its own state.
func thirdCorridorBranchFixture(t testing.TB) (*World, *PresentationPilot, Input) {
	t.Helper()
	w := nativeMotionFixture(t, 176, 176, 1622)
	w.MaterializationFrames = 0
	w.spawnEnemyShot(100, 0, EnemyShot{Direction: 4, Speed: 6})
	p := &PresentationPilot{PALRefreshes: 3, world: w, frame: w.Frame}
	p.planner.navigation = &demoNavigation{practiced: true}
	if _, _, ok := p.planner.navigation.pointWaypoint(w, 256, 1832); !ok {
		t.Fatal("original rear route missing")
	}
	p.thirdCorridorWatch.recovering = true
	before := forecastIsolationDigest(w)
	input, ok := p.chooseThirdCorridorBranch(w, Input{Motion: MotionInput{Down: true}}, [6]MotionInput{}, false)
	if !ok || p.thirdCorridorBranch == nil || p.thirdCorridorBranch.count != 6 {
		t.Fatal("clear rear branch was not retained")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("planning modified live source")
	}
	return w, p, input
}

func TestThirdCorridorBranchRetainsCommandsWithoutConsumingRepeatedCallsOptional(t *testing.T) {
	w, p, input := thirdCorridorBranchFixture(t)
	for pass := 0; pass < 6; pass++ {
		before := forecastIsolationDigest(w)
		got, ok := p.continueThirdCorridorBranch(w)
		at := p.thirdCorridorBranch.at
		repeated, again := p.continueThirdCorridorBranch(w)
		if !ok || !again || got != repeated || p.thirdCorridorBranch.at != at || pass == 0 && got != input {
			t.Fatal("reading a retained command consumed or changed it")
		}
		if forecastIsolationDigest(w) != before {
			t.Fatal("source replay modified the live world")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(got); err != nil {
			t.Fatal(err)
		}
		if retainedGuardStateKey(w) != p.thirdCorridorBranch.before[pass+1] || w.Equipment.Shield != 39 || w.Rewind.Timer != 0 {
			t.Fatal("retained rear branch diverged from actual callbacks")
		}
	}
	if _, ok := p.continueThirdCorridorBranch(w); ok {
		t.Fatal("completed branch continued beyond its six inputs")
	}
}

func TestThirdCorridorBranchRechecksAnExistingProjectileBeyondItsKeyOptional(t *testing.T) {
	w, p, _ := thirdCorridorBranchFixture(t)
	before := retainedGuardStateKey(w)
	shot := w.Projectiles[0]
	shot.Motion.X, shot.Motion.Y = int32(w.Player.X)<<16, int32(w.Player.Y-7)<<16
	shot.Motion.Direction, shot.Motion.Speed = 4, 6
	shot.X, shot.Y = float64(w.Player.X), float64(w.Player.Y-7)
	if retainedGuardStateKey(w) != before {
		t.Fatal("entity mutation unexpectedly changed the compact key")
	}
	var actual WorldForecast
	if err := actual.Load(w); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		actual.AdvancePALTick()
	}
	r, err := actual.Advance(p.thirdCorridorBranch.input[0])
	if err != nil || r.Shield >= w.Equipment.Shield {
		t.Fatal("changed projectile does not threaten the retained input")
	}
	digest := forecastIsolationDigest(w)
	if _, ok := p.continueThirdCorridorBranch(w); ok || p.thirdCorridorBranch.count != 0 {
		t.Fatal("changed projectile retained the unsafe rear branch")
	}
	if forecastIsolationDigest(w) != digest {
		t.Fatal("rejecting a changed projectile modified live state")
	}
}

func TestThirdCorridorBranchRejectsOtherOwnersAndCadencesOptional(t *testing.T) {
	for _, name := range []string{"world", "cadence", "ready", "frame"} {
		t.Run(name, func(t *testing.T) {
			w, p, _ := thirdCorridorBranchFixture(t)
			switch name {
			case "world":
				copy := *w
				w = &copy
			case "cadence":
				p.PALRefreshes = 2
			case "ready":
				w.Ready = true
			case "frame":
				w.Frame++
			}
			if _, ok := p.continueThirdCorridorBranch(w); ok || p.thirdCorridorBranch.count != 0 {
				t.Fatal("foreign source state retained a rear branch")
			}
		})
	}
}
