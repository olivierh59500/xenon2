package engine

type fourthMiddleRightPhase struct{ stage, target int }
type fourthMiddleRightPose struct {
	frame                              uint64
	player                             PlayerMotionState
	camera, maximum, deviation, shield int
	random                             RandomState
	rewind                             TerrainRewind
	upper                              [2]uint16
}
type fourthMiddleRightPlan struct {
	valid, alive            bool
	shield, distance, count int
	before, after           [6]fourthMiddleRightPose
	phase                   [6]fourthMiddleRightPhase
	input                   [6]Input
}
type fourthMiddleRightPilot struct {
	phase         fourthMiddleRightPhase
	world         *World
	plan          fourthMiddleRightPlan
	at            int
	pending       bool
	forecast, aim WorldForecast
}

func fourthMiddleRightState(w *World) fourthMiddleRightPose {
	return fourthMiddleRightPose{w.Frame, w.Player, w.ScrollY, w.MaximumScrollY, w.ScrollDeviationPasses, w.Equipment.Shield, w.RandomState(), w.Rewind, [2]uint16{w.FourthMiddle.Parts[18].Health, w.FourthMiddle.Parts[16].Health}}
}
func fourthMiddleRightGeometry(w *World, index int) (low, high, worldGoal int) {
	shot := CannonBallMotion{X: WeaponMountOffset[0].X, Y: WeaponMountOffset[0].Y - 11}
	shot.Advance(false)
	sprite := w.Weapons.animations["cannon-ball"].Animation.Frames[0].Sprite
	rect := w.Weapons.actorRect(sprite, shot.X, shot.Y)
	worldGoal = w.fourthMiddleArt.Components[18].InitialWorldY - (rect.Top+rect.Bottom)/2
	actor := w.fourthMiddleActors[18]
	low, high = actor.Collision.Left-rect.Right, actor.Collision.Right-rect.Left
	// The same complete footprint must fit along the actual facade, rather
	// than using the narrower animated player-hit rectangle as terrain width.
	low = max(low, 224-w.Level.PlayerStencil.OriginOffsetX)
	for low <= high && w.Coverage.Touches(low, 176, worldGoal-176, *w.Level.PlayerStencil) {
		low++
	}
	return low, high, worldGoal
}

func (phase *fourthMiddleRightPhase) intent(w *World, aimForecast *WorldForecast) Input {
	phase.target = 18
	if w.FourthMiddle.Parts[18].Disabled {
		return Input{}
	}
	low, high, goal := fourthMiddleRightGeometry(w, 18)
	if low > high {
		return Input{}
	}
	align := func() MotionInput {
		if w.Player.X < low {
			return MotionInput{Right: true}
		}
		if w.Player.X > high {
			return MotionInput{Left: true}
		}
		if w.Player.Inertia > 1 {
			return MotionInput{Left: true}
		}
		if w.Player.Inertia < -1 {
			return MotionInput{Right: true}
		}
		return MotionInput{}
	}
	for transitions := 0; transitions < 4; transitions++ {
		switch phase.stage {
		case 0:
			motion := align()
			if motion != (MotionInput{}) {
				return Input{Motion: motion}
			}
			phase.stage = 1
		case 1:
			if w.Player.X < low || absDemo(w.Player.Inertia) > 1 {
				phase.stage = 0
				continue
			}
			if w.Player.Y >= 176 {
				phase.stage = 2
				continue
			}
			return Input{Motion: MotionInput{Down: true}}
		case 2:
			if w.Player.X < low || absDemo(w.Player.Inertia) > 1 {
				phase.stage = 0
				continue
			}
			if w.Player.Y+w.ScrollY >= goal {
				phase.stage = 3
				continue
			}
			return Input{Motion: MotionInput{Down: true}}
		case 3:
			motion := align()
			worldY := w.Player.Y + w.ScrollY
			motion.Up = worldY > goal && w.Player.Y > 16
			motion.Down = worldY < goal
			fire, _ := fourthMiddleCannonShotOpportunity(w, motion, aimForecast, 3)
			return Input{Motion: motion, Fire: fire}
		}
	}
	return Input{}
}

func (p *fourthMiddleRightPilot) evaluate(w *World, held *MotionInput, fire bool) fourthMiddleRightPlan {
	plan := fourthMiddleRightPlan{valid: true, shield: w.Equipment.Shield}
	if err := p.forecast.Load(w); err != nil {
		return fourthMiddleRightPlan{}
	}
	phase := p.phase
	for pass := 0; pass < 6; pass++ {
		s := p.forecast.State()
		plan.before[pass] = fourthMiddleRightState(s)
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
			return fourthMiddleRightPlan{}
		}
		s = p.forecast.State()
		plan.count = pass + 1
		plan.shield = min(plan.shield, result.Shield)
		plan.after[pass], plan.phase[pass] = fourthMiddleRightState(s), phase
		if s.Rewind.Timer != 0 || s.Coverage.Touches(s.Player.X, s.Player.Y, s.ScrollY, *s.Level.PlayerStencil) {
			plan.valid = false
			return plan
		}
		if result.Boundary != ForecastRunning {
			break
		}
	}
	s := p.forecast.State()
	plan.alive = s.PlayerAlive
	low, high, goal := fourthMiddleRightGeometry(s, 18)
	x := (low + high) / 2
	y := 176
	plan.distance = absDemo(s.Player.X - x)
	if phase.stage == 0 {
		y = 16
	} else {
		plan.distance += absDemo(s.Player.Y + s.ScrollY - goal)
	}
	plan.distance += absDemo(s.Player.Y - y)
	return plan
}
func (p *fourthMiddleRightPilot) Input(w *World) Input {
	if p.world != nil && p.world != w || p.plan.count > 0 && w.Frame < p.plan.before[0].frame {
		p.phase = fourthMiddleRightPhase{}
		p.world = nil
		p.plan = fourthMiddleRightPlan{}
		p.at = 0
		p.pending = false
	}
	state := fourthMiddleRightState(w)
	if p.world == w && p.pending && p.at > 0 && p.at <= p.plan.count {
		if state == p.plan.before[p.at-1] {
			return p.plan.input[p.at-1]
		}
		if state == p.plan.after[p.at-1] {
			p.phase = p.plan.phase[p.at-1]
			p.pending = false
		}
	}
	if p.world == w && !p.pending && p.at < p.plan.count && state == p.plan.before[p.at] {
		i := p.plan.input[p.at]
		p.at++
		p.pending = true
		return i
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
