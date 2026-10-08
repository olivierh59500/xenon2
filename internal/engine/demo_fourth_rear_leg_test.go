package engine

import (
	"maps"
	"reflect"
	"testing"
)

func TestFourthRearLegHookRetainsSourceCornerOptional(t *testing.T) {
	w := fourthCapturedCornerScene(t)
	var original demoNavigation
	original.practiced = true
	original.refresh(w)
	// The capture first publishes this cache at F708, after an Up step from
	// its F707 source start23/world4071. Rebuild that original geometric cache.
	original.targetX, original.pointTargetX, original.goal, original.frame = 0, 100, 3943, 707
	if !original.search(23, 4071, 3943) || len(original.path) != 90 || original.path[len(original.path)-1] != (demoNavPoint{254, 3942}) {
		t.Fatalf("original captured ordinary path was not reconstructed: count%d goal%d target%d pathTarget%d", len(original.path), original.goal, original.targetX, original.pathTargetX)
	}

	pilot := DemoPilot{practicedRoute: true, navigation: &original}
	originalBefore := original
	originalBefore.tiles = append([]uint16(nil), original.tiles...)
	originalBefore.path = append([]demoNavPoint(nil), original.path...)
	originalBefore.queue = append([]demoNavNode(nil), original.queue...)
	originalBefore.nodes = append([]demoNavNode(nil), original.nodes...)
	originalBefore.visited = maps.Clone(original.visited)
	originalBefore.pointClosed = maps.Clone(original.pointClosed)
	if original.touchCache != nil {
		cache := *original.touchCache
		originalBefore.touchCache = &cache
	}
	input, ok := pilot.FourthRearLegInput(w)
	if !ok || len(pilot.nativeMotion.commands) != 8 {
		t.Fatalf("captured rearward corner changed: accepted%v commands%d", ok, len(pilot.nativeMotion.commands))
	}
	t.Logf("fresh helper commands%d target%d,%d endpoint%+v first%+v expanded%d", len(pilot.nativeMotion.commands), pilot.fourthRearTargetX, pilot.fourthRearTargetY, pilot.nativeMotion.states[len(pilot.nativeMotion.states)-1].player, input, pilot.nativeMotion.expanded)
	if pilot.fourthRearNavigation == pilot.navigation || !reflect.DeepEqual(pilot.navigation, &originalBefore) {
		t.Fatal("native corner probe changed the generic navigation cache")
	}
	commands := append([]MotionInput(nil), pilot.nativeMotion.commands...)
	for index, want := range commands {
		if index > 0 {
			input, ok = pilot.FourthRearLegInput(w)
		}
		if !ok || input.Motion != want || !pilot.fourthRearLegOwnsMotion(w, input.Motion) {
			t.Fatal("live hook replaced a retained command")
		}
		guard := PresentationPilot{PALRefreshes: 3, planner: pilot}
		if got := guard.forecastOpeningGuard(w, input); got != input {
			t.Fatalf("source six-command guard changed command%d: %+v to%+v", index, input, got)
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		if !nativeMotionMatches(w, pilot.nativeMotion.states[index+1]) || w.Rewind != pilot.nativeMotion.states[index+1].rewind || w.Rewind.Timer != 0 || w.Equipment.Shield != 15 || w.Equipment.Lives != 1 || w.ContinueCredits != 2 {
			t.Fatal("live hook changed source motion/history/health")
		}
	}
	if !reflect.DeepEqual(pilot.navigation, &originalBefore) {
		t.Fatal("retained corner changed generic cache")
	}
}

func TestFourthRearLegHookDeclinesOutsideScopeOptional(t *testing.T) {
	source := fourthCapturedCornerScene(t)
	for _, change := range []func(*World){
		func(w *World) { w.ScrollY = 3696 }, func(w *World) { w.ScrollY = 3969 }, func(w *World) { w.FourthMiddle = &FourthMiddleGuardian{} }, func(w *World) { w.Rewind.Timer = 1 }, func(w *World) { w.Dive.Phase = 1 }, func(w *World) { w.Ready = true }, func(w *World) { w.Level.Number = 3 },
	} {
		w := *source
		change(&w)
		p := DemoPilot{practicedRoute: true, navigation: &demoNavigation{}}
		before := p
		if _, ok := p.FourthRearLegInput(&w); ok || !reflect.DeepEqual(p, before) {
			t.Fatal("unsupported corner probe changed its fallback planner")
		}
	}
	p := DemoPilot{}
	if _, ok := p.FourthRearLegInput(source); ok || p.fourthRearNavigation != nil {
		t.Fatal("unpracticed route was admitted")
	}
}
