package engine

import (
	"fmt"
	"reflect"
	"testing"
)

// Original map poses isolate terrain and rewind semantics, not campaign progress.
func TestNativeMotionQueryReductionAllMapsAndTimersOptional(t *testing.T) {
	cases := 0
	for level := 1; level <= 5; level++ {
		w := forecastOriginalScene(t, level, false)
		camera := w.ScrollY
		var poses [2]ShipHistory
		var found [2]bool
		for y := 16; y <= 176 && (!found[0] || !found[1]); y += 8 {
			for x := 14; x <= 304; x += 2 {
				contact := w.Coverage.Touches(x, y, camera, *w.Level.PlayerStencil)
				index := 0
				if contact {
					index = 1
				}
				if !found[index] {
					poses[index], found[index] = ShipHistory{camera, x, y}, true
				}
			}
		}
		if !found[0] || !found[1] {
			t.Fatalf("level%d lacks clear/covered source poses", level)
		}
		for contact, pose := range poses {
			for _, timer := range []int{-17, -1, 0, 1} {
				for action, input := range demoDirections {
					t.Run(fmt.Sprintf("level%d/contact%d/timer%d/action%d", level, contact, timer, action), func(t *testing.T) {
						start := newDemoMotionForecast(w)
						start.player.X, start.player.Y = pose.X, pose.Y
						start.rewind = NewTerrainRewind(camera, pose.X, pose.Y)
						start.rewind.Timer = timer
						// Different source history positions exercise the restoring path.
						start.rewind.History[1] = ShipHistory{camera + 2, pose.X + 1, pose.Y - 1}
						before := forecastDigest(w)
						got, want := start, start
						for pass := 0; pass < 3; pass++ {
							accepted, expected := got.advance(w, input), want.advanceBeforeQueryReduction(w, input)
							if accepted != expected || got != want {
								t.Fatalf("pass%d changed acceptance/state: accepted%v expected%v got%+v want%+v", pass, accepted, expected, got, want)
							}
						}
						if forecastDigest(w) != before {
							t.Fatal("forecast changed live source state")
						}
					})
					cases++
				}
			}
		}
	}
	if cases != 360 {
		t.Fatalf("incomplete query-reduction cases:%d", cases)
	}
}

func TestNativeMotionQueryReductionPreservesExactSearchOptional(t *testing.T) {
	for _, fixture := range []struct {
		name                           string
		x, y, camera, targetX, targetY int
	}{
		{"rear-corner", 193, 176, 1896, 202, 2077},
		{"three-pixel-lane", 251, 176, 1732, 254, 1908},
		{"logged-budget-pose", 66, 21, 1722, 64, 1624},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			w := nativeMotionFixture(t, fixture.x, fixture.y, fixture.camera)
			before := forecastDigest(w)
			var got, want nativeMotionPlanner
			accepted, expected := got.search(w, fixture.targetX, fixture.targetY), want.searchBeforeQueryReduction(w, fixture.targetX, fixture.targetY)
			if accepted != expected || got.expanded != want.expanded || !reflect.DeepEqual(got.nodes, want.nodes) || !reflect.DeepEqual(got.queue, want.queue) || !reflect.DeepEqual(got.visited, want.visited) || !reflect.DeepEqual(got.commands, want.commands) || !reflect.DeepEqual(got.states, want.states) {
				t.Fatalf("search changed: accepted%v/%v expanded%d/%d nodes%d/%d commands%d/%d", accepted, expected, got.expanded, want.expanded, len(got.nodes), len(want.nodes), len(got.commands), len(want.commands))
			}
			if forecastDigest(w) != before {
				t.Fatal("search changed live source state")
			}
			t.Logf("accepted%v expanded%d nodes%d commands%d", accepted, got.expanded, len(got.nodes), len(got.commands))
		})
	}
}

