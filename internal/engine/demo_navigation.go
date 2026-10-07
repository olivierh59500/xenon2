package engine

import "math"

type demoNavPoint struct{ x, y int }
type demoNavNode struct {
	x, y, parent int
	cost, rank   float64
}

// demoNavigation caches terrain occupancy and a short world-coordinate route.
// It only plans ordinary input and never writes collision or gameplay state.
type demoNavigation struct {
	world                *World
	tiles                []uint16
	rows                 [4800][11]uint32
	stencilRows          []uint32
	originX, originY     int
	queue, nodes         []demoNavNode
	visited              map[demoNavPoint]float64
	path                 []demoNavPoint
	goal                 int
	frame                uint64
	retreat              bool
	targetX, pathTargetX int
	pointTargetX         int
	pointClosed          map[demoNavPoint]bool
	practiced            bool
}

// demoScrollMaximum includes the source stage prelude that runs before player
// movement. Its copy retains arena admission and shop-boundary ordering without
// changing gates, launch seeds, the world or the shared random stream.
func demoScrollMaximum(w *World, camera, maximum int) int {
	if w != nil && w.Level.Number == 1 && w.FirstMiddle != nil {
		state := *w.FirstMiddle
		event := state.Advance(camera, maximum, camera+w.Player.Y, [16]int{})
		return event.Maximum
	}
	if w != nil && w.Level.Number == 3 && w.ThirdFinal != nil {
		state := w.ThirdStage
		event := state.Advance(ThirdStageInput{ScrollY: camera, MinimumScrollY: w.MinimumScrollY, MaximumScrollY: maximum, RequestedStep: w.BaseScrollStep, MiddleUpdated: w.thirdMiddleUpdated, FinalUpdated: w.thirdFinalUpdated, FinalDefeated: w.ThirdFinal.Defeated})
		return event.MaximumScrollY
	}
	return maximum
}

func (n *demoNavigation) touching(x, y int) bool {
	x += n.originX
	y += n.originY
	if x < 0 || x >= 320 {
		return true
	}
	word, shift := x/32, uint(x%32)
	for index, mask := range n.stencilRows {
		if mask == 0 || y+index < 0 || y+index >= 4800 {
			continue
		}
		row := n.rows[y+index]
		pixels := row[word] << shift
		if shift != 0 {
			pixels |= row[word+1] >> (32 - shift)
		}
		if pixels&mask != 0 {
			return true
		}
	}
	return false
}

