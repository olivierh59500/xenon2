package engine

import (
	"testing"
)

func TestSecondCrossingObjectiveRetainsClearUpperDetour(t *testing.T) {
	w := originalSecondClosedLeftNodeFixture(t)
	ordinary := &demoNavigation{goal: 42, frame: 77, path: []demoNavPoint{{19, 2710}, {22, 2707}}}
	p := DemoPilot{practicedRoute: true, navigation: ordinary}
	before := forecastIsolationDigest(w)
	x, y, ok := secondArenaCrossingGoal(w, &p)
	if !ok || x <= w.Player.X || y >= w.Player.Y {
		t.Fatalf("opposite closed node did not first approach the right-upper route: %d,%d/%v", x, y, ok)
	}
	n := p.secondCrossingNavigation
	assertPointNavigationEdgesClear(t, n)
	minimum := 4800
	for _, point := range n.path {
		minimum = min(minimum, point.y)
	}
	if minimum >= 2616 || n.goal != 2664 || n.pointTargetX != 72 {
		t.Fatalf("route did not pass above the source structure and retain the left firing lane: minimum%d goal%d,%d", minimum, n.pointTargetX, n.goal)
	}
	if forecastIsolationDigest(w) != before || p.navigation != ordinary || ordinary.goal != 42 || ordinary.frame != 77 || len(ordinary.path) != 2 {
		t.Fatal("crossing planning changed the live game or ordinary navigation cache")
	}
	// The node opens at the halfway line, while its lower firing lane still
	// requires the retained upper route and descent around the left structure.
	w.Player.X, w.Player.Y = 147, 41
	w.advanceSecondNode(w.secondNodes[1])
	if w.secondNodes[1].Collision.Empty() {
		t.Fatal("native left-half eligibility did not open the node")
	}
	before = forecastIsolationDigest(w)
	if _, _, ok := secondArenaCrossingGoal(w, &p); !ok || p.secondCrossingNavigation != n || n.goal != 2664 {
		t.Fatal("opening the node discarded the still-needed descent")
	}
	if before != forecastIsolationDigest(w) {
		t.Fatal("retaining the route changed native state")
	}
	w.Player.X, w.Player.Y = 72, 136
	if _, _, ok := secondArenaCrossingGoal(w, &p); ok || len(n.path) != 0 {
		t.Fatal("reached source firing lane did not release crossing objective")
	}
}

func TestSecondCrossingScopeAndSameSideAreUnchanged(t *testing.T) {
	w := originalSecondClosedLeftNodeFixture(t)
	for _, mode := range []string{"ready", "dead", "gameover", "released", "rewind", "otherlevel", "outside", "disabled", "shielded"} {
		t.Run(mode, func(t *testing.T) {
			s := *w
			p := DemoPilot{}
			scheduler := *w.secondScheduler
			s.secondScheduler = &scheduler
			switch mode {
			case "ready":
				s.Ready = true
			case "dead":
				s.PlayerAlive = false
			case "gameover":
				s.GameOver = true
			case "released":
				s.secondMiddleReleased = true
			case "rewind":
				s.Rewind.Timer = 1
			case "otherlevel":
				s.Level.Number = 3
			case "outside":
				s.ScrollY = 3000
			case "disabled":
				p.Config.DisableBossAlignment = true
			case "shielded":
				scheduler.DefenseFlags = 3
			}
			before := forecastIsolationDigest(&s)
			if _, _, ok := secondArenaCrossingGoal(&s, &p); ok || p.secondCrossingNavigation != nil {
				t.Fatal("ineligible crossing changed the existing controller")
			}
			if before != forecastIsolationDigest(&s) {
				t.Fatal("scope check changed source")
			}
		})
	}
	w.Player.X = 72
	p := DemoPilot{}
	if _, _, ok := secondArenaCrossingGoal(w, &p); ok || p.secondCrossingNavigation != nil {
		t.Fatal("same-side node created an unnecessary route")
	}
}
