package engine

import "math"

// searchPoint retains a complete route to a firing position, including legal
// rearward legs. Node and queue bounds cap planner work.
func (n *demoNavigation) searchPoint(x, y, tx, goal int) bool {
	n.pathTargetX = 0

	low, high := max(0, min(goal, y)-48), min(4799, max(y, goal)+640)
	if w := n.world; w != nil && n.practiced {
		maximum := demoScrollMaximum(w, w.ScrollY, w.MaximumScrollY)
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
	if n.pointClosed == nil {
		n.pointClosed = make(map[demoNavPoint]bool, 12000)
	} else {
		clear(n.pointClosed)
	}
	n.visited[demoNavPoint{x, y}] = 0
	n.push(demoNavNode{x: x, y: y, parent: -1, rank: float64(absDemo(y-goal) + absDemo(x-tx))})
	for len(n.queue) > 0 && len(n.nodes) < 30000 {
		current := n.pop()
		point := demoNavPoint{current.x, current.y}
		if n.pointClosed[point] {
			continue
		}
		if best, ok := n.visited[demoNavPoint{current.x, current.y}]; ok && current.cost > best+.01 {
			continue
		}
		index := len(n.nodes)
		n.pointClosed[point] = true
		n.nodes = append(n.nodes, current)
		if absDemo(current.y-goal) <= 6 && absDemo(current.x-tx) <= 6 {
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
				if n.pointClosed[point] {
					continue
				}
				if prior, ok := n.visited[point]; ok && prior <= cost {
					continue
				}
				n.visited[point] = cost
				n.push(demoNavNode{x: nx, y: ny, parent: index, cost: cost, rank: cost + float64(absDemo(ny-goal)+absDemo(nx-tx))})
			}
		}
	}
	return false
}
func (n *demoNavigation) pointWaypoint(w *World, tx, ty int) (int, int, bool) {
	if w == nil || w.Coverage == nil || w.Level.PlayerStencil == nil || w.Coverage.Columns != 20 || w.Coverage.Rows != 300 || len(w.Coverage.Map) != 6000 {
		return 0, 0, false
	}
	changed := n.refresh(w)
	x, y := w.Player.X, w.Player.Y+w.ScrollY
	invalid := len(n.path) == 0 || n.goal != ty || n.pointTargetX != tx
	if !invalid && changed {
		// Animated cannon tiles can change without occupying the planned route.
		// Keep its rearward legs unless new coverage actually blocks a node.
		rear := demoScrollMaximum(w, w.ScrollY, w.MaximumScrollY) + 176
		for _, point := range n.path {
			if n.touching(point.x, point.y) || point.y > rear {
				invalid = true
				break
			}
		}
	}
	if invalid {
		n.goal, n.pointTargetX, n.frame = ty, tx, w.Frame
		if !n.searchPoint(x, y, tx, ty) {
			return 0, 0, false
		}
	}
	near, best := 0, math.MaxInt
	for i, q := range n.path {
		v := absDemo(q.x-x) + absDemo(q.y-y)
		if v < best {
			near, best = i, v
		}
	}
	if best > 48 {
		n.path = n.path[:0]
		return 0, 0, false
	}
	for i := min(len(n.path)-1, near+8); i >= near; i-- {
		q := n.path[i]
		if n.clearSegment(x, y, q) {
			return q.x, q.y, true
		}
	}
	return 0, 0, false
}
