package engine

import (
	"reflect"
	"testing"
)

// Expand every recorded parent edge with the original forecast. This verifies
// both compact node order and the exact history that each parent path implies.
func compactMaterializedNodes(t testing.TB, w *World, p *nativeMotionPlanner) []nativeMotionNode {
	t.Helper()
	if len(p.compactNodes) == 0 {
		return p.nodes
	}
	nodes := make([]nativeMotionNode, len(p.compactNodes))
	start := newDemoMotionForecast(w)
	for index, node := range p.compactNodes {
		state := start
		if index != 0 {
			if node.parent < 0 || node.parent >= index || node.action < 0 || node.action >= len(demoDirections) {
				t.Fatalf("invalid compact parent at%d", index)
			}
			state = nodes[node.parent].state
			if !state.advance(w, demoDirections[node.action]) || state.rewind.Timer != 0 {
				t.Fatalf("compact node%d has a rejected source edge", index)
			}
		}
		if state.player != node.state.player || state.scroll != node.state.scroll {
			t.Fatalf("compact node%d lost motion/camera state", index)
		}
		nodes[index] = nativeMotionNode{state: state, parent: node.parent, action: node.action, depth: node.depth, priority: node.priority, remaining: node.remaining, distance: node.distance}
	}
	return nodes
}

func assertCompactSearchEqual(t testing.TB, w *World, x, worldY int, compact bool) nativeMotionPlanner {
	t.Helper()
	before := forecastDigest(w)
	var got, want nativeMotionPlanner
	accepted, expected := got.search(w, x, worldY), want.searchFull(w, x, worldY)
	if (len(got.compactNodes) != 0) != compact {
		t.Fatalf("wrong search scope: compact%v want%v", len(got.compactNodes) != 0, compact)
	}
	if accepted != expected || got.expanded != want.expanded || !reflect.DeepEqual(compactMaterializedNodes(t, w, &got), want.nodes) || !reflect.DeepEqual(got.queue, want.queue) || !reflect.DeepEqual(got.visited, want.visited) || !reflect.DeepEqual(got.commands, want.commands) || !reflect.DeepEqual(got.states, want.states) {
		t.Fatalf("compact search differs: found%v/%v expanded%d/%d nodes%d/%d", accepted, expected, got.expanded, want.expanded, got.nodeCount(), len(want.nodes))
	}
	if forecastDigest(w) != before {
		t.Fatal("compact search changed live source state")
	}
	t.Logf("found%v expanded%d nodes%d commands%d", accepted, got.expanded, got.nodeCount(), len(got.commands))
	return got
}

func TestNativeMotionCompactFourthOriginalLegsOptional(t *testing.T) {
	for _, fixture := range []struct {
		name                 string
		x, y, camera, tx, ty int
	}{
		{"before-fork", 147, 151, 4128, 123, 4255},
		{"last-shared-branch", 211, 176, 4087, 187, 4239},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			w := fourthOpeningSourceFixture(t, fixture.camera, fixture.x, fixture.y, 0)
			assertCompactSearchEqual(t, w, fixture.tx, fixture.ty, true)
		})
	}
}

func TestNativeMotionCompactFallbackRetainsFullHistoryOptional(t *testing.T) {
	for _, fixture := range []struct {
		name   string
		modify func(*World)
	}{
		{"positive-rewind", func(w *World) { w.Rewind.Timer = 1 }},
		{"negative-rewind", func(w *World) { w.Rewind.Timer = -1 }},
		{"crush-limit", func(w *World) { w.Rewind.Timer = -17 }},
		{"dive", func(w *World) { w.Dive.Phase = 1 }},
		{"covered", func(w *World) {
			for y := 16; y <= 176; y += 4 {
				for x := 14; x <= 304; x += 2 {
					if w.Coverage.Touches(x, y, w.ScrollY, *w.Level.PlayerStencil) {
						w.Player.X, w.Player.Y = x, y
						return
					}
				}
			}
			t.Fatal("original map lacks a covered pose")
		}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			w := nativeMotionFixture(t, 193, 176, 1896)
			fixture.modify(w)
			assertCompactSearchEqual(t, w, 202, 2077, false)
		})
	}
	w := forecastOriginalScene(t, 2, false)
	assertCompactSearchEqual(t, w, w.Player.X, w.Player.Y+w.ScrollY-1, false)
}