func (s *demoMotionForecast) advanceBeforeQueryReduction(w *World, input MotionInput) bool {
	if w.Level.Number == 3 && s.scroll.Y > 208 && (w.ThirdMiddle == nil || w.ThirdMiddle.Defeated) {
		s.scroll.Maximum = demoScrollMaximum(w, s.scroll.Y, s.scroll.Maximum)
	}
	if w.Level.Number == 2 && w.secondBackward && w.secondDefenseRemaining > 0 {
		// Living source node callbacks reopen this reverse bound every pass.
		s.scroll.Maximum = max(s.scroll.Maximum, 2880)
	}
	if s.firstArena {
		event := s.firstMiddle.Advance(s.scroll.Y, s.scroll.Maximum, s.scroll.Y+s.player.Y, [16]int{})
		s.scroll.Y, s.scroll.Maximum = event.Scroll, event.Maximum
	}
	s.player.SpeedTier = w.Equipment.SpeedTier
	s.player.ScrollStep = w.BaseScrollStep
	touching := func() bool {
		return w.Dive.Phase == 0 && w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(s.player.X, s.player.Y, s.scroll.Y, *w.Level.PlayerStencil)
	}
	handled, crushed := false, false
	if w.Dive.Phase == 0 {
		handled, crushed = s.rewind.Advance(&s.player, s.scroll.Y, w.BaseScrollStep, touching())
	}
	if crushed {
		return false
	}
	if !handled {
		s.player.Advance(input, MotionContext{ScrollY: s.scroll.Y, VisitedScrollY: s.scroll.Maximum, BaseScrollStep: w.BaseScrollStep})
		s.rewind.Record(s.scroll.Y, s.player.X, s.player.Y)
		if touching() {
			s.rewind.Timer, s.player.Inertia = 1, 0
			s.scroll.Maximum = max(s.scroll.Maximum, s.scroll.Y+16)
			return false
		}
	}
	s.scroll.Advance(s.player.ScrollStep, w.BaseScrollStep, input.Down)
	return !touching()
}

func (p *nativeMotionPlanner) searchBeforeQueryReduction(w *World, x, worldY int) bool {
	p.nodes, p.queue = p.nodes[:0], p.queue[:0]
	p.commands, p.states, p.expanded = p.commands[:0], p.states[:0], 0
	if p.visited == nil {
		p.visited = make(map[nativeMotionKey]int)
	} else {
		clear(p.visited)
	}
	start := newDemoMotionForecast(w)
	h := nativeMotionTimeBound(start, x, worldY, w.Equipment.SpeedTier)
	p.nodes = append(p.nodes, nativeMotionNode{state: start, parent: -1, action: -1, priority: h, remaining: h, distance: absDemo(start.player.X-x) + absDemo(start.player.Y+start.scroll.Y-worldY)})
	p.visited[nativeMotionStateKey(start)] = 0
	p.push(0)
	for len(p.queue) != 0 && p.expanded < nativeMotionNodeBudget {
		index := p.pop()
		current := p.nodes[index]
		if p.visited[nativeMotionStateKey(current.state)] < current.depth {
			continue
		}
		p.expanded++
		if current.state.player.X == x && current.state.player.Y+current.state.scroll.Y == worldY {
			for at := index; at >= 0; at = p.nodes[at].parent {
				p.states = append(p.states, p.nodes[at].state)
				if p.nodes[at].action >= 0 {
					p.commands = append(p.commands, demoDirections[p.nodes[at].action])
				}
			}
			for i, j := 0, len(p.states)-1; i < j; i, j = i+1, j-1 {
				p.states[i], p.states[j] = p.states[j], p.states[i]
			}
			for i, j := 0, len(p.commands)-1; i < j; i, j = i+1, j-1 {
				p.commands[i], p.commands[j] = p.commands[j], p.commands[i]
			}
			return len(p.commands) != 0
		}
		if current.depth == nativeMotionDepthLimit {
			continue
		}
		for action, input := range demoDirections {
			next := current.state
			if !next.advanceBeforeQueryReduction(w, input) || next.rewind.Timer != 0 {
				continue
			}
			depth, key := current.depth+1, nativeMotionStateKey(next)
			if previous, ok := p.visited[key]; ok && previous <= depth {
				continue
			}
			p.visited[key] = depth
			h := nativeMotionTimeBound(next, x, worldY, w.Equipment.SpeedTier)
			p.nodes = append(p.nodes, nativeMotionNode{state: next, parent: index, action: action, depth: depth, priority: depth + h, remaining: h, distance: absDemo(next.player.X-x) + absDemo(next.player.Y+next.scroll.Y-worldY)})
			p.push(len(p.nodes) - 1)
		}
	}
	return false
}
