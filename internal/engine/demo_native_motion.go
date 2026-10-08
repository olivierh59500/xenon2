package engine

const (
	nativeMotionNodeBudget = 9000
	nativeMotionDepthLimit = 32
)

type nativeMotionKey struct{ x, y, inertia, camera, maximum, deviation int }
type nativeCompactMotionState struct {
	player PlayerMotionState
	scroll ScrollState
}

type nativeCompactMotionNode struct {
	state                         nativeCompactMotionState
	parent, action, depth         int
	priority, remaining, distance int
}

type nativeMotionNode struct {
	state                         demoMotionForecast
	parent, action, depth         int
	priority, remaining, distance int
}

// nativeMotionPlanner executes a short clear route with the actual integer
// inertia and scroll rules. Its committed commands prevent a moving horizon
// from postponing the same braking manoeuvre indefinitely.
type nativeMotionPlanner struct {
	world              *World
	targetX, targetY   int
	commands           []MotionInput
	states             []demoMotionForecast
	at                 int
	tiles              []uint16
	nodes              []nativeMotionNode
	compactNodes       []nativeCompactMotionNode
	queue              []int
	visited            map[nativeMotionKey]int
	expanded           int
	terminalHorizontal nativeHorizontalBound
	touchCache         *nativeMotionTouchCache
}

func nativeMotionStateKey(state demoMotionForecast) nativeMotionKey {
	return nativeMotionKey{state.player.X, state.player.Y, state.player.Inertia, state.scroll.Y, state.scroll.Maximum, state.scroll.DeviationPasses}
}

// nativeMotionTimeBound is optimistic about independent horizontal and vertical
// travel. It permits the source Down overshoot and accounts for every doubled
// reverse pass after the thirty-fifth nonbase scroll request.
func nativeMotionTimeBound(state demoMotionForecast, x, worldY, speed int) int {
	horizontal := 6
	if speed == 0 {
		horizontal = 3
	} else if speed >= 2 {
		horizontal = 9
	}
	ceil := func(value, step int) int { return (value + step - 1) / step }
	h := ceil(absDemo(state.player.X-x), horizontal)
	delta := worldY - state.player.Y - state.scroll.Y
	if delta < 0 {
		return max(h, ceil(-delta, 4+speed))
	}
	room := max(0, 176+(3+speed)-1-state.player.Y)
	needed := max(0, delta-room)
	low, high := 0, needed
	for low < high {
		passes := (low + high) / 2
		upper := passes + min(passes, max(0, state.scroll.DeviationPasses+passes-34))
		if upper >= needed {
			high = passes
		} else {
			low = passes + 1
		}
	}
	return max(h, low)
}

func nativeMotionNodeLess(a, b nativeMotionNode) bool {
	if a.priority != b.priority {
		return a.priority < b.priority
	}
	if a.remaining != b.remaining {
		return a.remaining < b.remaining
	}
	return a.distance < b.distance
}

func (p *nativeMotionPlanner) push(index int) {
	p.queue = append(p.queue, index)
	at := len(p.queue) - 1
	for at > 0 {
		parent := (at - 1) / 2
		if !nativeMotionNodeLess(p.nodes[p.queue[at]], p.nodes[p.queue[parent]]) {
			break
		}
		p.queue[parent], p.queue[at] = p.queue[at], p.queue[parent]
		at = parent
	}
}

func (p *nativeMotionPlanner) pop() int {
	index := p.queue[0]
	p.queue[0] = p.queue[len(p.queue)-1]
	p.queue = p.queue[:len(p.queue)-1]
	for at := 0; at < len(p.queue); {
		left := at*2 + 1
		if left >= len(p.queue) {
			break
		}
		child := left
		if left+1 < len(p.queue) && nativeMotionNodeLess(p.nodes[p.queue[left+1]], p.nodes[p.queue[left]]) {
			child = left + 1
		}
		if !nativeMotionNodeLess(p.nodes[p.queue[child]], p.nodes[p.queue[at]]) {
			break
		}
		p.queue[at], p.queue[child] = p.queue[child], p.queue[at]
		at = child
	}
	return index
}

func nativeCompactMotionNodeLess(a, b nativeCompactMotionNode) bool {
	if a.priority != b.priority {
		return a.priority < b.priority
	}
	if a.remaining != b.remaining {
		return a.remaining < b.remaining
	}
	return a.distance < b.distance
}

func (p *nativeMotionPlanner) compactPush(index int) {
	p.queue = append(p.queue, index)
	at := len(p.queue) - 1
	for at > 0 {
		parent := (at - 1) / 2
		if !nativeCompactMotionNodeLess(p.compactNodes[p.queue[at]], p.compactNodes[p.queue[parent]]) {
			break
		}
		p.queue[parent], p.queue[at] = p.queue[at], p.queue[parent]
		at = parent
	}
}

func (p *nativeMotionPlanner) compactPop() int {
	index := p.queue[0]
	p.queue[0] = p.queue[len(p.queue)-1]
	p.queue = p.queue[:len(p.queue)-1]
	for at := 0; at < len(p.queue); {
		left := at*2 + 1
		if left >= len(p.queue) {
			break
		}
		child := left
		if left+1 < len(p.queue) && nativeCompactMotionNodeLess(p.compactNodes[p.queue[left+1]], p.compactNodes[p.queue[left]]) {
			child = left + 1
		}
		if !nativeCompactMotionNodeLess(p.compactNodes[p.queue[child]], p.compactNodes[p.queue[at]]) {
			break
		}
		p.queue[at], p.queue[child] = p.queue[child], p.queue[at]
		at = child
	}
	return index
}

