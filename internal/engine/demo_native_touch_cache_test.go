package engine

import (
	"math/bits"
	"reflect"
	"testing"
)

func (p *nativeMotionPlanner) searchWithoutTouchCache(w *World, x, worldY int) bool {
	p.compactNodes = p.compactNodes[:0]
	if !nativeMotionSupported(w) || w.Dive.Phase != 0 || w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		return p.searchFull(w, x, worldY)
	}
	p.nodes, p.queue = p.nodes[:0], p.queue[:0]
	p.commands, p.states, p.expanded = p.commands[:0], p.states[:0], 0
	if p.visited == nil {
		p.visited = make(map[nativeMotionKey]int)
	} else {
		clear(p.visited)
	}
	start := newDemoMotionForecast(w)
	h := nativeMotionTimeBound(start, x, worldY, w.Equipment.SpeedTier)
	p.compactNodes = append(p.compactNodes, nativeCompactMotionNode{state: nativeCompactMotionState{start.player, start.scroll}, parent: -1, action: -1, priority: h, remaining: h, distance: absDemo(start.player.X-x) + absDemo(start.player.Y+start.scroll.Y-worldY)})
	p.visited[nativeMotionStateKey(start)] = 0
	p.compactPush(0)
	// Timer stays zero for every retained node. The scratch history is not
	// consulted; only canonical replay below publishes a retained history.
	next := start
	for len(p.queue) != 0 && p.expanded < nativeMotionNodeBudget {
		index := p.compactPop()
		current := p.compactNodes[index]
		key := nativeMotionKey{current.state.player.X, current.state.player.Y, current.state.player.Inertia, current.state.scroll.Y, current.state.scroll.Maximum, current.state.scroll.DeviationPasses}
		if p.visited[key] < current.depth {
			continue
		}
		p.expanded++
		if current.state.player.X == x && current.state.player.Y+current.state.scroll.Y == worldY {
			var path [nativeMotionDepthLimit + 1]int
			count := 0
			for at := index; at >= 0; at = p.compactNodes[at].parent {
				path[count], count = at, count+1
				if p.compactNodes[at].action >= 0 {
					p.commands = append(p.commands, demoDirections[p.compactNodes[at].action])
				}
			}
			for i, j := 0, len(p.commands)-1; i < j; i, j = i+1, j-1 {
				p.commands[i], p.commands[j] = p.commands[j], p.commands[i]
			}
			state := start
			if p.compactNodes[path[count-1]].state != (nativeCompactMotionState{state.player, state.scroll}) {
				p.compactNodes = p.compactNodes[:0]
				return p.searchFull(w, x, worldY)
			}
			p.states = append(p.states, state)
			for step, input := range p.commands {
				expected := p.compactNodes[path[count-2-step]].state
				if !state.advance(w, input) || state.rewind.Timer != 0 || state.player != expected.player || state.scroll != expected.scroll {
					p.compactNodes = p.compactNodes[:0]
					return p.searchFull(w, x, worldY)
				}
				p.states = append(p.states, state)
			}
			return len(p.commands) != 0
		}
		if current.depth == nativeMotionDepthLimit {
			continue
		}
		for action, input := range demoDirections {
			next.player, next.scroll = current.state.player, current.state.scroll
			next.rewind.Timer = 0
			if !next.advance(w, input) || next.rewind.Timer != 0 {
				continue
			}
			depth, key := current.depth+1, nativeMotionStateKey(next)
			if previous, ok := p.visited[key]; ok && previous <= depth {
				continue
			}
			p.visited[key] = depth
			h := nativeMotionTimeBound(next, x, worldY, w.Equipment.SpeedTier)
			p.compactNodes = append(p.compactNodes, nativeCompactMotionNode{state: nativeCompactMotionState{next.player, next.scroll}, parent: index, action: action, depth: depth, priority: depth + h, remaining: h, distance: absDemo(next.player.X-x) + absDemo(next.player.Y+next.scroll.Y-worldY)})
			p.compactPush(len(p.compactNodes) - 1)
		}
	}
	return false
}

