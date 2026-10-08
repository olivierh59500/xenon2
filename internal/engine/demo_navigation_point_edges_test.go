package engine

import "testing"

func pointNavigationThinObstacle(w *World) {
	demoPilotCoverage(w)
	var rows [16]uint16
	rows[1] = 0x4000 // One covered pixel at x161/world1121.
	w.Coverage.coverage[2] = rows
}

func assertPointNavigationEdgesClear(t *testing.T, navigation *demoNavigation) {
	t.Helper()
	if len(navigation.path) < 2 {
		t.Fatal("fixture did not produce a traversable geometric route")
	}
	for index, point := range navigation.path {
		if navigation.touching(point.x, point.y) {
			t.Fatalf("covered route node%d: %+v", index, point)
		}
		if index > 0 && !navigation.clearSegment(navigation.path[index-1].x, navigation.path[index-1].y, point) {
			t.Fatalf("covered route edge%d: %+v to %+v", index, navigation.path[index-1], point)
		}
	}
}

func TestPointNavigationOriginalMaskedGapChecksWholeEdgeOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	navigation := demoNavigation{practiced: true}
	navigation.refresh(w)
	from, to := demoNavPoint{251, 805}, demoNavPoint{254, 808}
	if navigation.touching(from.x, from.y) || navigation.touching(to.x, to.y) {
		t.Fatal("observed original gap does not have two clear endpoint nodes")
	}
	if navigation.clearSegment(from.x, from.y, to) {
		t.Fatal("the original masked edge no longer demonstrates the interior collision")
	}
	w.Player.X, w.Player.Y, w.ScrollY = from.x, 176, from.y-176
	w.MinimumScrollY, w.MaximumScrollY = 0, w.ScrollY+16
	if _, _, found := navigation.pointWaypoint(w, 263, 817); !found {
		t.Fatal("point search did not route around the original masked edge")
	}
	assertPointNavigationEdgesClear(t, &navigation)
}

func TestPointNavigationFreshSearchRejectsCoveredInterior(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y, w.ScrollY = 160, 120, 1000
	pointNavigationThinObstacle(w)
	w.Coverage.Map[(1121/16)*20+161/16] = 2
	navigation := demoNavigation{practiced: true}
	before, random := w.Player, w.RandomState()
	if _, _, found := navigation.pointWaypoint(w, 175, 1135); !found {
		t.Fatal("one interior pixel prevented an otherwise reachable point route")
	}
	assertPointNavigationEdgesClear(t, &navigation)
	if w.Player != before || w.RandomState() != random {
		t.Fatal("edge-aware point search changed the real player or random stream")
	}
}

func TestPointNavigationChangedTileInvalidatesEdgeWithClearEndpoints(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y, w.ScrollY = 160, 120, 1000
	pointNavigationThinObstacle(w)
	navigation := demoNavigation{practiced: true}
	navigation.refresh(w)
	navigation.goal, navigation.pointTargetX = 1135, 175
	navigation.path = []demoNavPoint{{160, 1120}, {163, 1123}, {166, 1126}, {169, 1129}, {172, 1132}, {175, 1135}}
	for index := 1; index < len(navigation.path); index++ {
		if !navigation.clearSegment(navigation.path[index-1].x, navigation.path[index-1].y, navigation.path[index]) {
			t.Fatal("cached fixture contained an invalid edge before its tile changed")
		}
	}
	w.Frame++
	w.Coverage.Map[(1121/16)*20+161/16] = 2
	if _, _, found := navigation.pointWaypoint(w, 175, 1135); !found {
		t.Fatal("changed interior pixel did not leave a reachable detour")
	}
	// Every old endpoint remains clear. Only the old first edge became
	// covered, so node-only validation would incorrectly retain the cache.
	for _, point := range []demoNavPoint{{160, 1120}, {163, 1123}, {166, 1126}, {169, 1129}, {172, 1132}, {175, 1135}} {
		if navigation.touching(point.x, point.y) {
			t.Fatal("the mutable edge fixture also changed endpoint coverage")
		}
	}
	if navigation.frame != w.Frame {
		t.Fatal("covered edge did not cause the cached point route to be rebuilt")
	}
	assertPointNavigationEdgesClear(t, &navigation)
}