func TestNativeMotionCompactTerminalKeepsActualHistoryOptional(t *testing.T) {
	w := nativeMotionFixture(t, 255, 16, 1836)
	w.MaterializationFrames = 0
	n := demoNavigation{practiced: true}
	n.refresh(w)
	var p nativeMotionPlanner
	_, ok := p.commandClearTerminal(w, &n, 256, 1832)
	if !ok || len(p.commands) != 20 {
		t.Fatal("original20-command terminal plan changed")
	}
	for index, input := range p.commands {
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(Input{Motion: input}); err != nil {
			t.Fatal(err)
		}
		if p.states[index+1].rewind != w.Rewind {
			t.Fatalf("terminal history changed at%d", index)
		}
	}
}

func BenchmarkNativeMotionCompactOriginalRoutes(b *testing.B) {
	for _, fixture := range []struct {
		name                 string
		x, y, camera, tx, ty int
		found                bool
	}{
		{"rear-corner", 193, 176, 1896, 202, 2077, true},
		{"three-pixel-lane", 251, 176, 1732, 254, 1908, true},
		{"logged-nine-thousand", 66, 21, 1722, 64, 1624, false},
	} {
		b.Run(fixture.name, func(b *testing.B) {
			w := nativeMotionFixture(b, fixture.x, fixture.y, fixture.camera)
			for _, compact := range []bool{false, true} {
				name := "full"
				if compact {
					name = "compact"
				}
				b.Run(name, func(b *testing.B) {
					var p nativeMotionPlanner
					search := p.searchFull
					if compact {
						search = p.search
					}
					if search(w, fixture.tx, fixture.ty) != fixture.found {
						b.Fatal("fixture result changed")
					}
					b.ReportAllocs()
					b.ResetTimer()
					for b.Loop() {
						if search(w, fixture.tx, fixture.ty) != fixture.found {
							b.Fatal("search result changed")
						}
					}
					b.ReportMetric(float64(p.expanded), "expanded/op")
					b.ReportMetric(float64(p.nodeCount()), "states/op")
				})
			}
		})
	}
}

// evaluate copies source policy settings but must retain each worker's private
// mutable planner storage, including the newly compact node backing array.
func TestNativeMotionCompactMiddleWorkerOwnsBackingOptional(t *testing.T) {
	corridor := nativeMotionFixture(t, 193, 176, 1896)
	var source DemoPilot
	var worker thirdMiddleForecastWorker
	if !source.nativeMotion.search(corridor, 202, 2077) || !worker.policy.nativeMotion.search(corridor, 202, 2077) {
		t.Fatal("original compact route missing")
	}
	sourceBacking := &source.nativeMotion.compactNodes[0]
	workerBacking := &worker.policy.nativeMotion.compactNodes[0]
	sourceFirst := *sourceBacking
	if sourceBacking == workerBacking {
		t.Fatal("separate planners share compact storage")
	}
	middle := expertMiddleForecastScene(t)
	worker.evaluate(middle, MotionInput{}, Input{}, 3, source)
	if worker.err != nil || len(worker.policy.nativeMotion.compactNodes) == 0 || &worker.policy.nativeMotion.compactNodes[0] != workerBacking || &source.nativeMotion.compactNodes[0] != sourceBacking {
		t.Fatal("middle evaluation replaced worker compact storage with source storage")
	}
	worker.policy.nativeMotion.compactNodes[0].state.player.X++
	if *sourceBacking != sourceFirst {
		t.Fatal("worker compact mutation escaped into source planner")
	}
}
