package engine

type fourthMiddleCorePose struct {
	frame          uint64
	player         PlayerMotionState
	scroll, shield int
	random         RandomState
	rewind         TerrainRewind
}

type fourthMiddleCorePlan struct {
	valid, alive            bool
	shield, distance, count int
	before                  [6]fourthMiddleCorePose
	inputs                  [6]Input
}

type fourthMiddleCorePilot struct {
	forecast WorldForecast
	terminal fourthMiddleNextInputViability
	world    *World
	plan     fourthMiddleCorePlan
	at       int
}

func fourthMiddleCoreState(w *World) fourthMiddleCorePose {
	return fourthMiddleCorePose{w.Frame, w.Player, w.ScrollY, w.Equipment.Shield, w.RandomState(), w.Rewind}
}

func fourthMiddleCoreGeometry(w *World) (aim, goal int) {
	actor := w.fourthMiddleActors[4]
	if actor == nil || !actor.Active || actor.Collision.Empty() {
		return w.Player.X, w.ScrollY + w.Player.Y
	}
	aim = (actor.Collision.Left + actor.Collision.Right) / 2
	top := 0
	for _, sprite := range w.Level.Ships.Atlas.Sprites {
		if sprite.Collision != nil {
			top = min(top, sprite.Collision.Y)
		}
	}
	for index := 0; index <= 5; index++ {
		body := w.fourthMiddleActors[index]
		if body != nil && body.Active && !body.Collision.Empty() {
			goal = max(goal, body.Collision.Bottom+w.ScrollY-top+1)
		}
	}
	for goal <= fourthMiddleRearWorldLimit && w.Coverage.Touches(aim, goal-w.ScrollY, w.ScrollY, *w.Level.PlayerStencil) {
		goal++
	}
	return aim, goal
}

// The native head stays at world2536 while four linked pieces each take
// Budget sine substeps. The largest companion bottom is31 and the ship top
// is-10, so a body-safe world floor is2536+4*Budget+31+10+1.
// The existing six-pass phase may exceed the arena's native rear limit.
func fourthMiddleCorePhaseRetreat(w *World) bool {
	head := w.FourthMiddle.Parts[0]
	counter, direction, budget := int(head.Counter), int(head.Direction), head.Arc.Budget
	if (2536+31+10+1)+4*budget > fourthMiddleRearWorldLimit {
		return true
	}
	for pass := 0; pass < 6; pass++ {
		if counter == 0 {
			counter, direction = 2, 2
		} else {
			counter += direction
			if counter >= 24 {
				direction = -direction
			}
		}
		budget = counter + 4
		if (2536+31+10+1)+4*budget > fourthMiddleRearWorldLimit {
			return true
		}
	}
	return false
}

func fourthMiddleCoreTargets(w *World) (aim, goal int) {
	aim, goal = fourthMiddleCoreGeometry(w)
	if !w.FourthMiddle.Defeated && (goal > fourthMiddleRearWorldLimit || fourthMiddleCorePhaseRetreat(w)) {
		return 80, fourthMiddleRearWorldLimit
	}
	return aim, goal
}

func fourthMiddleCoreIntent(w *World) Input {
	if w.FourthMiddle.Defeated {
		return Input{}
	}
	aim, goal := fourthMiddleCoreTargets(w)
	if goal > fourthMiddleRearWorldLimit {
		return Input{Fire: true}
	}
	motion, _ := fourthMiddleCoastMotion(w.Player, aim)
	worldY := w.ScrollY + w.Player.Y
	motion.Up = worldY > goal && w.Player.Y > 16
	motion.Down = worldY < goal
	return Input{Motion: motion, Fire: true}
}

