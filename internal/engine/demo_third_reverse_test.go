package engine

import "testing"

func thirdReverseFixture(t *testing.T) *World {
	t.Helper()
	w := testWorld(t)
	w.Level.Number = 3
	final := NewThirdFinalState(80)
	w.ThirdFinal = &final
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 1986, 2002, 2002
	w.Player.X, w.Player.Y = 211, 176
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	return w
}

func TestDemoThirdReversePreludeIsReadOnly(t *testing.T) {
	w := thirdReverseFixture(t)
	for _, fixture := range []struct {
		camera, minimum, maximum, want int
		heartbeat                      bool
	}{
		{1986, 0, 2002, 2608, false},
		{209, 0, 225, 2608, false},
		{208, 0, 224, 224, false},
		{208, 0, 0, 32, false},
		{2800, 2800, 2816, 2800, true},
		{2800, 2800, 2816, 2800, false},
		{2801, 2800, 2817, 2817, false},
	} {
		w.ScrollY, w.MinimumScrollY, w.MaximumScrollY = fixture.camera, fixture.minimum, fixture.maximum
		w.ThirdStage.MiddleHeartbeat = fixture.heartbeat
		before, final, player, random, pool := w.ThirdStage, *w.ThirdFinal, w.Player, w.RandomState(), *w.Pool
		if got := demoScrollMaximum(w, fixture.camera, fixture.maximum); got != fixture.want {
			t.Fatalf("camera %d minimum %d: next-pass maximum %d, want %d", fixture.camera, fixture.minimum, got, fixture.want)
		}
		if w.ThirdStage != before || *w.ThirdFinal != final || w.Player != player || w.RandomState() != random || *w.Pool != pool {
			t.Fatal("reverse planning changed live heartbeat, final state, ship, RNG or actor storage")
		}
	}
}

func TestDemoThirdReverseForecastMatchesSixGameplayPasses(t *testing.T) {
	w := thirdReverseFixture(t)
	forecast := newDemoMotionForecast(w)
	input := MotionInput{Down: true}
	for pass := 1; pass <= 6; pass++ {
		if !forecast.advance(w, input) {
			t.Fatalf("clear source reverse rejected at pass %d", pass)
		}
		if err := w.Step(Input{Motion: input}); err != nil {
			t.Fatal(err)
		}
		if forecast.player != w.Player || forecast.scroll.Y != w.ScrollY || forecast.scroll.Minimum != w.MinimumScrollY || forecast.scroll.Maximum != w.MaximumScrollY || forecast.scroll.DeviationPasses != w.ScrollDeviationPasses || forecast.rewind != w.Rewind {
			t.Fatalf("pass %d diverged from gameplay: forecast ship %+v camera %+v, actual ship %+v camera %d/%d/%d", pass, forecast.player, forecast.scroll, w.Player, w.ScrollY, w.MinimumScrollY, w.MaximumScrollY)
		}
		if w.ScrollY != 1986+pass || !w.Player.ScrollReverseRequested {
			t.Fatalf("source stage prelude did not admit reverse at pass %d: camera %d reverse %v", pass, w.ScrollY, w.Player.ScrollReverseRequested)
		}
	}
}

func TestThirdMiddleHoldPrecedesOrdinaryPlayerMovement(t *testing.T) {
	for _, input := range []MotionInput{{}, {Down: true}} {
		w := thirdReverseFixture(t)
		w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 2800, 2800, 2816, 2816
		w.ThirdStage.MiddleHeartbeat = true
		for pass := 1; pass <= 2; pass++ {
			if err := w.Step(Input{Motion: input}); err != nil {
				t.Fatal(err)
			}
			// The stage sets ScrollStep to zero before the ordinary player
			// callback resets it to one. The first heartbeat retains the arena;
			// the following pass releases it instead of retaining a stale hold.
			wantCamera, wantMinimum := 2800, 2800
			if pass == 2 {
				wantCamera, wantMinimum = 2799, 0
			}
			if w.ScrollY != wantCamera || w.MinimumScrollY != wantMinimum || w.Player.ScrollStep != 1 {
				t.Fatalf("input %+v pass %d: camera %d minimum %d requested step %d", input, pass, w.ScrollY, w.MinimumScrollY, w.Player.ScrollStep)
			}
		}
	}
}

func TestDemoThirdOriginalPocketFindsSourceRetreatOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	// The reference route reached this right-hand pocket after the middle
	// merchant. The current pass reports 2002, while the next stage prelude
	// admits reversing to 2608 before the player callback.
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 1986, 2002, 2002
	w.Player.X, w.Player.Y = 211, 176
	before, random, player, pool := w.ThirdStage, w.RandomState(), w.Player, *w.Pool
	n := demoNavigation{practiced: true}
	x, y, found := n.waypoint(w, 2090)
	if !found || !n.retreat || y <= 2162 || !demoTestSegmentClear(&n, 211, 2162, demoNavPoint{x, y}) {
		t.Fatalf("source reverse passage was not found: %d/%d found %v retreat %v", x, y, found, n.retreat)
	}
	rear, left := 2162, 211
	for _, point := range n.path {
		if n.touching(point.x, point.y) || point.y > demoScrollMaximum(w, w.ScrollY, w.MaximumScrollY)+176 {
			t.Fatal("source route contains blocked or unreachable terrain")
		}
		rear, left = max(rear, point.y), min(left, point.x)
	}
	if rear < 2290 || left > 100 {
		t.Fatalf("route did not follow the rear U-turn to the left exit: rear %d left %d", rear, left)
	}
	if w.ThirdStage != before || w.RandomState() != random || w.Player != player || *w.Pool != pool {
		t.Fatal("finding the source route changed gameplay")
	}
}
