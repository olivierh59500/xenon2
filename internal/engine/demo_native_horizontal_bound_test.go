package engine

import "testing"

func TestNativeHorizontalReverseGraphContainsEverySourceTransition(t *testing.T) {
	for speed := 0; speed <= 2; speed++ {
		table := newNativeHorizontalTable(speed)
		if int(table.starts[nativeHorizontalStates]) != nativeHorizontalEdges {
			t.Fatal("reverse graph lost a native horizontal transition")
		}
		// Compare edge multiplicities as well as endpoints; clamps can send
		// multiple physical states to a shared successor.
		type edge struct{ before, after int }
		edges := make(map[edge]int, nativeHorizontalEdges)
		for after := range nativeHorizontalStates {
			for _, before := range table.predecessors[table.starts[after]:table.starts[after+1]] {
				edges[edge{int(before), after}]++
			}
		}
		for x := 14; x <= 304; x++ {
			for inertia := -6; inertia <= 6; inertia++ {
				for _, input := range []MotionInput{{Left: true}, {}, {Right: true}} {
					state := PlayerMotionState{X: x, Y: 176, Inertia: inertia, SpeedTier: speed}
					// Simultaneous vertical controls must not alter this relaxed
					// horizontal machine, including either vertical overshoot.
					input.Up, input.Down = true, true
					state.Advance(input, MotionContext{ScrollY: 1000, VisitedScrollY: 1016, BaseScrollStep: 1})
					key := edge{(x-14)*13 + inertia + 6, (state.X-14)*13 + state.Inertia + 6}
					if edges[key] == 0 {
						t.Fatalf("missing source edge: speed%d x%d inertia%d input%+v next%+v", speed, x, inertia, input, state)
					}
					edges[key]--
				}
			}
		}
		for key, count := range edges {
			if count != 0 {
				t.Fatalf("extra reverse edge %+v count%d", key, count)
			}
		}
	}
}

func TestNativeHorizontalDistancesSatisfyEverySourceBellmanEquation(t *testing.T) {
	var bound nativeHorizontalBound
	cases, maximum := 0, 0
	for speed := 0; speed <= 2; speed++ {
		for targetX := 14; targetX <= 304; targetX++ {
			for x := 14; x <= 304; x++ {
				for inertia := -6; inertia <= 6; inertia++ {
					distance, reachable := bound.distance(x, inertia, targetX, speed)
					if !reachable {
						t.Fatalf("legal horizontal state is unreachable: speed%d x%d inertia%d target%d", speed, x, inertia, targetX)
					}
					want := int(nativeHorizontalUnreached)
					if x == targetX {
						want = 0
					} else {
						for _, input := range []MotionInput{{Left: true}, {}, {Right: true}} {
							state := PlayerMotionState{X: x, Y: 100, Inertia: inertia, SpeedTier: speed}
							state.Advance(input, MotionContext{BaseScrollStep: 1})
							next, found := bound.distance(state.X, state.Inertia, targetX, speed)
							if found {
								want = min(want, next+1)
							}
						}
					}
					if distance != want {
						t.Fatalf("source shortest-path equation: speed%d x%d inertia%d target%d distance%d expected%d", speed, x, inertia, targetX, distance, want)
					}
					maximum = max(maximum, distance)
					cases++
				}
			}
		}
	}
	if cases != 3*291*291*13 || maximum >= nativeHorizontalStates {
		t.Fatal("incomplete state-space proof or finite distance exceeds a simple graph path")
	}
	t.Logf("%d exact source-distance equations; maximum%d passes", cases, maximum)
}

func TestNativeHorizontalThreePixelAlignmentNeedsSevenActualPasses(t *testing.T) {
	var bound nativeHorizontalBound
	if distance, found := bound.distance(251, 0, 254, 2); !found || distance != 7 {
		t.Fatalf("three-pixel source lane: distance%d reachable%v", distance, found)
	}
	type horizontal struct{ x, inertia int }
	frontier := map[horizontal]bool{{251, 0}: true}
	for pass := 1; pass <= 7; pass++ {
		next := make(map[horizontal]bool)
		for before := range frontier {
			for _, input := range []MotionInput{{Left: true}, {}, {Right: true}} {
				state := PlayerMotionState{X: before.x, Y: 176, Inertia: before.inertia, SpeedTier: 2}
				state.Advance(input, MotionContext{BaseScrollStep: 1})
				next[horizontal{state.X, state.Inertia}] = true
			}
		}
		found := false
		for state := range next {
			found = found || state.x == 254
		}
		if found != (pass == 7) {
			t.Fatalf("actual callback first reaches three-pixel lane at pass%d: %v", pass, found)
		}
		frontier = next
	}
	// A source clamp changes reachability: the same three-pixel displacement
	// at the right edge takes one held command and retains inertia one.
	if distance, found := bound.distance(301, 0, 304, 2); !found || distance != 1 {
		t.Fatal("right clamp was treated like unbounded horizontal travel")
	}
}

func TestNativeHorizontalCacheReusesTargetsAndRejectsUnsupportedStates(t *testing.T) {
	var bound nativeHorizontalBound
	for _, input := range [][4]int{{13, 0, 14, 0}, {305, 0, 304, 0}, {14, -7, 14, 0}, {14, 7, 14, 0}, {14, 0, 13, 0}, {14, 0, 305, 0}, {14, 0, 14, -1}, {14, 0, 14, 3}} {
		if distance, found := bound.distance(input[0], input[1], input[2], input[3]); found || distance != 0 {
			t.Fatal("unsupported source state produced a travel guarantee")
		}
	}
	if bound.speeds != [3]*nativeHorizontalTable{} {
		t.Fatal("unsupported queries allocated speed graphs")
	}
	bound.distance(251, 0, 254, 2)
	table := bound.speeds[2]
	call := 0
	if allocations := testing.AllocsPerRun(100, func() {
		bound.distance(251, 0, 254+call%2, 2)
		call++
	}); allocations != 0 {
		t.Fatalf("cached source queries allocate %v times", allocations)
	}
	if bound.speeds[2] != table || bound.speeds[0] != nil || bound.speeds[1] != nil {
		t.Fatal("target changes rebuilt the graph or initialized unused speeds")
	}
}
