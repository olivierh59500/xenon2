package engine

import "testing"

func thirdCommittedHoldFixture(t testing.TB, camera int) *World {
	t.Helper()
	w := nativeMotionFixture(t, 255, 176, camera)
	w.MaterializationFrames = 0
	a := thirdRouteCannonBirth(t, w, 224, 1328)
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = camera, 0, camera+16, camera+16
	w.Player = PlayerMotionState{X: 255, Y: 176, SpeedTier: 2, ScrollStep: 1}
	w.Rewind = NewTerrainRewind(camera, 255, 176)
	w.cursor = EncounterCursor{MovingHighWater: 0, FixedHighWater: 0}
	w.advanceThirdCannon(a)
	if w.Coverage.Touches(255, 176, camera, *w.Level.PlayerStencil) {
		t.Fatal("source firing-hold fixture is not clear")
	}
	return w
}

func TestThirdCommittedNativeCommandSurvivesFiringHold(t *testing.T) {
	w := thirdCommittedHoldFixture(t, 1306)
	before := forecastIsolationDigest(w)
	var p DemoPilot
	input, handled := p.ThirdCorridorInput(w)
	if !handled || p.nativeMotion.world != w || p.nativeMotion.at != 1 || len(p.nativeMotion.commands) != 2 {
		t.Fatal("original two-command source approach was not admitted")
	}
	if planned := p.nativeMotion.commands[0]; input.Motion != planned || !planned.Up || planned.Down {
		t.Fatalf("firing hold changed an admitted native command: planned%+v returned%+v", planned, input.Motion)
	}
	if _, ok := p.nativeMotion.guardSequence(w, input.Motion); !ok {
		t.Fatal("native command lost its exact safety preview")
	}
	if before != forecastIsolationDigest(w) {
		t.Fatal("planning changed source physics")
	}
	for range 3 {
		w.AdvancePALTick()
	}
	if err := w.Step(input); err != nil {
		t.Fatal(err)
	}
	if !nativeMotionMatches(w, p.nativeMotion.states[1]) {
		t.Fatal("ordinary source step no longer matches its retained plan")
	}
	continued, ok := p.nativeMotion.continueRoute(w, 256, 1464)
	if !ok || continued != p.nativeMotion.commands[1] {
		t.Fatal("correct first command did not retain the second command")
	}
}

func TestThirdUncommittedFiringHoldRemainsAvailable(t *testing.T) {
	w := thirdCommittedHoldFixture(t, 1288)
	before := forecastIsolationDigest(w)
	var p DemoPilot
	input, handled := p.ThirdCorridorInput(w)
	if !handled || !input.Motion.Down || p.nativeMotion.world != nil || len(p.nativeMotion.commands) != 0 {
		t.Fatalf("ordinary uncommitted firing hold changed: input%+v handled%v", input, handled)
	}
	if before != forecastIsolationDigest(w) {
		t.Fatal("fallback hold changed source state while planning")
	}
}
