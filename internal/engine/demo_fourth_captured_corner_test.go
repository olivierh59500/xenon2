package engine

import "testing"

// This initializes captured F716 player/scroll/equipment/RNG fields against the
// original map. It isolates the corner without replaying prior enemies/shots.
func fourthCapturedCornerScene(t testing.TB) *World {
	t.Helper()
	gear := NewEquipment()
	for _, item := range []Item{ItemPowerup, ItemCannon, ItemRearShot, ItemSpeedup, ItemSpeedup, ItemAutofire, ItemAutofire} {
		if !gear.ApplyItem(item) {
			t.Fatal("captured initial inventory rejected")
		}
	}
	gear.Lives, gear.Shield = 1, 15
	data := playableOriginalWorldData(t, 4)
	data.InitialEquipment = &gear
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.ContinueCredits = 2
	w.Frame = 716
	w.ScrollY, w.PreviousScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 3896, 3897, 2176, 3912, 3912
	w.ScrollDelta, w.ScrollDeviationPasses = 1, 0
	w.Player = PlayerMotionState{X: 23, Y: 156, Inertia: 0, SpeedTier: 2, ScrollStep: 1}
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.Ready, w.MaterializationFrames = false, 0
	w.SetRandomState(RandomState{A: 3217259765, B: 2406828990})
	w.cursor = EncounterCursor{MovingHighWater: 0, FixedHighWater: 0}
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) || !nativeMotionSupported(w) {
		t.Fatal("captured pre-rewind pose is covered/unsupported")
	}
	return w
}

func TestFourthCapturedCornerExecutesNativeRearLegOptional(t *testing.T) {
	w := fourthCapturedCornerScene(t)
	before := forecastIsolationDigest(w)
	var baseline WorldForecast
	if err := baseline.Load(w); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		baseline.AdvancePALTick()
	}
	if _, err := baseline.Advance(Input{Motion: MotionInput{Left: true}}); err != nil {
		t.Fatal(err)
	}
	if baseline.State().Rewind.Timer != 1 {
		t.Fatalf("captured Left does not enter native terrain rewind: player%+v camera%d timer%d", baseline.State().Player, baseline.State().ScrollY, baseline.State().Rewind.Timer)
	}
	// This is the captured path's source-clear next point and endpoint. Future
	// integration must obtain its endpoint from its own separate live cache.
	const nextX, nextWorldY, targetX, targetWorldY = 47, 4056, 254, 3942
	if nextWorldY <= w.Player.Y+w.ScrollY || w.Coverage.Touches(nextX, nextWorldY-w.ScrollY, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("captured target is not a clear rearward leg")
	}
	var planner nativeMotionPlanner
	first, ok := planner.command(w, nextX, nextWorldY, targetX, targetWorldY)
	if !ok {
		t.Fatalf("bounded source corner unavailable: expanded%d nodes%d", planner.expanded, planner.nodeCount())
	}
	t.Logf("native corner commands%d expanded%d nodes%d first%+v", len(planner.commands), planner.expanded, planner.nodeCount(), first)
	if forecastIsolationDigest(w) != before {
		t.Fatal("source planning changed live captured fixture")
	}
	commands := append([]MotionInput(nil), planner.commands...)
	for index, want := range commands {
		motion := first
		if index > 0 {
			motion, ok = planner.continueRoute(w, targetX, targetWorldY)
			if !ok || motion != want {
				t.Fatalf("retained command changed at%d", index)
			}
		}
		sequence, ok := planner.guardSequence(w, motion)
		if !ok || sequence[0] != want {
			t.Fatalf("source guard preview missing at%d", index)
		}
		for offset := 0; offset < 6 && index+offset < len(commands); offset++ {
			if sequence[offset] != commands[index+offset] {
				t.Fatal("guard changed the retained six-command prefix")
			}
		}
		guard := PresentationPilot{PALRefreshes: 3, planner: DemoPilot{practicedRoute: true, nativeMotion: planner}}
		input := Input{Motion: motion}
		if got := guard.forecastOpeningGuard(w, input); got != input {
			t.Fatalf("six-pass source guard changed retained command%d: planned%+v guarded%+v", index, input, got)
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		expected := planner.states[index+1]
		if !nativeMotionMatches(w, expected) || w.Rewind != expected.rewind || w.Rewind.Timer != 0 || w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) || w.Equipment.Shield != 15 || w.Equipment.Lives != 1 || w.ContinueCredits != 2 || !w.PlayerAlive {
			t.Fatalf("source corner diverged/damaged at%d: player%+v camera%d rewind%d shield%d", index, w.Player, w.ScrollY, w.Rewind.Timer, w.Equipment.Shield)
		}
	}
	if w.Player.X != nextX || w.Player.Y+w.ScrollY != nextWorldY {
		t.Fatal("retained route did not reach exact captured corner point")
	}
	t.Logf("completedF%d camera%d ship%d,%d shield%d credit%d rewind%d", w.Frame, w.ScrollY, w.Player.X, w.Player.Y, w.Equipment.Shield, w.ContinueCredits, w.Rewind.Timer)
}
