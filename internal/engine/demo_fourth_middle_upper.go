package engine

type fourthMiddleUpperPhase struct{ stage, target int }
type fourthMiddleUpperPose struct {
	frame                              uint64
	player                             PlayerMotionState
	camera, maximum, deviation, shield int
	random                             RandomState
	rewind                             TerrainRewind
	upper                              [2]uint16
}
type fourthMiddleUpperPlan struct {
	valid, alive            bool
	shield, distance, count int
	before, after           [6]fourthMiddleUpperPose
	phase                   [6]fourthMiddleUpperPhase
	input                   [6]Input
}
type fourthMiddleUpperPilot struct {
	phase         fourthMiddleUpperPhase
	world         *World
	plan          fourthMiddleUpperPlan
	at            int
	pending       bool
	forecast, aim WorldForecast
}

func fourthMiddleUpperState(w *World) fourthMiddleUpperPose {
	return fourthMiddleUpperPose{w.Frame, w.Player, w.ScrollY, w.MaximumScrollY, w.ScrollDeviationPasses, w.Equipment.Shield, w.RandomState(), w.Rewind, [2]uint16{w.FourthMiddle.Parts[17].Health, w.FourthMiddle.Parts[19].Health}}
}
func fourthMiddleUpperGeometry(w *World, index int) (aim, front, reverse int) {
	a := w.fourthMiddleActors[index]
	aim = (a.Collision.Left + a.Collision.Right) / 2
	front = 2288
	for front > 2176 && w.Coverage.Touches(aim, front-w.ScrollY, w.ScrollY, *w.Level.PlayerStencil) {
		front--
	}
	topCamera := front - 16
	upPasses := (176 - 16 + (3 + w.Equipment.SpeedTier) - 1) / (3 + w.Equipment.SpeedTier)
	horizontal := 3
	if w.Equipment.SpeedTier == 1 {
		horizontal = 6
	}
	if w.Equipment.SpeedTier >= 2 {
		horizontal = 9
	}
	alignPasses := (absDemo(aim-80) + horizontal - 1) / horizontal
	return aim, front, topCamera + upPasses + alignPasses
}
func (phase *fourthMiddleUpperPhase) intent(w *World, aimForecast *WorldForecast) Input {
	if phase.target != 17 && phase.target != 19 {
		phase.target = 17
	}
	if w.FourthMiddle.Parts[phase.target].Disabled {
		if phase.target == 17 && !w.FourthMiddle.Parts[19].Disabled {
			phase.target, phase.stage = 19, 0
		} else {
			return Input{}
		}
	}
	_, front, reverse := fourthMiddleUpperGeometry(w, phase.target)
	for transitions := 0; transitions < 6; transitions++ {
		switch phase.stage {
		case 0:
			if motion, settled := fourthMiddleCoastMotion(w.Player, 80); !settled {
				return Input{Motion: motion}
			}
			phase.stage = 1
		case 1:
			if motion, settled := fourthMiddleCoastMotion(w.Player, 80); !settled {
				return Input{Motion: motion}
			}
			if w.Player.Y >= 176 && w.ScrollY >= reverse {
				phase.stage = 2
				continue
			}
			return Input{Motion: MotionInput{Down: true}}
		case 2:
			if w.Player.Y <= 16 {
				phase.stage = 3
				continue
			}
			return Input{Motion: MotionInput{Up: true}}
		case 3:
			if w.Player.Y > 16 || w.ScrollY > front-16 {
				return Input{Motion: MotionInput{Up: true}}
			}
			phase.stage = 4
		case 4:
			if w.Player.Y > 16 {
				return Input{Motion: MotionInput{Up: true}}
			}
			stable, _, _, valid := fourthMiddleStableAim(w, phase.target)
			if !valid {
				return Input{}
			}
			motion, settled := fourthMiddleCoastMotion(w.Player, stable)
			if settled {
				phase.stage = 5
				continue
			}
			return Input{Motion: motion}
		case 5:
			if w.ScrollY <= w.MinimumScrollY {
				phase.stage = 0
				continue
			}
			stable, _, _, valid := fourthMiddleStableAim(w, phase.target)
			if !valid {
				return Input{}
			}
			if _, settled := fourthMiddleCoastMotion(w.Player, stable); !settled {
				phase.stage = 4
				continue
			}
			motion := MotionInput{Up: w.Player.Y > 16}
			fire, _ := fourthMiddleRearShotOpportunity(w, motion, aimForecast, 3)
			return Input{Motion: motion, Fire: fire}
		}
	}
	return Input{}
}
func (p *fourthMiddleUpperPilot) evaluate(w *World, held *MotionInput, fire bool) fourthMiddleUpperPlan {
	plan := fourthMiddleUpperPlan{valid: true, shield: w.Equipment.Shield}
	if err := p.forecast.Load(w); err != nil {
		return fourthMiddleUpperPlan{}
	}
	phase := p.phase
	for pass := 0; pass < 6; pass++ {
		s := p.forecast.State()
		plan.before[pass] = fourthMiddleUpperState(s)
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
			return fourthMiddleUpperPlan{}
		}
		s = p.forecast.State()
		plan.count = pass + 1
		plan.shield = min(plan.shield, result.Shield)
		plan.after[pass], plan.phase[pass] = fourthMiddleUpperState(s), phase
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
	aim, front, reverse := fourthMiddleUpperGeometry(s, phase.target)
	x, y := 80, 176
	switch phase.stage {
	case 1:
		y = 176
		plan.distance = absDemo(s.ScrollY - reverse)
	case 2, 3:
		y = 16
		plan.distance = max(0, s.ScrollY-(front-16))
	case 4, 5:
		x, y = aim, 16
	}
	plan.distance += absDemo(s.Player.X-x) + absDemo(s.Player.Y-y)
	return plan
}
func (p *fourthMiddleUpperPilot) Input(w *World) Input {
	if p.world != nil && p.world != w || p.plan.count > 0 && w.Frame < p.plan.before[0].frame {
		p.phase = fourthMiddleUpperPhase{}
		p.world = nil
		p.plan = fourthMiddleUpperPlan{}
		p.at = 0
		p.pending = false
	}
	state := fourthMiddleUpperState(w)
	if p.world == w && p.pending && p.at > 0 && p.at <= p.plan.count {
		if state == p.plan.before[p.at-1] {
			return p.plan.input[p.at-1]
		}
		if state == p.plan.after[p.at-1] {
			p.phase = p.plan.phase[p.at-1]
			p.pending = false
		}
	}
	if p.phase.stage < 4 && p.world == w && !p.pending && p.at < p.plan.count && state == p.plan.before[p.at] {
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
