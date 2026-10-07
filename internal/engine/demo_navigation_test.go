package engine

import "testing"

func demoTestSegmentClear(n *demoNavigation, x, y int, to demoNavPoint) bool {
	steps := max(absDemo(to.x-x), absDemo(to.y-y))
	if steps == 0 {
		return !n.touching(x, y)
	}
	for step := 1; step <= steps; step++ {
		if n.touching(x+(to.x-x)*step/steps, y+(to.y-y)*step/steps) {
			return false
		}
	}
	return true
}

func TestDemoNavigationRetainsClearCornerBeforeFarWaypoint(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y, w.ScrollY = 115, 138, 1000
	demoPilotCoverage(w, [2]int{112, 1120})
	n := demoNavigation{practiced: true}
	n.refresh(w)
	n.goal, n.frame = 1110, w.Frame
	n.path = []demoNavPoint{{115, 1138}, {118, 1138}, {121, 1138}, {124, 1138}, {127, 1138}, {130, 1138}, {130, 1135}, {130, 1132}, {130, 1129}}
	for _, point := range n.path {
		if n.touching(point.x, point.y) {
			t.Fatal("fixture path contains a blocked point")
		}
	}
	x, y, found := n.waypoint(w, 1110)
	if !found || !demoTestSegmentClear(&n, 115, 1138, demoNavPoint{x, y}) {
		t.Fatalf("waypoint cut the terrain corner: %d/%d", x, y)
	}
	if x != 130 || y < 1135 || y > 1138 || n.retreat {
		t.Fatalf("the clear first leg changed: %d/%d retreat %v", x, y, n.retreat)
	}
}

func TestDemoScrollMaximumCopiesFirstArenaPrelude(t *testing.T) {
	w := testWorld(t)
	w.Level.Number = 1
	state := NewFirstMiddleState()
	w.FirstMiddle = &state
	w.Player.Y = 176
	before, random, pool := *w.FirstMiddle, w.RandomState(), *w.Pool
	for _, test := range []struct{ camera, maximum, want int }{{3006, 3022, 3344}, {2624, 2640, 3344}, {2623, 2639, 2639}, {3345, 3361, 3361}, {3006, 4000, 4000}} {
		if got := demoScrollMaximum(w, test.camera, test.maximum); got != test.want {
			t.Fatalf("camera %d: source maximum %d, want %d", test.camera, got, test.want)
		}
	}
	if *w.FirstMiddle != before || w.RandomState() != random || *w.Pool != pool {
		t.Fatal("planning advanced live gates, launch seeds, random or actor storage")
	}
	w.Player.Y = 16
	if got := demoScrollMaximum(w, 2624, 2640); got != 2496 || w.FirstMiddle.Crossed {
		t.Fatal("the source shop boundary did not precede arena expansion on the copied state")
	}
	w.FirstMiddle.Crossed = true
	w.Player.Y = 176
	if got := demoScrollMaximum(w, 3006, 3022); got != 3344 {
		t.Fatal("the source arena expansion incorrectly depends on the shop flag")
	}
}

func TestDemoNavigationOriginalCornerUsesClearFirstLegOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	// Observed recording pass 1369. The map and full ship stencil come from
	// the actual level resources; each cached node was individually clear.
	w.ScrollY, w.MaximumScrollY, w.Player.X, w.Player.Y = 3240, 3256, 208, 104
	n := demoNavigation{practiced: true}
	n.refresh(w)
	n.goal, n.frame = 3233, w.Frame
	n.path = []demoNavPoint{{208, 3343}, {208, 3340}, {211, 3337}, {214, 3334}, {217, 3331}, {220, 3328}, {223, 3325}, {226, 3322}, {226, 3319}, {229, 3316}}
	for _, point := range n.path {
		if n.touching(point.x, point.y) {
			t.Fatal("original corner fixture contains a blocked node")
		}
	}
	x, y, found := n.waypoint(w, 3233)
	if !found || !demoTestSegmentClear(&n, 208, 3344, demoNavPoint{x, y}) {
		t.Fatalf("original terrain waypoint skipped its turn: %d/%d", x, y)
	}
}

func TestDemoNavigationOriginalRightPocketFindsSourceRetreatOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	// Recording pass 1990: the post-scroll limit is 3022, but the original
	// first-arena prelude reopens it to 3344 before the next player callback.
	w.ScrollY, w.MaximumScrollY, w.Player.X, w.Player.Y = 3006, 3022, 250, 176
	before, random, player, pool := *w.FirstMiddle, w.RandomState(), w.Player, *w.Pool
	n := demoNavigation{practiced: true}
	x, y, found := n.waypoint(w, 3054)
	if !found || !n.retreat || y <= 3182 || !demoTestSegmentClear(&n, 250, 3182, demoNavPoint{x, y}) {
		t.Fatalf("source rear exit was not admitted: point %d/%d found %v retreat %v", x, y, found, n.retreat)
	}
	rear, left := 3182, 250
	for _, point := range n.path {
		if n.touching(point.x, point.y) || point.y > demoScrollMaximum(w, w.ScrollY, w.MaximumScrollY)+176 {
			t.Fatal("retreat path uses blocked or physically unreachable terrain")
		}
		rear, left = max(rear, point.y), min(left, point.x)
	}
	if rear < 3420 || left > 160 {
		t.Fatalf("route did not traverse the real rear passage and left exit: rear %d left %d", rear, left)
	}
	t.Logf("Source rear route: %d path points, %d search nodes, first %d/%d, rear %d, left %d", len(n.path), len(n.nodes), x, y, rear, left)
	if *w.FirstMiddle != before || w.RandomState() != random || w.Player != player || *w.Pool != pool {
		t.Fatal("finding the source retreat changed gameplay state")
	}
}
