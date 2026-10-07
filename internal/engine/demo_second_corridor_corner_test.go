package engine

import "testing"

func TestPracticedSecondCorridorWaypointRetainsOriginalRearCornerOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY, w.Player.X, w.Player.Y = 1189, 1204, 174, 176
	p := DemoPilot{practicedRoute: true, navigation: &demoNavigation{}}
	n := p.navigation
	n.refresh(w)
	n.goal, n.frame = 1072, w.Frame
	if !n.searchSecondFinal(174, 1365, 1072) || len(n.path) < 9 {
		t.Fatal("original corridor has no bounded route")
	}
	for _, point := range n.path {
		if n.touching(point.x, point.y) {
			t.Fatal("original rear-corner fixture contains a blocked source node")
		}
	}
	if demoTestSegmentClear(n, 174, 1365, n.path[8]) {
		t.Fatal("original fixture no longer exercises the blocked waypoint shortcut")
	}
	// The full source stencil needs a short rearward leg before the right turn.
	// Looking eight nodes ahead skips that leg and crosses occupied terrain.
	x, y, found := p.secondFinalWaypoint(w, 1072)
	if !found || !demoTestSegmentClear(n, 174, 1365, demoNavPoint{x, y}) || y <= 1365 {
		t.Fatalf("practiced corridor cut its original rear corner: %d/%d found%v", x, y, found)
	}
}