func assertNativeTouchCacheSearchEqual(t testing.TB, w *World, p *nativeMotionPlanner, x, worldY int, reusedBaseline ...*nativeMotionPlanner) bool {
	t.Helper()
	before := forecastDigest(w)
	var freshBaseline nativeMotionPlanner
	want := &freshBaseline
	if len(reusedBaseline) != 0 {
		want = reusedBaseline[0]
	}
	got, expected := p.search(w, x, worldY), want.searchWithoutTouchCache(w, x, worldY)
	if got != expected || p.expanded != want.expanded || !reflect.DeepEqual(p.compactNodes, want.compactNodes) || !reflect.DeepEqual(p.nodes, want.nodes) || !reflect.DeepEqual(p.queue, want.queue) || !reflect.DeepEqual(p.visited, want.visited) || !reflect.DeepEqual(p.commands, want.commands) || !reflect.DeepEqual(p.states, want.states) {
		t.Fatalf("cached search changed result/order: found%v/%v expanded%d/%d nodes%d/%d", got, expected, p.expanded, want.expanded, p.nodeCount(), want.nodeCount())
	}
	if forecastDigest(w) != before {
		t.Fatal("cached search changed source world")
	}
	known := 0
	if p.touchCache != nil {
		for _, row := range p.touchCache.known {
			for _, word := range row {
				known += bits.OnesCount32(word)
			}
		}
	}
	t.Logf("found%v expanded%d nodes%d commands%d uniqueMemoPoints%d", got, p.expanded, p.nodeCount(), len(p.commands), known)
	return got
}

func TestNativeTouchCachePreservesExactOriginalSearchOptional(t *testing.T) {
	for _, fixture := range []struct {
		name                 string
		x, y, camera, tx, ty int
	}{
		{"rear-corner", 193, 176, 1896, 202, 2077},
		{"three-pixel-lane", 251, 176, 1732, 254, 1908},
		{"logged-nine-thousand", 66, 21, 1722, 64, 1624},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			w := nativeMotionFixture(t, fixture.x, fixture.y, fixture.camera)
			var p nativeMotionPlanner
			assertNativeTouchCacheSearchEqual(t, w, &p, fixture.tx, fixture.ty)
		})
	}
	for _, fixture := range []struct {
		name                 string
		x, y, camera, tx, ty int
	}{
		{"fourth-before-fork", 147, 151, 4128, 123, 4255},
		{"fourth-shared-branch", 211, 176, 4087, 187, 4239},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			w := fourthOpeningSourceFixture(t, fixture.camera, fixture.x, fixture.y, 0)
			var p nativeMotionPlanner
			assertNativeTouchCacheSearchEqual(t, w, &p, fixture.tx, fixture.ty)
		})
	}
}

func TestNativeTouchCacheMatchesAllMotionQueriesOptional(t *testing.T) {
	cases := 0
	for level := 1; level <= 5; level++ {
		w := forecastOriginalScene(t, level, false)
		before := forecastDigest(w)
		camera := w.ScrollY
		var poses [2]ShipHistory
		var found [2]bool
		for y := 16; y <= 176 && (!found[0] || !found[1]); y += 8 {
			for x := 14; x <= 304; x += 2 {
				index := 0
				if w.Coverage.Touches(x, y, camera, *w.Level.PlayerStencil) {
					index = 1
				}
				if !found[index] {
					poses[index], found[index] = ShipHistory{camera, x, y}, true
				}
			}
		}
		if !found[0] || !found[1] {
			t.Fatal("original clear/covered poses missing")
		}
		var cache nativeMotionTouchCache
		for _, pose := range poses {
			for _, timer := range []int{-17, -1, 0, 1} {
				for _, input := range demoDirections {
					got := newDemoMotionForecast(w)
					got.player.X, got.player.Y = pose.X, pose.Y
					got.rewind = NewTerrainRewind(camera, pose.X, pose.Y)
					got.rewind.Timer = timer
					got.rewind.History[1] = ShipHistory{camera + 2, pose.X + 1, pose.Y - 1}
					want := got
					cache.reset(pose.Y + camera - 224)
					for pass := 0; pass < 3; pass++ {
						a, b := got.advanceWithTouchCache(w, input, &cache), want.advance(w, input)
						if a != b || got != want {
							t.Fatalf("cache changed complete motion state level%d timer%d pass%d input%+v", level, timer, pass, input)
						}
					}
					cases++
				}
			}
		}
		if forecastDigest(w) != before {
			t.Fatal("motion queries changed source world")
		}
		// Camera/screen decomposition is absent from the exact world-point key.
		pose := poses[0]
		cache.reset(pose.Y + camera - 224)
		a := cache.touches(w, pose.X, pose.Y, camera)
		b := cache.touches(w, pose.X, pose.Y+7, camera-7)
		if a != b || a != w.Coverage.Touches(pose.X, pose.Y, camera, *w.Level.PlayerStencil) {
			t.Fatal("same world point changed with camera decomposition")
		}
		for _, row := range []int{-1, 320} {
			y := cache.top + row - camera
			if cache.touches(w, pose.X, y, camera) != w.Coverage.Touches(pose.X, y, camera, *w.Level.PlayerStencil) {
				t.Fatal("outside-window fallback differs")
			}
		}
	}
	if cases != 360 {
		t.Fatal("incomplete cached motion query coverage")
	}
}

