package engine

import "testing"

func originalSecondClosedLeftNodeFixture(t testing.TB) *World {
	t.Helper()
	w := originalSecondNodeContactFixture(t, 192, 176)
	w.damageSecondNode(w.secondNodes[0], 4)
	w.secondStreamsUpdated[0] = false
	if err := w.advanceSecondDefenseWaves(); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range w.Actors {
		if a.Active && a.secondSegment != nil && a.secondSegment.Stream == 0 && a.secondPart.Index == 0 {
			w.damageSecondSegment(a, uint16(a.Health))
			found = true
			break
		}
	}
	if !found || w.secondScheduler.DefenseFlags != 2 {
		t.Fatal("source wave-head removal did not expose the node phase")
	}
	w.advanceSecondNode(w.secondNodes[1])
	return w
}

func TestSecondNodeCrossingGeometry(t *testing.T) {
	w := originalSecondClosedLeftNodeFixture(t)
	n := demoNavigation{practiced: true}
	n.refresh(w)
	before := forecastIsolationDigest(w)
	for _, y := range []int{2560, 2580, 2600, 2620, 2640, 2660, 2680, 2700, 2720} {
		var intervals [][2]int
		start := -1
		for x := 14; x <= 305; x++ {
			clear := x <= 304 && !n.touching(x, y)
			if clear && start < 0 {
				start = x
			}
			if !clear && start >= 0 {
				intervals = append(intervals, [2]int{start, x - 1})
				start = -1
			}
		}
		t.Logf("full-stencil row%d screen%d clear%v", y, y-w.ScrollY, intervals)
	}
	if n.searchPoint(192, 2704, 72, 2568) {
		t.Logf("point route nodes%d length%d start%v end%v", len(n.nodes), len(n.path), n.path[0], n.path[len(n.path)-1])
		for i := 0; i < len(n.path); i += max(1, len(n.path)/12) {
			t.Logf("route%d %+v", i, n.path[i])
		}
	} else {
		t.Logf("NO_POINT_ROUTE nodes%d", len(n.nodes))
	}
	for _, input := range []MotionInput{{Left: true}, {Left: true, Up: true}, {Left: true, Down: true}} {
		s := newDemoMotionForecast(w)
		for pass := 0; pass < 16; pass++ {
			if !s.advance(w, input) {
				t.Logf("held%+v blockedpass%d xy%d,%d camera%d inertia%d", input, pass, s.player.X, s.player.Y, s.scroll.Y, s.player.Inertia)
				break
			}
			if s.player.X < 160 {
				t.Logf("held%+v CROSSED pass%d xy%d,%d camera%d inertia%d", input, pass, s.player.X, s.player.Y, s.scroll.Y, s.player.Inertia)
				break
			}
		}
	}
	if before != forecastIsolationDigest(w) {
		t.Fatal("geometry query changed original world")
	}
}

