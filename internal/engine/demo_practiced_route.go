package engine

// Route protection evaluates an initial dodge followed by the known terrain
// route. A held-direction test alone misses the consequences of rejoining a
// narrow passage while a cannon volley or formation is arriving.
type expertRouteGuard struct {
	forecast   WorldForecast
	navigation demoNavigation
	plan       retainedGuardPlan
}

func (p *PresentationPilot) forecastPracticedRoute(w *World, planned Input) (Input, bool) {
	if !thirdCorridorBranchWindow(w) {
		return planned, false
	}
	if p.routeGuard == nil {
		p.routeGuard = &expertRouteGuard{}
	}
	g := p.routeGuard
	pal := thirdMiddlePALRefreshes(p.PALRefreshes)
	if input, ok := g.continuePlan(w, pal); ok {
		return input, true
	}
	p.thirdCorridorRecoveryNeeded(w)
	sequence, useSequence := p.planner.nativeMotion.guardCommands(w, planned.Motion)
	if p.planner.navigation == nil || len(p.planner.navigation.path) == 0 {
		return planned, false
	}
	path := p.planner.navigation.path
	baseline, _, _, ok := g.evaluate(w, planned, sequence, useSequence, MotionInput{}, 0, pal, path)
	if !ok {
		return planned, false
	}
	if baseline.alive && baseline.damage == 0 && baseline.contacts == 0 {
		return planned, true
	}
	best := baseline
	var selected [6]Input
	var keys [7]retainedGuardKey
	for _, motion := range demoDirections {
		candidate, inputs, nextKeys, ok := g.evaluate(w, planned, sequence, useSequence, motion, 6, pal, path)
		if ok && candidate.better(best) {
			best, selected, keys = candidate, inputs, nextKeys
		}
	}
	if best.alive && (!baseline.alive || best.damage < baseline.damage || best.contacts < baseline.contacts) && keys[0].frame == w.Frame {
		g.plan.world, g.plan.before, g.plan.input, g.plan.at, g.plan.count, g.plan.pal = w, keys, selected, 1, 6, pal
		return selected[0], true
	}
	return planned, false
}

func (g *expertRouteGuard) evaluate(w *World, planned Input, sequence []MotionInput, useSequence bool, motion MotionInput, held, pal int, path []demoNavPoint) (expertEncounterScore, [6]Input, [7]retainedGuardKey, bool) {
	var inputs [6]Input
	var keys [7]retainedGuardKey
	if err := g.forecast.Load(w); err != nil {
		return expertEncounterScore{}, inputs, keys, false
	}
	keys[0] = retainedGuardStateKey(w)
	score := expertEncounterScore{alive: true}
	horizon := 24
	if useSequence {
		// A verified commitment ends at its own waypoint. Beyond that point
		// the live controller will choose another leg; a different local
		// continuation must not veto the commitment's final safe turn.
		horizon = min(horizon, max(6, len(sequence)))
	}
	for pass := 0; pass < horizon; pass++ {
		q := g.forecast.State()
		x, y := g.waypoint(q, path)
		input := Input{Motion: expertEncounterSteering(q, x, y, true)}
		if pass == 0 {
			input = planned
		}
		if pass < len(sequence) && useSequence {
			input.Motion = sequence[pass]
		}
		if pass < held {
			input.Motion = motion
		}
		input.Fire = !q.blockedFireUntilRelease && q.Dive.Phase == 0 && (presentationShotOpportunityForMotion(q, input.Motion) || presentationAuxiliaryShotOpportunity(q, input.Motion))
		shield := q.Equipment.Shield
		for range pal {
			g.forecast.AdvancePALTick()
		}
		r, err := g.forecast.Advance(input)
		if err != nil {
			return score, inputs, keys, false
		}
		q = g.forecast.State()
		score.damage += max(0, shield-r.Shield)
		if q.Rewind.Timer != 0 {
			score.contacts++
		}
		if pass < len(inputs) {
			inputs[pass], keys[pass+1] = input, retainedGuardStateKey(q)
		}
		if !r.Alive {
			score.alive = false
		}
		if r.Boundary != ForecastRunning {
			break
		}
	}
	q := g.forecast.State()
	score.value = (q.Score-w.Score)*4 + (q.Money-w.Money)*20 + (w.ScrollY-q.ScrollY)*20
	return score, inputs, keys, true
}

// Follow the already prepared route, checking its edges against the branch's
// current map. Re-running a full motion search at every forecast pass is both
// unnecessary and prohibitively expensive on a phone.
func (g *expertRouteGuard) waypoint(w *World, path []demoNavPoint) (int, int) {
	g.navigation.refresh(w)
	x, y := w.Player.X, w.Player.Y+w.ScrollY
	nearest, distance := 0, int(^uint(0)>>1)
	for index, point := range path {
		if d := absDemo(point.x-x) + absDemo(point.y-y); d < distance {
			nearest, distance = index, d
		}
	}
	for index := min(len(path)-1, nearest+8); index >= nearest; index-- {
		point := path[index]
		if g.navigation.clearSegment(x, y, point) {
			return point.x, point.y
		}
	}
	return x, y
}

func (g *expertRouteGuard) continuePlan(w *World, pal int) (Input, bool) {
	plan := &g.plan
	if plan.world != w || plan.pal != pal || plan.count == 0 {
		return Input{}, false
	}
	key, index := retainedGuardStateKey(w), plan.at
	if index > 0 && key == plan.before[index-1] {
		index--
	}
	if index >= plan.count || key != plan.before[index] || plan.forecast.Load(w) != nil {
		plan.count = 0
		return Input{}, false
	}
	for pass := index; pass < plan.count; pass++ {
		for range pal {
			plan.forecast.AdvancePALTick()
		}
		r, err := plan.forecast.Advance(plan.input[pass])
		if err != nil || !r.Alive || retainedGuardStateKey(plan.forecast.State()) != plan.before[pass+1] {
			plan.count = 0
			return Input{}, false
		}
	}
	if plan.at == index {
		plan.at++
	}
	return plan.input[index], true
}
