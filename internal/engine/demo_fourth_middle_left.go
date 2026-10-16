package engine

// The native live core renews maximum camera2480; the player bottom is176.
const fourthMiddleRearWorldLimit = 2480 + 176

type fourthMiddleLeftPhase struct {
	stage, target, goal, fireGoal int
	crossing                      bool
}
type fourthMiddleLeftPose struct {
	frame                              uint64
	player                             PlayerMotionState
	camera, maximum, deviation, shield int
	random                             RandomState
	rewind                             TerrainRewind
	upper                              [2]uint16
}
type fourthMiddleLeftPlan struct {
	valid, alive            bool
	shield, distance, count int
	before, after           [6]fourthMiddleLeftPose
	phase                   [6]fourthMiddleLeftPhase
	input                   [6]Input
}
type fourthMiddleLeftPilot struct {
	phase         fourthMiddleLeftPhase
	world         *World
	plan          fourthMiddleLeftPlan
	at            int
	pending       bool
	forecast, aim WorldForecast
	terminal      fourthMiddleNextInputViability
}

func fourthMiddleLeftState(w *World) fourthMiddleLeftPose {
	return fourthMiddleLeftPose{w.Frame, w.Player, w.ScrollY, w.MaximumScrollY, w.ScrollDeviationPasses, w.Equipment.Shield, w.RandomState(), w.Rewind, [2]uint16{w.FourthMiddle.Parts[16].Health, w.FourthMiddle.Parts[4].Health}}
}
func fourthMiddleLeftGeometry(w *World) (aim, floor, body int) {
	target := w.fourthMiddleActors[16]
	aim = (target.Collision.Left + target.Collision.Right) / 2
	floor = w.ScrollY + target.Collision.Bottom + 72
	for floor <= fourthMiddleRearWorldLimit && w.Coverage.Touches(aim, floor-w.ScrollY, w.ScrollY, *w.Level.PlayerStencil) {
		floor++
	}
	top := 0
	for _, sprite := range w.Level.Ships.Atlas.Sprites {
		if sprite.Collision != nil && sprite.Collision.Y < top {
			top = sprite.Collision.Y
		}
	}
	body = floor
	for i := 0; i <= 5; i++ {
		actor := w.fourthMiddleActors[i]
		if actor != nil && actor.Active && !actor.Collision.Empty() {
			body = max(body, actor.Collision.Bottom+w.ScrollY-top+1)
		}
	}
	return aim, floor, body
}

func (phase *fourthMiddleLeftPhase) intent(w *World, aimForecast *WorldForecast) Input {
	phase.target = 16
	if w.FourthMiddle.Parts[16].Disabled {
		return Input{}
	}
	aim, floor, body := fourthMiddleLeftGeometry(w)
	if body <= fourthMiddleRearWorldLimit && phase.stage < 2 {
		phase.goal = max(phase.goal, body)
	}
	if phase.goal == 0 || floor > fourthMiddleRearWorldLimit {
		return Input{}
	}
	if w.Player.Y+w.ScrollY >= phase.goal {
		phase.crossing = true
		phase.stage = max(phase.stage, 1)
	}
	if !phase.crossing || phase.stage < 2 && w.Player.Y+w.ScrollY < phase.goal-w.BaseScrollStep {
		// Only the native one-base-pass max-camera oscillation may retain a
		// horizontal crossing below its goal. Recover larger dodge/goal gaps.
		return Input{Motion: MotionInput{Down: true}}
	}
	if absDemo(w.Player.X-aim) <= 6 {
		phase.stage = 2
	}
	motionFor := func(goal int) MotionInput {
		worldY := w.Player.Y + w.ScrollY
		return MotionInput{Left: w.Player.X > aim+6, Right: w.Player.X < aim-6, Down: worldY < goal, Up: worldY > goal && w.Player.Y > 16}
	}
	goal := phase.goal
	if phase.stage == 2 {
		if phase.fireGoal == 0 && w.Player.Y+w.ScrollY >= body && !fourthMiddleLeftBodyTerminalUnsafe(w) {
			possible, _ := presentationGuardianShotOpportunity(w, motionFor(phase.goal), aimForecast, 3)
			if possible {
				phase.fireGoal = phase.goal
			}
		}
		if phase.fireGoal > 0 {
			goal = max(phase.fireGoal, body)
		} else {
			goal = floor
		}
	}
	motion := motionFor(goal)
	fire, _ := presentationGuardianShotOpportunity(w, motion, aimForecast, 3)
	return Input{Motion: motion, Fire: fire}
}

