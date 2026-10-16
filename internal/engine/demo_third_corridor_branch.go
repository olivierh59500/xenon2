package engine

// thirdCorridorProgress detects a repeated local dodge without discarding an
// ordinary rearward leg that advances along the complete terrain route.
type thirdCorridorProgress struct {
	world                             *World
	targetX, targetY, score, furthest int
	pathFrame, since                  uint64
	recovering                        bool
}

func thirdCorridorBranchWindow(w *World) bool {
	return w != nil && w.Level.Number == 3 && w.ThirdMiddle != nil && w.ThirdMiddle.Defeated && w.ScrollY > 208 && w.PlayerAlive && !w.GameOver && !w.Ready && !w.ShopReady && !w.ExitReady && !w.LevelFinished && w.PendingExitDrops == 0 && w.ScreenClearFrames == 0 && !w.stepContinuation.active && w.Rewind.Timer == 0 && w.Coverage != nil && w.Level.PlayerStencil != nil
}

func (p *PresentationPilot) thirdCorridorRecoveryNeeded(w *World) bool {
	n := p.planner.navigation
	if !thirdCorridorBranchWindow(w) || n == nil || len(n.path) == 0 {
		return false
	}
	activeTarget := false
	for _, actor := range w.Actors {
		if actor.Active && actor.thirdCannon != nil && actor.thirdCannon.X+32 == n.pointTargetX && actor.thirdCannon.WorldY+136 == n.goal {
			activeTarget = true
			break
		}
	}
	if !activeTarget {
		p.thirdCorridorWatch = thirdCorridorProgress{}
		return false
	}
	// A ship already at the firing lane may legitimately wait for the gun's
	// vulnerable phase. That is different from failing to reach the lane.
	if absDemo(w.Player.X-n.pointTargetX) <= 16 && absDemo(w.Player.Y+w.ScrollY-n.goal) <= 36 {
		p.thirdCorridorWatch = thirdCorridorProgress{}
		return false
	}
	nearest, distance := 0, int(^uint(0)>>1)
	for i, point := range n.path {
		d := absDemo(point.x-w.Player.X) + absDemo(point.y-w.Player.Y-w.ScrollY)
		if d < distance {
			nearest, distance = i, d
		}
	}
	watch := &p.thirdCorridorWatch
	if watch.world != w || watch.targetX != n.pointTargetX || watch.targetY != n.goal || watch.score != w.Score || w.Frame < watch.since {
		*watch = thirdCorridorProgress{world: w, targetX: n.pointTargetX, targetY: n.goal, score: w.Score, furthest: nearest, pathFrame: n.frame, since: w.Frame}
	} else if watch.pathFrame != n.frame {
		watch.pathFrame, watch.furthest, watch.since = n.frame, nearest, w.Frame
	} else if nearest > watch.furthest {
		watch.furthest, watch.since = nearest, w.Frame
	}
	watch.recovering = watch.recovering || w.Frame-watch.since >= 120
	return watch.recovering
}

func thirdCorridorBranchSafe(w *World, r ForecastResult, shield int) bool {
	return thirdCorridorBranchWindow(w) && r.Boundary == ForecastRunning && r.Alive && r.Shield >= shield && !w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil)
}

// chooseThirdCorridorBranch tries a short initial dodge followed by the route's
// remaining commands. Holding the dodge for the entire horizon can hide this
// safe retreat and repeatedly drive the player back into the same pocket.
func (p *PresentationPilot) chooseThirdCorridorBranch(w *World, planned Input, sequence [6]MotionInput, retained bool) (Input, bool) {
	if !thirdCorridorBranchWindow(w) || !p.thirdCorridorWatch.recovering {
		return Input{}, false
	}
	n := p.planner.navigation
	if n == nil {
		return Input{}, false
	}
	x, y, found := n.pointWaypoint(w, n.pointTargetX, n.goal)
	if !found {
		return Input{}, false
	}
	if !retained {
		for i := range sequence {
			sequence[i] = planned.Motion
			sequence[i].Left, sequence[i].Right = false, false
			if y > w.ScrollY+w.Player.Y {
				sequence[i].Up, sequence[i].Down = false, true
			}
		}
	}
	if p.thirdCorridorBranch == nil {
		p.thirdCorridorBranch = &retainedGuardPlan{}
	}
	plan := p.thirdCorridorBranch
	plan.count = 0
	pal := thirdMiddlePALRefreshes(p.PALRefreshes)
	best := int(^uint(0) >> 1)
	var selected [6]Input
	var selectedKeys [7]retainedGuardKey
	for _, motion := range demoDirections {
		if err := plan.forecast.Load(w); err != nil {
			return Input{}, false
		}
		var inputs [6]Input
		var keys [7]retainedGuardKey
		keys[0] = retainedGuardStateKey(w)
		safe := true
		for pass := range inputs {
			inputs[pass] = planned
			inputs[pass].Motion = sequence[pass]
			if pass == 0 {
				inputs[pass].Motion = motion
			}
			for range pal {
				plan.forecast.AdvancePALTick()
			}
			r, err := plan.forecast.Advance(inputs[pass])
			q := plan.forecast.State()
			if err != nil || !thirdCorridorBranchSafe(q, r, w.Equipment.Shield) {
				safe = false
				break
			}
			keys[pass+1] = retainedGuardStateKey(q)
		}
		if !safe {
			continue
		}
		q := plan.forecast.State()
		distance := absDemo(q.Player.X-x) + absDemo(q.Player.Y+q.ScrollY-y)*2
		if distance < best {
			best, selected, selectedKeys = distance, inputs, keys
		}
	}
	initialDistance := absDemo(w.Player.X-x) + absDemo(w.Player.Y+w.ScrollY-y)*2
	if best >= initialDistance {
		return Input{}, false
	}
	plan.world, plan.before, plan.input, plan.at, plan.count, plan.pal = w, selectedKeys, selected, 1, 6, pal
	return selected[0], true
}

func (p *PresentationPilot) continueThirdCorridorBranch(w *World) (Input, bool) {
	plan := p.thirdCorridorBranch
	if plan == nil || plan.count == 0 {
		return Input{}, false
	}
	if !thirdCorridorBranchWindow(w) || plan.world != w || plan.pal != thirdMiddlePALRefreshes(p.PALRefreshes) {
		plan.count = 0
		return Input{}, false
	}
	key, index := retainedGuardStateKey(w), plan.at
	repeated := index > 0 && key == plan.before[index-1]
	if repeated {
		index--
	}
	if index >= plan.count || key != plan.before[index] {
		plan.count = 0
		return Input{}, false
	}
	if err := plan.forecast.Load(w); err != nil {
		plan.count = 0
		return Input{}, false
	}
	for pass := index; pass < plan.count; pass++ {
		shield := plan.forecast.State().Equipment.Shield
		for range plan.pal {
			plan.forecast.AdvancePALTick()
		}
		r, err := plan.forecast.Advance(plan.input[pass])
		q := plan.forecast.State()
		if err != nil || !thirdCorridorBranchSafe(q, r, shield) || retainedGuardStateKey(q) != plan.before[pass+1] {
			plan.count = 0
			return Input{}, false
		}
	}
	if !repeated {
		plan.at = index + 1
	}
	return plan.input[index], true
}
