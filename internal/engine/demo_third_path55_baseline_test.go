package engine

func (p *PresentationPilot) normalInputWithoutPath55Preparation(w *World) Input {
	if w == nil || w.GameOver || !w.PlayerAlive {
		return Input{}
	}
	if w.Ready {
		return Input{Fire: true}
	}
	if p.world != w || w.Frame < p.frame {
		*p = PresentationPilot{PALRefreshes: p.PALRefreshes, world: w, frame: w.Frame, decisionAt: w.Frame + 3, planner: DemoPilot{practicedRoute: true}}
	}
	p.frame = w.Frame
	base := p.planner.NormalInput(w)
	input := base
	if w.Level.Number == 2 && w.secondScheduler != nil && !w.secondMiddleReleased && w.ScrollY >= 2512 && w.ScrollY <= 2896 {
		input.Motion = demoSecondArenaBeam(w, &p.planner, base.Motion)
	}
	retreat := p.planner.retreatGoal != 0
	if !retreat && w.Rewind.Timer == 0 && !presentationSpecialist(w) {
		if w.Frame >= p.decisionAt {
			goal := presentationChooseGoal(w)
			if goal.actor != p.goal.actor || goal.bonus != p.goal.bonus {
				p.acquiredAt = w.Frame
			}
			p.goal = goal
			// About half a second of attention at the native 50/3 game passes.
			p.decisionAt = w.Frame + 9
		}
		if p.goal.valid(w) && w.Frame >= p.acquiredAt+2 {
			input.Motion = p.tacticalMotion(w, base.Motion)
		}
	}
	if w.Level.Number == 3 && w.ThirdMiddle == nil && w.Checkpoint.ScrollY > 4032 && w.Rewind.Timer == 0 {
		x, y := 250, 120
		if p.goal.valid(w) {
			x, y = p.goal.x, p.goal.y
		}
		if preparedX, preparedY, found := thirdChainPreparation(w); found {
			x, y = preparedX, preparedY
		}
		// Known opening formations need turns within the reaction horizon;
		// holding one direction can collide after a path changes heading.
		input.Motion = demoRouteMotionWithClearance(w, x, w.ScrollY+y, y, 4)
	}
	input.Fire = p.selectiveFireForMotion(w, input.Motion)
	if w.blockedFireUntilRelease || w.Dive.Phase != 0 {
		input.Fire = false
	}
	input = p.forecastOpeningGuard(w, input)
	input = p.forecastThirdMiddleInput(w, input)
	return input
}
