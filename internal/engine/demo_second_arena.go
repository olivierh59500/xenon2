package engine

// demoSecondArenaBeam allows turns within a bounded eight-pass horizon. It
// ranks native player/rewind/camera copies against the active source formation,
// retaining ordinary node targets and returning controls without changing state.
func demoSecondArenaBeam(w *World, p *DemoPilot, fallback MotionInput) MotionInput {
	if w.Rewind.Timer != 0 || w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		return demoSecondArenaBeamOriginal(w, p, fallback)
	}
	x, y, nodeVisible, found := p.secondDefenseTarget(w)
	routeX, routeY, routeFound := secondArenaCrossingGoal(w, p)
	if !found && !routeFound {
		return fallback
	}
	for _, actor := range w.Actors {
		if !actor.Active || actor.secondSegment == nil {
			continue
		}
		if actor.path == nil {
			return fallback
		}
		for _, command := range actor.path.Commands {
			if command.Kind != "curve" && command.Kind != "end" {
				return fallback
			}
		}
	}
	if w.secondScheduler.DefenseFlags != 3 && nodeVisible {
		y = 176
	}
	if routeFound {
		x, y = routeX, routeY
	}
	const horizon = 8
	type branch struct {
		motion     demoMotionForecast
		first      int
		actorShift int
		nodes      [3]SecondDefenseNodeState
		gates      [8]int
		score      float64
	}
	var current, next [8]branch
	current[0] = branch{motion: newDemoMotionForecast(w), first: -1}
	count := 1
	var ordered [ActorPoolCapacity]*WorldActor
	nodeCount := 0
	for _, actor := range w.orderedMovingActors(&ordered) {
		if actor.Active && actor.secondNode != nil && nodeCount < len(current[0].nodes) {
			current[0].nodes[nodeCount] = *actor.secondNode
			current[0].nodes[nodeCount].Collision = actor.Collision
			nodeCount++
		}
	}
	if nodeCount == 0 {
		return demoSecondArenaBeamOriginal(w, p, fallback)
	}
	current[0].gates = w.secondGateCounters
	if len(p.secondArenaScratch) < horizon*ActorPoolCapacity {
		p.secondArenaScratch = make([]demoSecondDefenseView, horizon*ActorPoolCapacity)
	}
	views := p.secondArenaScratch
	// Defense paths update only whole Y by the common camera delta. Predict
	// their path work once; candidate camera histories add a rigid translation.
	var zeroDeltas [8]int
	members, valid := demoSecondDefenseForecast(w, horizon, zeroDeltas, views)
	if !valid {
		return fallback
	}
	for depth := 0; depth < horizon; depth++ {
		nextCount := 0
		for index := 0; index < count; index++ {
			for action, input := range demoDirections {
				b := current[index]
				prefix := thirdMiddlePlayerBounds(w, b.motion.player)
				contact := false
				for _, node := range b.nodes[:nodeCount] {
					if !node.Destroyed && prefix.Intersects(node.Collision) {
						contact = true
					}
				}
				contactShift := b.actorShift
				nodeScroll := b.motion.scroll
				nodeCamera := nodeScroll.Y
				if depth == 0 {
					b.actorShift = w.ScrollDelta
				} else {
					b.actorShift += b.motion.scroll.ActualStep
				}
				if !b.motion.advance(w, input) {
					continue
				}
				if b.first < 0 {
					b.first = action
				}
				player := b.motion.player
				for i := range b.nodes[:nodeCount] {
					event := b.nodes[i].Advance(SecondDefenseNodeInput{Frame: w.Frame + uint64(depth) + 1, PlayerX: player.X, ScrollY: nodeCamera, MaximumScrollY: nodeScroll.Maximum, Remaining: w.secondDefenseRemaining, DefenseFlags: w.secondScheduler.DefenseFlags, BackwardScroll: w.secondBackward}, &b.gates)
					nodeScroll.Maximum = event.MaximumScrollY
				}
				nodeScroll.Advance(player.ScrollStep, w.BaseScrollStep, input.Down)
				b.motion.scroll = nodeScroll
				b.score += float64(absDemo(player.X-x) + absDemo(player.Y-y)*2)
				bounds := prefix
				if depth == 0 {
					for _, actor := range w.orderedMovingActors(&ordered) {
						if actor.Active && actor.secondSegment != nil && bounds.Intersects(actor.Collision) {
							contact = true
						}
					}
				} else {
					for _, v := range views[(depth-1)*members : depth*members] {
						other := v.Bounds
						other.Top, other.Bottom = other.Top+contactShift, other.Bottom+contactShift
						if v.Active && bounds.Intersects(other) {
							contact = true
						}
					}
				}
				if contact {
					b.score += 1000000
				}
				for _, shot := range w.Projectiles {
					sx, sy, active := demoProjectilePosition(w, shot, depth+1, w.ScrollDelta)
					if active && sx >= bounds.Left-3 && sx <= bounds.Right+3 && sy >= bounds.Top-3 && sy <= bounds.Bottom+3 {
						b.score += 1000000
					}
				}
				pos := nextCount
				if pos == len(next) {
					if b.score >= next[pos-1].score {
						continue
					}
					pos--
				} else {
					nextCount++
				}
				for pos > 0 && b.score < next[pos-1].score {
					next[pos] = next[pos-1]
					pos--
				}
				next[pos] = b
			}
		}
		if nextCount == 0 {
			break
		}
		current, count = next, nextCount
	}
	if current[0].first >= 0 {
		return demoDirections[current[0].first]
	}
	return fallback
}

