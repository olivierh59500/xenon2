package engine

import (
	"testing"
)

func originalSecondHeadPhaseFixture(t testing.TB, warmup int) *World {
	t.Helper()
	w := originalSecondNodeContactFixture(t, 228, 166)
	launch := &w.secondWaveArt.Launches[13]
	if launch.Path.ID != 14 {
		t.Fatal("original path14 launch absent")
	}
	for _, a := range w.Actors {
		if a.secondSegment == nil || a.secondSegment.Stream != 1 {
			continue
		}
		s, err := NewSecondDefenseSegment(*launch, *a.secondPart, 1, 2528, 1)
		if err != nil {
			t.Fatal(err)
		}
		a.secondSegment = &s
		a.path = &launch.Path
		a.motion = s.Motion
		a.Collision = CollisionRect{Right: -1, Bottom: -1}
	}
	w.ScrollDelta = 0
	for pass := 0; pass < warmup; pass++ {
		w.Frame++
		for _, a := range w.Actors {
			if a.Active && a.secondSegment != nil && a.secondSegment.Stream == 1 {
				if err := w.advanceSecondSegment(a); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	w.Player.Inertia = 4
	return w
}
func secondPhaseHead(t testing.TB, w *World) *WorldActor {
	t.Helper()
	for _, a := range w.Actors {
		if a.Active && a.secondSegment != nil && a.secondSegment.Stream == 1 && a.secondPart.Index == 0 {
			return a
		}
	}
	t.Fatal("original stream-one head is absent")
	return nil
}
func TestSecondStreamContactUsesPublishedPrefixBeforeMovement(t *testing.T) {
	w := originalSecondHeadPhaseFixture(t, 31)
	w.Player.X, w.Player.Y, w.Player.Inertia = 237, 171, 5
	head := secondPhaseHead(t, w)
	if int(head.X) != 245 || int(head.Y) != 160 || head.Collision != (CollisionRect{Left: 240, Top: 155, Right: 251, Bottom: 166}) {
		t.Fatal("bounded native path14 replay missed the captured head")
	}
	before := forecastIsolationDigest(w)
	prefix := thirdMiddlePlayerBounds(w, w.Player)
	if !prefix.Intersects(head.Collision) {
		t.Fatal("published source head did not touch the pre-movement ship")
	}
	player := w.Player
	player.Advance(MotionInput{Left: true}, MotionContext{ScrollY: w.ScrollY, VisitedScrollY: w.VisitedScrollY, BaseScrollStep: w.BaseScrollStep})
	var views [8 * ActorPoolCapacity]demoSecondDefenseView
	members, ok := demoSecondDefenseForecast(w, 1, [8]int{}, views[:])
	if !ok {
		t.Fatal("original stream prediction failed")
	}
	found := false
	for _, v := range views[:members] {
		if v.ID == head.ID {
			found = true
			if thirdMiddlePlayerBounds(w, player).Intersects(v.Bounds) {
				t.Fatal("fixture failed to distinguish old post-movement/future-row comparison")
			}
		}
	}
	if !found {
		t.Fatal("actual head identity disappeared from its forecast")
	}
	var actual WorldForecast
	if err := actual.Load(w); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		actual.AdvancePALTick()
	}
	result, err := actual.Advance(Input{Motion: MotionInput{Left: true}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Shield != 31 {
		t.Fatalf("actual original pre-movement callback did not apply8damage: %+v", result)
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("phase comparison changed source state")
	}
}

// This proves bounded legal avoidance with the new phase. The isolated arena
// also permits safe actions under the old beam, so no unique HP gain is claimed.
func TestSecondStreamPhaseReplanPreservesSourceShield(t *testing.T) {
	w := originalSecondHeadPhaseFixture(t, 29)
	w.Player.X, w.Player.Y, w.Player.Inertia = 219, 166, 3
	head := secondPhaseHead(t, w)
	if int(head.X) != 243 || int(head.Y) != 144 {
		t.Fatal("original preceding clear head pose changed")
	}
	pilot := DemoPilot{practicedRoute: true}
	for pass := 0; pass < 6; pass++ {
		before := forecastIsolationDigest(w)
		motion := demoSecondArenaBeam(w, &pilot, MotionInput{Right: true})
		if forecastIsolationDigest(w) != before {
			t.Fatal("stream phase planning changed live source state")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(Input{Motion: motion}); err != nil {
			t.Fatal(err)
		}
		if !w.PlayerAlive || w.Equipment.Shield != 39 {
			t.Fatalf("ordinary six-step avoidance lost shield at%d", pass)
		}
	}
}
