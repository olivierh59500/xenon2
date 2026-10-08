package engine

import "testing"

func TestFifthFinalGeometryRetainsOriginalRearPassageOptional(t *testing.T) {
	w := fifthSecondPracticeSourceFixture(t)
	p := PresentationPilot{PALRefreshes: 3}
	for range fifthSecondControls {
		in, ok := p.fifthPracticedOpeningInput(w)
		if !ok {
			t.Fatal("earned final admission rejected")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(in); err != nil {
			t.Fatal(err)
		}
	}
	// Isolate the observed right-side pose and its original arena coverage.
	// The source controller reopens the rear camera bound before movement.
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 118, 134, 134
	w.Player.X, w.Player.Y, w.Player.Inertia = 244, 176, 0
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	before := forecastIsolationDigest(w)
	n := demoNavigation{practiced: true}
	x, y, found := n.pointWaypointMinimum(w, 140, 480, 0)
	if !found || len(n.path) < 2 || !n.clearSegment(w.Player.X, w.ScrollY+w.Player.Y, demoNavPoint{x, y}) {
		t.Fatal("original rear crossing excluded by the contracted previous bound")
	}
	passedOldBound := false
	for i, q := range n.path {
		passedOldBound = passedOldBound || q.y > 134+176
		if n.touching(q.x, q.y) || i > 0 && !n.clearSegment(n.path[i-1].x, n.path[i-1].y, q) {
			t.Fatal("reopened path crosses original solid arena terrain")
		}
	}
	last := n.path[len(n.path)-1]
	if !passedOldBound || absDemo(last.x-140) > 6 || absDemo(last.y-480) > 6 {
		t.Fatal("route did not use the original rear allowance to reach its clear destination")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("geometric route planning changed source actor, terrain or randomness")
	}
}