func TestSecondNodeCrossingMotionAndNativeGate(t *testing.T) {
	w := originalSecondClosedLeftNodeFixture(t)
	before := forecastIsolationDigest(w)
	s := newDemoMotionForecast(w)
	passes := 0
	for leg, stage := range []struct {
		motion MotionInput
		done   func(demoMotionForecast) bool
	}{
		{MotionInput{Up: true, Right: true}, func(s demoMotionForecast) bool { return s.player.X == 304 }},
		{MotionInput{Up: true}, func(s demoMotionForecast) bool { return s.player.Y <= 41 }},
		{MotionInput{Left: true}, func(s demoMotionForecast) bool { return s.player.X == 14 }},
		{MotionInput{Down: true}, func(s demoMotionForecast) bool { return s.player.Y >= 136 }},
		{MotionInput{Right: true}, func(s demoMotionForecast) bool { return s.player.X >= 68 }},
	} {
		for count := 0; count < 40 && !stage.done(s); count++ {
			if !s.advance(w, stage.motion) || s.rewind.Timer != 0 {
				t.Fatalf("source terrain motion blocked leg%d pass%d", leg, passes)
			}
			passes++
		}
		if !stage.done(s) {
			t.Fatalf("bounded native leg%d failed", leg)
		}
		t.Logf("motion-only leg%d at%d,%d/c%d passes%d", leg, s.player.X, s.player.Y, s.scroll.Y, passes)
	}
	if s.player.X != 68 || s.player.Y != 136 || s.scroll.Y != 2528 {
		t.Fatalf("source detour endpoint differs: %+v", s)
	}
	node := *w.secondNodes[1].secondNode
	gates := w.secondGateCounters
	node.Advance(SecondDefenseNodeInput{Frame: w.Frame + uint64(passes), PlayerX: s.player.X, ScrollY: s.scroll.Y, MaximumScrollY: s.scroll.Maximum, Remaining: w.secondDefenseRemaining, DefenseFlags: 2}, &gates)
	if node.Collision != (CollisionRect{Left: 64, Top: 96, Right: 79, Bottom: 111}) {
		t.Fatalf("source left-half gate did not publish its collider: %+v", node)
	}
	if before != forecastIsolationDigest(w) {
		t.Fatal("route proof modified the original live world")
	}
	// Isolate the ordinary weapon phase at the proven endpoint. This is not
	// a live-wave victory: the full World.Step detour below records that risk.
	var shotFixture WorldForecast
	if err := shotFixture.Load(w); err != nil {
		t.Fatal(err)
	}
	a := shotFixture.State()
	a.Player = s.player
	a.advanceSecondNode(a.secondNodes[1])
	if !presentationShotOpportunityForMotion(a, MotionInput{}) {
		t.Fatal("left firing lane lacks ordinary shot opportunity")
	}
	oldHealth := a.secondNodes[1].Health
	if err := a.Weapons.AdvanceEquipment(a.weaponContext(Input{Fire: true}, true)); err != nil {
		t.Fatal(err)
	}
	for range 6 {
		if err := a.Weapons.AdvanceProjectiles(a.weaponContext(Input{}, false)); err != nil {
			t.Fatal(err)
		}
	}
	if a.secondNodes[1].Health >= oldHealth {
		t.Fatal("ordinary gun projectile did not damage the open left node")
	}
	t.Logf("source ordinary projectile changed left health%d to%d", oldHealth, a.secondNodes[1].Health)
}

func TestSecondCrossingLiveStreamLimit(t *testing.T) {
	w := originalSecondClosedLeftNodeFixture(t)
	before := forecastIsolationDigest(w)
	var f WorldForecast
	if err := f.Load(w); err != nil {
		t.Fatal(err)
	}
	s := f.State()
	for pass := 1; pass <= 60; pass++ {
		input := MotionInput{Left: true}
		if pass <= 13 {
			input = MotionInput{Up: true, Right: true}
		} else if pass <= 27 {
			input = MotionInput{Up: true}
		}
		for range 3 {
			f.AdvancePALTick()
		}
		r, err := f.Advance(Input{Motion: input})
		if err != nil {
			t.Fatal(err)
		}
		if s.Rewind.Timer != 0 || s.Coverage.Touches(s.Player.X, s.Player.Y, s.ScrollY, *s.Level.PlayerStencil) {
			t.Fatal("live detour touched terrain")
		}
		if pass == 27 && (s.Player.X != 304 || s.Player.Y != 41 || r.Shield != 39) {
			t.Fatalf("source right-upper approach differs: %+v", r)
		}
		if !r.Alive {
			if pass != 49 || s.Player.X != 115 || s.Player.Y != 41 {
				t.Fatalf("captured isolated stream limit differs pass%d %+v", pass, r)
			}
			t.Logf("LIVE_STREAM_LIMIT pass%d xy%d,%d shield%d flags%d; terrain route remains clear", pass, s.Player.X, s.Player.Y, r.Shield, s.secondScheduler.DefenseFlags)
			if before != forecastIsolationDigest(w) {
				t.Fatal("live detour probe mutated source")
			}
			return
		}
	}
	t.Fatal("isolated source stream limit was not reproduced")
}
