package engine

import "math"

// SecondFinalInput opens the original lower terrain barrier with ordinary
// bullets, enters through the cleared passage and retreats to a firing position.
// Every position, tile and guardian health change remains owned by World.Step.
func (p *DemoPilot) SecondFinalInput(w *World) (Input, bool) {
	if w == nil || w.Level.Number != 2 || w.Ready || w.GameOver || !w.PlayerAlive || !w.secondMiddleReleased || w.SecondGuardian == nil || w.secondGuardianActor == nil || !w.secondGuardianActor.Active || w.ScrollY > 352 || w.secondTerrainCells == nil || w.Coverage == nil || w.Level.PlayerStencil == nil || p.Config.DisableBossAlignment {
		return Input{}, false
	}
	input := Input{Fire: (w.Frame+1)%2 != 0 && !w.blockedFireUntilRelease}
	if w.ScrollY > 288 {
		x, y, found := p.secondFinalWaypoint(w, 464)
		if !found {
			return Input{}, false
		}
		if p.practicedRoute {
			// Approaching the barrier still crosses live pod-creature traffic.
			// Preserve the route goal while forecasting terrain and body contact.
			input.Motion = demoRouteMotion(w, x, y)
		} else {
			input.Motion = secondFinalRouteMotion(w, x, y)
		}
		return input, true
	}
	worldY := w.Player.Y + w.ScrollY
	if w.SecondGuardian.WaitTimer < 0 && w.SecondGuardian.MotionRemaining == 0 && w.SecondGuardian.BodyCollision.Empty() {
		// Three adjacent columns create a passage wide enough for the complete
		// ship stencil. Intact flags change only when real projectiles hit them.
		for _, column := range []int{192, 208, 224} {
			for index, cell := range w.secondTerrainCells.Cells {
				if cell.Quadrant == 0 && cell.X == column && w.secondTerrainCells.Intact[index] {
					input.Motion = demoAimMotion(w.Player, column+8, 176)
					input.Motion.Down = true
					return input, true
				}
			}
		}
		x, y, found := p.secondFinalWaypoint(w, 176)
		if !found {
			return Input{}, false
		}
		input.Motion = secondFinalRouteMotion(w, x, y)
		return input, true
	}
	if worldY < 448 && (w.Player.X < 124 || w.Player.X > 196) {
		x, y, found := p.secondFinalPointWaypoint(w, 160, 448)
		if !found {
			return Input{}, false
		}
		input.Motion = secondFinalRouteMotion(w, x, y)
		return input, true
	}
	// The solid central rock ends below world Y362. Cross beneath its lower
	// barrier, then keep the camera around240 while firing along the live
	// weak point's X152..167 span.
	input.Motion = demoAimMotion(w.Player, 160, 176)
	if worldY >= 432 || absDemo(w.Player.X-160) <= 8 {
		input.Motion.Down = w.ScrollY <= 240
	}
	return input, true
}

func demoAimMotion(player PlayerMotionState, x, y int) MotionInput {
	return MotionInput{Left: player.X > x+1, Right: player.X < x-1, Up: player.Y > y+1, Down: player.Y < y-1 && player.Y < 176}
}

func secondFinalRouteMotion(w *World, x, y int) MotionInput {
	best, rank := MotionInput{}, math.Inf(1)
	for _, motion := range demoDirections {
		forecast := newDemoMotionForecast(w)
		if !forecast.advance(w, motion) {
			continue
		}
		player, scroll := forecast.player, forecast.scroll
		distance := float64(absDemo(player.X-x) + absDemo(player.Y+scroll.Y-y)*2)
		if distance < rank {
			best, rank = motion, distance
		}
	}
	return best
}

func (p *DemoPilot) secondFinalWaypoint(w *World, goal int) (int, int, bool) {
	return p.secondFinalPointWaypoint(w, 0, goal)
}

func (p *DemoPilot) secondFinalPointWaypoint(w *World, targetX, goal int) (int, int, bool) {
	if w.Coverage == nil || w.Level.PlayerStencil == nil || w.Coverage.Columns != 20 || w.Coverage.Rows != 300 || len(w.Coverage.Map) != 6000 {
		return 0, 0, false
	}
	if p.navigation == nil {
		p.navigation = &demoNavigation{}
	}
	n := p.navigation
	n.targetX = targetX
	changed := n.refresh(w)
	x, y := w.Player.X, w.Player.Y+w.ScrollY
	if changed || len(n.path) == 0 || n.goal != goal || n.pathTargetX != targetX || w.Frame-n.frame >= 24 {
		n.goal, n.frame = goal, w.Frame
		if !n.searchSecondFinal(x, y, goal) {
			return 0, 0, false
		}
	}
	nearest, distance := 0, math.MaxInt
	for index, point := range n.path {
		d := absDemo(point.x-x) + absDemo(point.y-y)
		if d < distance {
			nearest, distance = index, d
		}
	}
	if distance > 48 {
		n.frame = 0
		return 0, 0, false
	}
	if p.practicedRoute {
		for index := min(len(n.path)-1, nearest+8); index >= nearest; index-- {
			point := n.path[index]
			if n.clearSegment(x, y, point) {
				return point.x, point.y, true
			}
		}
		return 0, 0, false
	}
	point := n.path[min(len(n.path)-1, nearest+8)]
	return point.x, point.y, true
}

// searchSecondFinal shares the bounded terrain cache and heap with ordinary
// navigation, while allowing the source arena's required backward traversal.
func (n *demoNavigation) searchSecondFinal(x, y, goal int) bool {
	n.pathTargetX = n.targetX
	direction := 1
	if goal < y {
		direction = -1
	}
	low, high := max(0, min(goal, y)-48), min(4799, max(goal, y)+192)
	n.queue, n.nodes, n.path = n.queue[:0], n.nodes[:0], n.path[:0]
	if n.visited == nil {
		n.visited = make(map[demoNavPoint]float64, 9000)
	} else {
		clear(n.visited)
	}
	n.visited[demoNavPoint{x, y}] = 0
	distance := func(x, y int) float64 {
		dy := max(0, direction*(goal-y))
		dx := 0
		if n.targetX != 0 {
			dx = max(0, absDemo(x-n.targetX)-3)
		}
		return (float64(max(dx, dy)) + .42*float64(min(dx, dy))) / 3
	}
	n.push(demoNavNode{x: x, y: y, parent: -1, rank: distance(x, y)})
	for len(n.queue) > 0 && len(n.nodes) < 9000 {
		current := n.pop()
		if best, ok := n.visited[demoNavPoint{current.x, current.y}]; ok && current.cost > best+.01 {
			continue
		}
		index := len(n.nodes)
		n.nodes = append(n.nodes, current)
		if direction*(current.y-goal) >= 0 && (n.targetX == 0 || absDemo(current.x-n.targetX) <= 3) {
			for at := index; at >= 0; at = n.nodes[at].parent {
				node := n.nodes[at]
				n.path = append(n.path, demoNavPoint{node.x, node.y})
			}
			for l, r := 0, len(n.path)-1; l < r; l, r = l+1, r-1 {
				n.path[l], n.path[r] = n.path[r], n.path[l]
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
				point := demoNavPoint{nx, ny}
				if prior, ok := n.visited[point]; ok && prior <= cost {
					continue
				}
				n.visited[point] = cost
				n.push(demoNavNode{x: nx, y: ny, parent: index, cost: cost, rank: cost + distance(nx, ny)})
			}
		}
	}
	return false
}