func TestNativeTouchCacheResetsAfterActualTilePatchOptional(t *testing.T) {
	w := nativeMotionFixture(t, 193, 176, 1896)
	var p nativeMotionPlanner
	if !assertNativeTouchCacheSearchEqual(t, w, &p, 202, 2077) {
		t.Fatal("original route missing")
	}
	// Both implementations retain their original successful slices before
	// the map changes; this also compares empty reused outputs exactly.
	var uncached nativeMotionPlanner
	if !uncached.searchWithoutTouchCache(w, 202, 2077) {
		t.Fatal("uncached original route missing")
	}
	endpoint := p.states[len(p.states)-1]
	if p.touchCache.touches(w, endpoint.player.X, endpoint.player.Y, endpoint.scroll.Y) {
		t.Fatal("clear endpoint was covered")
	}
	var opaque uint16
	for _, tile := range w.Level.Terrain.Tiles {
		if !tile.Masked {
			opaque = tile.ID
			break
		}
	}
	if opaque == 0 {
		t.Fatal("original opaque tile missing")
	}
	stencil := w.Level.PlayerStencil
	patched := false
	// Choose an original tile cell touched by the goal stencil but not the
	// initial stencil, so the next attempt exercises the cached search itself.
	tried := make(map[[2]int]bool)
	for row, word := range stencil.Rows {
		for word != 0 {
			column := bits.LeadingZeros32(word)
			word &^= uint32(1) << uint(31-column)
			tileX := (endpoint.player.X + stencil.OriginOffsetX + column) / 16
			tileY := (endpoint.player.Y + endpoint.scroll.Y + stencil.OriginOffsetY + row) / 16
			key := [2]int{tileX, tileY}
			if tried[key] {
				continue
			}
			tried[key] = true
			old := w.Coverage.Map[tileY*w.Coverage.Columns+tileX]
			w.setSecondMapCell(tileX, tileY, opaque)
			if w.Coverage.Touches(endpoint.player.X, endpoint.player.Y, endpoint.scroll.Y, *stencil) && !w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *stencil) {
				patched = true
				break
			}
			w.setSecondMapCell(tileX, tileY, old)
		}
		if patched {
			break
		}
	}
	if !patched || !w.Coverage.Touches(endpoint.player.X, endpoint.player.Y, endpoint.scroll.Y, *stencil) {
		t.Fatal("real tile patch did not cover the stencil")
	}
	if assertNativeTouchCacheSearchEqual(t, w, &p, 202, 2077, &uncached) {
		t.Fatal("patched goal was accepted through stale memo")
	}
	if !p.touchCache.touches(w, endpoint.player.X, endpoint.player.Y, endpoint.scroll.Y) {
		t.Fatal("new search retained old clear value")
	}
}

func TestNativeTouchCacheMiddleWorkerOwnsMemoOptional(t *testing.T) {
	corridor := nativeMotionFixture(t, 193, 176, 1896)
	var source DemoPilot
	var worker thirdMiddleForecastWorker
	if !source.nativeMotion.search(corridor, 202, 2077) || !worker.policy.nativeMotion.search(corridor, 202, 2077) {
		t.Fatal("original cached route missing")
	}
	sourceCache, workerCache := source.nativeMotion.touchCache, worker.policy.nativeMotion.touchCache
	if sourceCache == nil || workerCache == nil || sourceCache == workerCache {
		t.Fatal("memo is not private to each planner")
	}
	sourceTop := sourceCache.top
	middle := expertMiddleForecastScene(t)
	worker.evaluate(middle, MotionInput{}, Input{}, 3, source)
	if worker.err != nil || worker.policy.nativeMotion.touchCache != workerCache || source.nativeMotion.touchCache != sourceCache {
		t.Fatal("policy copy replaced worker memo ownership")
	}
	workerCache.top++
	if sourceCache.top != sourceTop {
		t.Fatal("worker memo mutation escaped into source")
	}
}

func BenchmarkNativeMotionTouchCacheOriginalRoutes(b *testing.B) {
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
			for _, cached := range []bool{false, true} {
				name := "uncached"
				if cached {
					name = "cached"
				}
				b.Run(name, func(b *testing.B) {
					var p nativeMotionPlanner
					search := p.searchWithoutTouchCache
					if cached {
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