func (p *nativeMotionPlanner) searchFull(w *World, x, worldY int) bool {
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
			if !next.advance(w, input) || next.rewind.Timer != 0 {
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

func nativeMotionMatches(w *World, state demoMotionForecast) bool {
	return w.Player == state.player && w.ScrollY == state.scroll.Y && w.MaximumScrollY == state.scroll.Maximum && w.MinimumScrollY == state.scroll.Minimum && w.ScrollDeviationPasses == state.scroll.DeviationPasses && w.Rewind.Timer == state.rewind.Timer
}

func nativeMotionSupported(w *World) bool {
	thirdCorridor := w != nil && w.Level.Number == 3 && w.ThirdMiddle != nil && w.ThirdMiddle.Defeated
	return (thirdCorridor || fourthCorridorActive(w) || fourthRearLegWindow(w)) && w.PlayerAlive && !w.Ready && !w.GameOver && w.Rewind.Timer == 0 && w.Coverage != nil && w.Level.PlayerStencil != nil
}

// guardSequence describes the command already returned to this pass's caller
// and its retained successors. The caller has not advanced the world yet, so
// even the final command must match its before-state rather than its endpoint.
// Reading this preview never consumes or invalidates the retained route.
func (p *nativeMotionPlanner) guardSequence(w *World, first MotionInput) ([6]MotionInput, bool) {
	var sequence [6]MotionInput
	index := p.at - 1
	if p.world != w || w == nil || index < 0 || index >= len(p.commands) || index >= len(p.states) || p.commands[index] != first || !nativeMotionMatches(w, p.states[index]) {
		return sequence, false
	}
	copy(sequence[:], p.commands[index:])
	return sequence, true
}

// continueRoute must be called before asking the geometric planner for another
// waypoint. An intermediate braking pose can temporarily lack a straight
// geometric shortcut even while the committed source-motion sequence is valid.
func (p *nativeMotionPlanner) continueRoute(w *World, targetX, targetY int) (MotionInput, bool) {
	if !nativeMotionSupported(w) || p.world != w || p.at >= len(p.commands) || !nativeMotionMatches(w, p.states[p.at]) {
		p.world = nil
		return MotionInput{}, false
	}
	if p.targetX != targetX || p.targetY != targetY {
		// A caller may check several live cannon candidates before reaching
		// the target that owns this still-valid commitment.
		return MotionInput{}, false
	}
	changed := len(p.tiles) != len(w.Coverage.Map)
	if !changed {
		for index, id := range p.tiles {
			if id != w.Coverage.Map[index] {
				changed = true
				break
			}
		}
	}
	if changed {
		for _, state := range p.states[p.at:] {
			if w.Coverage.Touches(state.player.X, state.player.Y, state.scroll.Y, *w.Level.PlayerStencil) {
				p.world = nil
				return MotionInput{}, false
			}
		}
		p.tiles = append(p.tiles[:0], w.Coverage.Map...)
	}
	input := p.commands[p.at]
	p.at++
	return input, true
}

func (p *nativeMotionPlanner) command(w *World, x, worldY, targetX, targetY int) (MotionInput, bool) {
	if input, ok := p.continueRoute(w, targetX, targetY); ok {
		return input, true
	}
	p.world = nil
	if !nativeMotionSupported(w) || !p.search(w, x, worldY) {
		return MotionInput{}, false
	}
	p.world, p.targetX, p.targetY, p.at = w, targetX, targetY, 0
	p.tiles = append(p.tiles[:0], w.Coverage.Map...)
	return p.continueRoute(w, targetX, targetY)
}

// search stores only motion/camera state while terrain rewind is impossible in
// retained nodes. Every successor still runs the existing forecast routine.
func (p *nativeMotionPlanner) search(w *World, x, worldY int) bool {
	p.compactNodes = p.compactNodes[:0]
	if !nativeMotionSupported(w) || w.Dive.Phase != 0 || w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		return p.searchFull(w, x, worldY)
	}
	if p.touchCache == nil {
		p.touchCache = new(nativeMotionTouchCache)
	}
	p.touchCache.reset(w.Player.Y + w.ScrollY - 224)
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
			if !next.advanceWithTouchCache(w, input, p.touchCache) || next.rewind.Timer != 0 {
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

func (p *nativeMotionPlanner) nodeCount() int {
	if p.expanded != 0 && len(p.compactNodes) != 0 {
		return len(p.compactNodes)
	}
	return len(p.nodes)
}

// The original terrain map and stencil are immutable during one synchronous
// search. Positions outside this bounded memo still use the source query.
type nativeMotionTouchCache struct {
	top          int
	known, solid [320][10]uint32
}

func (c *nativeMotionTouchCache) reset(top int) {
	c.top = top
	clear(c.known[:])
}

func (c *nativeMotionTouchCache) touches(w *World, x, y, camera int) bool {
	row := y + camera - c.top
	if x < 0 || x >= 320 || row < 0 || row >= len(c.known) {
		return w.Coverage.Touches(x, y, camera, *w.Level.PlayerStencil)
	}
	column, bit := x>>5, uint32(1)<<uint(x&31)
	if c.known[row][column]&bit != 0 {
		return c.solid[row][column]&bit != 0
	}
	solid := w.Coverage.Touches(x, y, camera, *w.Level.PlayerStencil)
	c.known[row][column] |= bit
	if solid {
		c.solid[row][column] |= bit
	} else {
		c.solid[row][column] &^= bit
	}
	return solid
}