func (p *fourthMiddleLeftPilot) evaluate(w *World, held *MotionInput, fire bool) fourthMiddleLeftPlan {
	plan := fourthMiddleLeftPlan{valid: true, shield: w.Equipment.Shield}
	if err := p.forecast.Load(w); err != nil {
		return fourthMiddleLeftPlan{}
	}
	phase := p.phase
	for pass := 0; pass < 6; pass++ {
		s := p.forecast.State()
		plan.before[pass] = fourthMiddleLeftState(s)
		input := phase.intent(s, &p.aim)
		if held != nil {
			input.Motion = *held
			input.Fire = fire
		}
		plan.input[pass] = input
		for range 3 {
			p.forecast.AdvancePALTick()
		}
		result, err := p.forecast.Advance(input)
		if err != nil {
			return fourthMiddleLeftPlan{}
		}
		s = p.forecast.State()
		plan.count = pass + 1
		plan.shield = min(plan.shield, result.Shield)
		plan.after[pass], plan.phase[pass] = fourthMiddleLeftState(s), phase
		if fourthMiddleLeftBodyTerminalUnsafe(s) || s.Rewind.Timer != 0 || s.Coverage.Touches(s.Player.X, s.Player.Y, s.ScrollY, *s.Level.PlayerStencil) {
			plan.valid = false
			return plan
		}
		if result.Boundary != ForecastRunning {
			break
		}
	}
	s := p.forecast.State()
	if fourthMiddleLeftBodyTerminalUnsafe(s) {
		plan.valid = false
		return plan
	}
	plan.alive = s.PlayerAlive
	if plan.alive {
		viable, err := p.terminal.check(s, false)
		if err != nil || !viable.safe {
			plan.valid = false
			return plan
		}
	}
	aim, floor, body := fourthMiddleLeftGeometry(s)
	x := w.Player.X
	if phase.crossing {
		x = aim
	}
	goal := phase.goal
	if goal == 0 {
		goal = min(fourthMiddleRearWorldLimit, max(floor, body))
	}
	if phase.stage == 2 {
		if phase.fireGoal > 0 {
			goal = max(phase.fireGoal, body)
		} else {
			goal = floor
		}
	}
	y := 176
	plan.distance = absDemo(s.Player.X - x)
	plan.distance += absDemo(s.Player.Y + s.ScrollY - goal)
	plan.distance += absDemo(s.Player.Y - y)
	return plan
}
func (p *fourthMiddleLeftPilot) Input(w *World) Input {
	if p.world != nil && p.world != w || p.plan.count > 0 && w.Frame < p.plan.before[0].frame {
		p.phase = fourthMiddleLeftPhase{}
		p.world = nil
		p.plan = fourthMiddleLeftPlan{}
		p.at = 0
		p.pending = false
	}
	// Every new source pass replans; only an identical before-state reuses the
	// sampled input. The completed first step commits its predicted phase.
	state := fourthMiddleLeftState(w)
	if p.world == w && p.pending && p.at > 0 && p.at <= p.plan.count {
		if state == p.plan.before[p.at-1] {
			return p.plan.input[p.at-1]
		}
		if state == p.plan.after[p.at-1] {
			p.phase = p.plan.phase[p.at-1]
			p.pending = false
		}
	}
	phase := p.phase
	intent := phase.intent(w, &p.aim)
	best := p.evaluate(w, nil, intent.Fire)
	found := best.valid
	if !(found && best.alive && best.shield >= w.Equipment.Shield) {
		for _, motion := range demoDirections {
			candidate := p.evaluate(w, &motion, intent.Fire)
			if !candidate.valid {
				continue
			}
			if !found || candidate.alive && !best.alive || candidate.alive == best.alive && (candidate.shield > best.shield || candidate.shield == best.shield && candidate.distance < best.distance) {
				best, found = candidate, true
			}
		}
	}

	if !found {
		return intent
	}
	p.world, p.plan, p.at, p.pending = w, best, 1, true
	return best.input[0]
}

func fourthMiddleLeftBodyTerminalUnsafe(w *World) bool {
	if !w.PlayerAlive || w.Dive.Phase != 0 || w.Equipment.ShadesFrames > 0 {
		return false
	}
	prefix := thirdMiddlePlayerBounds(w, w.Player)
	var storage [ActorPoolCapacity]*WorldActor
	for _, actor := range w.orderedMovingActors(&storage) {
		if !actor.Active || actor.ActorList != "moving" || !actor.Collision.Intersects(prefix) {
			continue
		}
		index := actor.fourthIndex - 1
		if index < 0 || index > 5 || actor.part == nil {
			return false
		}
		damage := ApplyShieldDamage(w.Equipment.Shield, ContactDamage(actor.part.StrongHealth), w.Equipment.Protection, w.InvulnerableFrames != 0 || w.PendingExitDrops != 0 || w.Cheats.InfiniteEnergy)
		return damage.Applied && (damage.Lost > 0 || damage.Destroyed)
	}
	return false
}