func (p *fourthMiddleCorePilot) evaluate(w *World, held *MotionInput) fourthMiddleCorePlan {
	plan := fourthMiddleCorePlan{valid: true, shield: w.Equipment.Shield}
	if err := p.forecast.Load(w); err != nil {
		return fourthMiddleCorePlan{}
	}
	for pass := 0; pass < 6; pass++ {
		state := p.forecast.State()
		plan.before[pass] = fourthMiddleCoreState(state)
		input := fourthMiddleCoreIntent(state)
		if held != nil {
			input.Motion = *held
		}
		plan.inputs[pass] = input
		for range 3 {
			p.forecast.AdvancePALTick()
		}
		result, err := p.forecast.Advance(input)
		if err != nil {
			return fourthMiddleCorePlan{}
		}
		state = p.forecast.State()
		plan.count = pass + 1
		plan.shield = min(plan.shield, result.Shield)
		if fourthMiddleCoreTerminalUnsafe(state) || state.Rewind.Timer != 0 || state.Coverage.Touches(state.Player.X, state.Player.Y, state.ScrollY, *state.Level.PlayerStencil) {
			plan.valid = false
			return plan
		}
		if result.Boundary != ForecastRunning || state.FourthMiddle.Defeated {
			break
		}
	}
	state := p.forecast.State()
	if fourthMiddleCoreTerminalUnsafe(state) {
		plan.valid = false
		return plan
	}
	plan.alive = state.PlayerAlive
	if plan.alive && !state.FourthMiddle.Defeated {
		viable, err := p.terminal.check(state, true)
		if err != nil || !viable.safe {
			plan.valid = false
			return plan
		}
	}
	aim, goal := fourthMiddleCoreTargets(state)
	plan.distance = absDemo(state.Player.X-aim) + absDemo(state.Player.Y+state.ScrollY-goal) + absDemo(state.Player.Y-176)
	return plan
}

func (p *fourthMiddleCorePilot) Input(w *World) Input {
	if p.world != nil && p.world != w || p.plan.count > 0 && w.Frame < p.plan.before[0].frame {
		p.world = nil
		p.plan = fourthMiddleCorePlan{}
		p.at = 0
	}
	// Replan every new source pass; identical repeated samples reuse only
	// the first input for this exact before-state.

	if p.world == w {
		pose := fourthMiddleCoreState(w)
		if p.at > 0 && p.at <= p.plan.count && p.plan.before[p.at-1] == pose {
			return p.plan.inputs[p.at-1]
		}

	}
	p.plan.count = 0
	best := p.evaluate(w, nil)
	found := best.valid
	for _, motion := range demoDirections {
		candidate := p.evaluate(w, &motion)
		if !candidate.valid {
			continue
		}
		if !found || candidate.alive && !best.alive || candidate.alive == best.alive && (candidate.shield > best.shield || candidate.shield == best.shield && candidate.distance < best.distance) {
			best, found = candidate, true
		}
	}
	if !found {
		return fourthMiddleCoreIntent(w)
	}
	p.world, p.plan, p.at = w, best, 1
	return best.inputs[0]
}

// Moving contact precedes the next input's movement. A damaging published body
// overlap with the fresh player bank prefix is therefore already unavoidable.
// This constrains the exposed fourth-middle core controller's endpoint.
func fourthMiddleCoreTerminalUnsafe(w *World) bool {
	if w == nil || w.Level.Number != 4 || w.FourthMiddle == nil || w.FourthMiddle.Defeated || w.FourthMiddle.OuterTargets != 0 || !w.PlayerAlive || w.GameOver || w.Ready || w.ScreenClearFrames != 0 || w.stepContinuation.active || w.Dive.Phase != 0 || w.Equipment.ShadesFrames > 0 {
		return false
	}
	prefix := thirdMiddlePlayerBounds(w, w.Player)
	var order [ActorPoolCapacity]*WorldActor
	for _, actor := range w.orderedMovingActors(&order) {
		if !actor.Active || actor.ActorList != "moving" || !actor.Collision.Intersects(prefix) {
			continue
		}
		index := actor.fourthIndex - 1
		if actor.fourthFinal || index < 0 || index > 5 || actor.part == nil {
			return false
		}
		damage := ApplyShieldDamage(w.Equipment.Shield, ContactDamage(actor.part.StrongHealth), w.Equipment.Protection, w.InvulnerableFrames != 0 || w.PendingExitDrops != 0 || w.Cheats.InfiniteEnergy)
		return damage.Applied && damage.Lost > 0
	}
	return false
}