// Preserve the baseline stream beam outside the clear node-anticipation scope.
func demoSecondArenaBeamOriginal(w *World, p *DemoPilot, fallback MotionInput) MotionInput {
	x, y, nodeVisible, found := p.secondDefenseTarget(w)
	if !found {
		return fallback
	}
	for _, actor := range w.Actors {
		if !actor.Active || actor.secondSegment == nil {
			continue
		}
		if actor.path == nil {
			return fallback
		}
		for _, command := range actor.path.Commands {
			if command.Kind != "curve" && command.Kind != "end" {
				return fallback
			}
		}
	}
	if w.secondScheduler.DefenseFlags != 3 && nodeVisible {
		y = 176
	}
	const horizon = 8
	type branch struct {
		motion     demoMotionForecast
		first      int
		actorShift int
		score      float64
	}
	var current, next [8]branch
	current[0] = branch{motion: newDemoMotionForecast(w), first: -1}
	count := 1
	if len(p.secondArenaScratch) < horizon*ActorPoolCapacity {
		p.secondArenaScratch = make([]demoSecondDefenseView, horizon*ActorPoolCapacity)
	}
	views := p.secondArenaScratch
	// Defense paths update only whole Y by the common camera delta. Predict
	// their path work once; candidate camera histories add a rigid translation.
	var zeroDeltas [8]int
	members, valid := demoSecondDefenseForecast(w, horizon, zeroDeltas, views)
	if !valid {
		return fallback
	}
	for depth := 0; depth < horizon; depth++ {
		nextCount := 0
		for index := 0; index < count; index++ {
			for action, input := range demoDirections {
				b := current[index]
				if depth == 0 {
					b.actorShift = w.ScrollDelta
				} else {
					b.actorShift += b.motion.scroll.ActualStep
				}
				if !b.motion.advance(w, input) {
					continue
				}
				if b.first < 0 {
					b.first = action
				}
				player := b.motion.player
				b.score += float64(absDemo(player.X-x) + absDemo(player.Y-y)*2)
				bounds := thirdMiddlePlayerBounds(w, player)
				for _, v := range views[depth*members : (depth+1)*members] {
					other := v.Bounds
					other.Top, other.Bottom = other.Top+b.actorShift, other.Bottom+b.actorShift
					if v.Active && bounds.Intersects(other) {
						b.score += 1000000
					}
				}
				for _, shot := range w.Projectiles {
					sx, sy, active := demoProjectilePosition(w, shot, depth+1, w.ScrollDelta)
					if active && sx >= bounds.Left-3 && sx <= bounds.Right+3 && sy >= bounds.Top-3 && sy <= bounds.Bottom+3 {
						b.score += 1000000
					}
				}
				pos := nextCount
				if pos == len(next) {
					if b.score >= next[pos-1].score {
						continue
					}
					pos--
				} else {
					nextCount++
				}
				for pos > 0 && b.score < next[pos-1].score {
					next[pos] = next[pos-1]
					pos--
				}
				next[pos] = b
			}
		}
		if nextCount == 0 {
			break
		}
		current, count = next, nextCount
	}
	if current[0].first >= 0 {
		return demoDirections[current[0].first]
	}
	return fallback
}