func (n *demoNavigation) clearSegment(x, y int, to demoNavPoint) bool {
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
func (n *demoNavigation) refresh(w *World) bool {
	changed := false
	if n.world != w {
		n.world = w
		n.tiles = make([]uint16, len(w.Coverage.Map))
		for index := range n.tiles {
			n.tiles[index] = 65535
		}
		n.rows = [4800][11]uint32{}
		n.path = n.path[:0]
		n.stencilRows = w.Level.PlayerStencil.Rows
		n.originX, n.originY = w.Level.PlayerStencil.OriginOffsetX, w.Level.PlayerStencil.OriginOffsetY
	}
	for index, id := range w.Coverage.Map {
		if n.tiles[index] == id {
			continue
		}
		changed = true
		n.tiles[index] = id
		column, row := index%20, index/20
		shift := uint(16 * (1 - column%2))
		mask := uint32(0xffff) << shift
		coverage := w.Coverage.coverage[id]
		if id == 0 {
			coverage = [16]uint16{}
		}
		for y, pixels := range coverage {
			word := &n.rows[row*16+y][column/2]
			*word = *word&^mask | uint32(pixels)<<shift
		}
	}
	return changed
}
func (n *demoNavigation) push(value demoNavNode) {
	if len(n.queue) >= 36000 {
		return
	}
	n.queue = append(n.queue, value)
	index := len(n.queue) - 1
	for index > 0 {
		parent := (index - 1) / 2
		if n.queue[parent].rank <= n.queue[index].rank {
			break
		}
		n.queue[parent], n.queue[index] = n.queue[index], n.queue[parent]
		index = parent
	}
}
func (n *demoNavigation) pop() demoNavNode {
	value := n.queue[0]
	last := len(n.queue) - 1
	n.queue[0] = n.queue[last]
	n.queue = n.queue[:last]
	for index := 0; index < len(n.queue); {
		left := index*2 + 1
		if left >= len(n.queue) {
			break
		}
		child := left
		if right := left + 1; right < len(n.queue) && n.queue[right].rank < n.queue[left].rank {
			child = right
		}
		if n.queue[index].rank <= n.queue[child].rank {
			break
		}
		n.queue[index], n.queue[child] = n.queue[child], n.queue[index]
		index = child
	}
	return value
}
func (n *demoNavigation) search(x, y, goal int) bool {
	n.pathTargetX = n.targetX
	if goal >= y {
		goal = y - 48
	}
	low, high := max(0, goal-48), min(4799, y+192)
	if w := n.world; w != nil && n.practiced {
		maximum := demoScrollMaximum(w, w.ScrollY, w.MaximumScrollY)
		// The first arena reopens its reverse limit every pass. Its rear exit
		// can lie farther away than one screen even when the displayed limit
		// has just contracted to the ordinary sixteen-pixel reverse buffer.
		if w.Level.Number == 1 && w.FirstMiddle != nil && w.ScrollY >= 2624 && w.ScrollY <= 3344 && maximum >= 3344 {
			high = min(4799, y+384)
		}
		high = min(high, maximum+176)
	}
	n.queue = n.queue[:0]
	n.nodes = n.nodes[:0]
	n.path = n.path[:0]
	if n.visited == nil {
		n.visited = make(map[demoNavPoint]float64, 9000)
	} else {
		clear(n.visited)
	}
	n.visited[demoNavPoint{x, y}] = 0
	n.push(demoNavNode{x: x, y: y, parent: -1, rank: float64(max(0, y-goal)) / 3})
	for len(n.queue) > 0 && len(n.nodes) < 9000 {
		current := n.pop()
		if best, ok := n.visited[demoNavPoint{current.x, current.y}]; ok && current.cost > best+.01 {
			continue
		}
		index := len(n.nodes)
		n.nodes = append(n.nodes, current)
		if current.y <= goal && (n.targetX == 0 || absDemo(current.x-n.targetX) <= 6) {
			for at := index; at >= 0; at = n.nodes[at].parent {
				node := n.nodes[at]
				n.path = append(n.path, demoNavPoint{node.x, node.y})
			}
			for left, right := 0, len(n.path)-1; left < right; left, right = left+1, right-1 {
				n.path[left], n.path[right] = n.path[right], n.path[left]
			}
			return true
		}
		for dx := -3; dx <= 3; dx += 3 {
			for dy := -3; dy <= 3; dy += 3 {
				if dx == 0 && dy == 0 {
					continue
				}
				nx, ny := current.x+dx, current.y+dy
				if nx < 14 || nx > 304 || ny < low || ny > high || n.touching(nx, ny) {
					continue
				}
				cost := current.cost + 1
				if dx != 0 && dy != 0 {
					cost += .42
				}
				cost += math.Abs(float64(nx-160)) * .0001
				point := demoNavPoint{nx, ny}
				if prior, ok := n.visited[point]; ok && prior <= cost {
					continue
				}
				n.visited[point] = cost
				n.push(demoNavNode{x: nx, y: ny, parent: index, cost: cost, rank: cost + float64(max(0, ny-goal))/3})
			}
		}
	}
	return false
}
func (n *demoNavigation) waypoint(w *World, goal int) (int, int, bool) {
	n.retreat = false
	if w.Coverage == nil || w.Level.PlayerStencil == nil || w.Coverage.Columns != 20 || w.Coverage.Rows != 300 || len(w.Coverage.Map) != 6000 {
		return 0, 0, false
	}
	changed := n.refresh(w)
	x, y := w.Player.X, w.Player.Y+w.ScrollY
	unreachable := false
	rear := demoScrollMaximum(w, w.ScrollY, w.MaximumScrollY) + 176
	for _, point := range n.path {
		if n.practiced && point.y > rear {
			unreachable = true
			break
		}
	}
	if changed || unreachable || n.targetX != n.pathTargetX || len(n.path) == 0 || goal != n.goal && absDemo(goal-n.goal) > 48 || w.Frame-n.frame >= 24 {
		n.goal, n.frame = goal, w.Frame
		if !n.search(x, y, goal) {
			return 0, 0, false
		}
	}
	nearest, best := 0, math.MaxInt
	for index, point := range n.path {
		distance := absDemo(point.x-x) + absDemo(point.y-y)
		if distance < best {
			nearest, best = index, distance
		}
	}
	if best > 48 {
		n.frame = 0
		return 0, 0, false
	}
	if !n.practiced {
		point := n.path[min(len(n.path)-1, nearest+8)]
		return point.x, point.y, true
	}
	// A clear distant endpoint can lie beyond a blocked corner. Keep the
	// first leg until the complete ship stencil can traverse the shortcut.
	for index := min(len(n.path)-1, nearest+8); index >= nearest; index-- {
		point := n.path[index]
		if n.clearSegment(x, y, point) {
			n.retreat = point.y > y
			return point.x, point.y, true
		}
	}
	n.frame = 0
	return 0, 0, false
}
