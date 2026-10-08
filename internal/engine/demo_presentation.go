package engine

import "math"

// PresentationPilot plays through ordinary controls with a short reaction time,
// committed tactical goals and held firing bursts. The game still owns every
// movement, hit, reward, terrain change and random value.
type PresentationPilot struct {
	fourthFinalNative *fourthFinalNativePilot
	fourthMiddleUpper *fourthMiddleUpperPilot
	fourthMiddleRight *fourthMiddleRightPilot
	fourthMiddleLeft  *fourthMiddleLeftPilot
	fourthMiddleCore  *fourthMiddleCorePilot
	fourthBranch      *fourthOpeningBranchPlan
	// PALRefreshes matches the host's gameplay cadence; zero uses three ticks.
	PALRefreshes            int
	forecast                WorldForecast
	guardianAimForecast     WorldForecast
	middleForecastPolicy    DemoPilot
	middleWorkers           *thirdMiddleForecastWorkers
	planner                 DemoPilot
	world                   *World
	frame                   uint64
	goal                    presentationGoal
	decisionAt, motionUntil uint64
	motion                  MotionInput
	burstUntil, restUntil   uint64
	acquiredAt              uint64
}

type presentationGoal struct {
	x, y    int
	actor   *WorldActor
	bonus   *WorldCollectible
	arrival int
}

func (p *PresentationPilot) NormalInput(w *World) Input {
	if w == nil || w.GameOver || !w.PlayerAlive {
		p.clearFourthOpeningBranch()
		return Input{}
	}
	if w.Ready {
		p.clearFourthOpeningBranch()
		return Input{Fire: true}
	}
	if p.world != w || w.Frame < p.frame {
		*p = PresentationPilot{PALRefreshes: p.PALRefreshes, world: w, frame: w.Frame, decisionAt: w.Frame + 3, planner: DemoPilot{practicedRoute: true}}
	}
	p.frame = w.Frame
	if input, handled := p.fourthFinalNativeInput(w); handled {
		return input
	}
	if input, handled := p.fourthMiddleSpecialistInput(w); handled {
		return input
	}
	if input, retained := p.continueFourthOpeningBranch(w); retained {
		return input
	}
	p.planner.palRefreshes = p.PALRefreshes
	base := p.planner.NormalInput(w)
	input := base
	if w.Level.Number == 2 && w.secondScheduler != nil && !w.secondMiddleReleased && w.ScrollY >= 2512 && w.ScrollY <= 2896 {
		input.Motion = demoSecondArenaBeam(w, &p.planner, base.Motion)
	}
	retreat := p.planner.retreatGoal != 0
	if !retreat && w.Rewind.Timer == 0 && !presentationSpecialist(w) && !p.planner.fourthRearLegOwnsMotion(w, base.Motion) {
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
	if !p.planner.Config.DisableBossAlignment {
		if motion, prepare := thirdPath55Preparation(w); prepare {
			input.Motion = motion
		}
	}
	if motion, prepare := p.thirdFinalEntryPreparation(w); prepare {
		input.Motion = motion
	}
	input.Fire = p.selectiveFireForMotion(w, input.Motion)
	if w.blockedFireUntilRelease || w.Dive.Phase != 0 {
		input.Fire = false
	}
	fireMotion := input.Motion
	input = p.forecastOpeningGuard(w, input)
	if w.Level.Number == 1 && input.Fire && input.Motion != fireMotion {
		// Recheck the final gun position without advancing burst state twice.
		input.Fire = presentationShotOpportunityWithForecast(w, input.Motion, &p.guardianAimForecast, p.PALRefreshes)
	}
	input = p.forecastThirdMiddleInput(w, input)
	return input
}

// Source-specific arena movement retains the already verified controller.
// Trigger decisions still require a live shot opportunity in those arenas.
func presentationSpecialist(w *World) bool {
	return fifthBarrierTarget(w) != nil || fourthCorridorActive(w) ||
		w.Level.Number == 2 && (w.secondScheduler != nil && w.ScrollY >= 2512 && w.ScrollY <= 2896 || w.secondMiddleReleased && w.ScrollY <= 1280) ||
		w.Level.Number == 3 && (w.ThirdMiddle != nil || w.ThirdFinal != nil && !w.ThirdFinal.Defeated && w.ScrollY <= 208) ||
		w.Level.Number == 4 && w.ScrollY <= 176 ||
		w.Level.Number == 1 && (w.FirstMiddle != nil && !w.FirstMiddle.Crossed && w.ScrollY < 3456 || w.FirstGuardian != nil && w.FirstGuardian.Active && !w.FirstGuardian.Defeated)
}

func (g presentationGoal) valid(w *World) bool {
	if g.arrival != 0 {
		return w.ScrollY > g.arrival && w.ScrollY-g.arrival <= 48 && presentationClearPoint(w, g.x, g.y)
	}
	if g.bonus != nil {
		return g.bonus.Active && g.bonus.Y >= 12 && g.bonus.Y <= 172 && g.bonus.Y >= float64(w.Player.Y-65)
	}
	if g.actor != nil {
		_, ok := presentationTargetBounds(w, g.actor)
		return ok && g.actor.Collision.Top < w.Player.Y-12
	}
	return false
}

func presentationChooseGoal(w *World) presentationGoal {
	goal, best := presentationGoal{}, math.Inf(1)
	for _, actor := range w.Actors {
		bounds, ok := presentationTargetBounds(w, actor)
		if !ok || bounds.Bottom < 0 || bounds.Top >= w.Player.Y-12 || bounds.Top > 156 {
			continue
		}
		flight := max(1, min(12, (w.Player.Y-(bounds.Top+bounds.Bottom)/2-6)/9))
		if predicted, supported := demoActorPrediction(w, actor, flight, w.ScrollY-flight*w.BaseScrollStep); supported && predicted.Active && predicted.Visible && !predicted.Bounds.Empty() {
			bounds = predicted.Bounds
		} else {
			dx, dy := int(math.Round(actor.X-actor.PreviousX))*flight, int(math.Round(actor.Y-actor.PreviousY))*flight
			bounds.Left, bounds.Right = bounds.Left+dx, bounds.Right+dx
			bounds.Top, bounds.Bottom = bounds.Top+dy, bounds.Bottom+dy
		}
		x := (bounds.Left + bounds.Right) / 2
		y := max(88, min(152, bounds.Bottom+72))
		x = max(24, min(296, x))
		if !presentationClearPoint(w, x, y) {
			y = w.Player.Y
		}
		if !presentationClearPoint(w, x, y) {
			continue
		}
		score := float64(absDemo(x-w.Player.X)) + float64(absDemo(y-w.Player.Y))*.8 + float64(w.Player.Y-bounds.Bottom)*.18
		if actor.part != nil && actor.part.DamageMode == "drop-equipment" {
			score -= 25
		}
		if score < best {
			goal, best = presentationGoal{x: x, y: y, actor: actor}, score
		}
	}
	for _, bonus := range w.Collectibles {
		if !bonus.Active || bonus.Y < 16 || bonus.Y > 164 || bonus.Y < float64(w.Player.Y-65) {
			continue
		}
		x, y := presentationBonusIntercept(w, bonus)
		if !presentationReachableBonus(w, x, y) {
			continue
		}
		score := float64(absDemo(x-w.Player.X))*.9 + float64(absDemo(y-w.Player.Y))*1.1 - 18
		if score < best {
			goal, best = presentationGoal{x: x, y: y, bonus: bonus}, score
		}
	}
	if goal.actor == nil && goal.bonus == nil {
		return presentationPrepareWave(w)
	}
	return goal
}

// An experienced player can prepare for an unvisited encounter using the
// known formation data. This never creates an enemy or fires at an empty field.
func presentationPrepareWave(w *World) presentationGoal {
	goal := presentationGoal{}
	if w.Level.Encounters == nil || w.Level.Paths == nil {
		return goal
	}
	for _, wave := range w.Level.Encounters.Moving {
		if wave.TriggerY >= w.ScrollY || wave.TriggerY >= w.cursor.MovingHighWater || w.ScrollY-wave.TriggerY > 48 || wave.TriggerY <= goal.arrival {
			continue
		}
		kind, path := w.kinds[wave.EnemyKind], w.paths[wave.PathID]
		if kind == nil || len(kind.Parts) == 0 || path == nil {
			continue
		}
		part := kind.Parts[0]
		if part.MotionMode != "path" && part.MotionMode != "path-heading-frames" && part.MotionMode != "path-entry-edge-frames" || part.DamageMode == "block-shot" {
			continue
		}
		config, err := FormationMotion(wave, 0, 0, part)
		if err != nil {
			continue
		}
		if kind.MotionBudgetOverride > 0 {
			config.Budget = kind.MotionBudgetOverride
		}
		motion, err := NewPathMotion(path, config)
		if err != nil {
			continue
		}
		random := w.RandomState()
		for range 12 {
			if err := motion.Advance(path, &w.Level.Paths.SineTable, func() uint16 { return uint16(random.Next()) }); err != nil {
				motion.Active = false
				break
			}
		}
		x, y := max(24, min(296, int(motion.X>>16))), max(88, min(152, int(motion.Y>>16)+72))
		if motion.Active && presentationClearPoint(w, x, y) && presentationReachableBonus(w, x, y) {
			goal = presentationGoal{x: x, y: y, arrival: wave.TriggerY}
		}
	}
	return goal
}

func presentationBonusIntercept(w *World, bonus *WorldCollectible) (int, int) {
	motion := bonus.Motion
	motion.X, motion.Y = int(bonus.X), int(bonus.Y)
	passes := max(1, min(18, max(absDemo(motion.X-w.Player.X)/max(3, 3+w.Equipment.SpeedTier*3), absDemo(motion.Y-w.Player.Y)/max(3, 3+w.Equipment.SpeedTier))))
	for range passes {
		if !motion.Advance() {
			return -1, -1
		}
	}
	return motion.X, motion.Y
}

func presentationClearPoint(w *World, x, y int) bool {
	return x >= 20 && x <= 300 && y >= 20 && y <= 170 && (w.Coverage == nil || w.Level.PlayerStencil == nil || !w.Coverage.Touches(x, y, w.ScrollY, *w.Level.PlayerStencil))
}

func presentationReachableBonus(w *World, x, y int) bool {
	if !presentationClearPoint(w, x, y) {
		return false
	}
	passes := max(1, max(absDemo(x-w.Player.X)/max(3, 3+w.Equipment.SpeedTier*3), absDemo(y-w.Player.Y)/max(3, 3+w.Equipment.SpeedTier)))
	if passes > 32 {
		return false
	}
	for step := 1; step <= passes; step++ {
		px, py := w.Player.X+(x-w.Player.X)*step/passes, w.Player.Y+(y-w.Player.Y)*step/passes
		if w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(px, py, w.ScrollY-step*w.BaseScrollStep, *w.Level.PlayerStencil) {
			return false
		}
		for _, actor := range w.Actors {
			if actor.Active && !actor.Collision.Empty() && (actor.ActorList == "moving" || actor.ActorList == "scenery") && (CollisionRect{Left: px - 12, Top: py - 12, Right: px + 12, Bottom: py + 12}).Intersects(actor.Collision) {
				return false
			}
		}
	}
	return true
}

func (p *PresentationPilot) tacticalMotion(w *World, fallback MotionInput) MotionInput {
	best, score := fallback, math.Inf(1)
	for _, motion := range demoDirections {
		if motion.Down && w.Player.Y >= 168 && w.Rewind.Timer == 0 {
			continue
		}
		risk, distance := presentationMotionScoreWithRisk(w, motion, p.goal.x, p.goal.y, p.planner.fifthRisk, p.PALRefreshes)
		if risk >= 100000 {
			continue
		}
		value := risk + distance
		if motion != p.motion {
			value += 1.5
		}
		if value < score {
			best, score = motion, value
		}
	}
	if w.Frame < p.motionUntil {
		risk, distance := presentationMotionScoreWithRisk(w, p.motion, p.goal.x, p.goal.y, p.planner.fifthRisk, p.PALRefreshes)
		if risk == 0 && distance <= score+6 {
			return p.motion
		}
	}
	p.motion, p.motionUntil = best, w.Frame+3
	return best
}

func presentationMotionScore(w *World, motion MotionInput, x, y int) (risk, distance float64) {
	return presentationMotionScoreWithRisk(w, motion, x, y, nil, 0)
}

func presentationMotionScoreWithRisk(w *World, motion MotionInput, x, y int, native *demoFifthNativeRisk, pal int) (risk, distance float64) {
	player := w.Player
	scroll := ScrollState{Y: w.ScrollY, Minimum: w.MinimumScrollY, Maximum: w.MaximumScrollY, DeviationPasses: w.ScrollDeviationPasses}
	const horizon = 6
	damage, nativeRisk := native.losses(w, motion, horizon, pal)
	for future := 1; future <= horizon; future++ {
		if nativeRisk && damage[future-1] != 0 {
			risk += float64(damage[future-1]) * 100000 / float64(future)
		}
		actorCamera := scroll.Y
		previousPlayer := player
		player.Advance(motion, MotionContext{ScrollY: scroll.Y, VisitedScrollY: scroll.Maximum, BaseScrollStep: w.BaseScrollStep})
		if w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(player.X, player.Y, scroll.Y, *w.Level.PlayerStencil) {
			return 10000000, 0
		}
		scroll.Advance(player.ScrollStep, w.BaseScrollStep, motion.Down)
		if w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(player.X, player.Y, scroll.Y, *w.Level.PlayerStencil) {
			return 10000000, 0
		}
		for _, actor := range w.Actors {
			if !demoActorHazard(actor) || nativeRisk && actor.fixedAiming != nil {
				continue
			}
			ox, oy := int(math.Round(actor.X-actor.PreviousX))*future, int(math.Round(actor.Y-actor.PreviousY))*future
			r := actor.Collision
			r.Left, r.Right, r.Top, r.Bottom = r.Left+ox, r.Right+ox, r.Top+oy, r.Bottom+oy
			if predicted, supported := demoActorPrediction(w, actor, future, actorCamera); supported {
				if !predicted.Active {
					continue
				}
				r = predicted.Bounds
			}
			bounds := CollisionRect{Left: player.X - 14, Top: player.Y - 16, Right: player.X + 14, Bottom: player.Y + 16}
			if actor.fifthColumn != nil {
				// World.Step publishes the ship prefix before movement. Later
				// projectile callbacks still test that earlier position and bank.
				bounds = thirdMiddlePlayerBounds(w, previousPlayer)
			}
			if bounds.Intersects(r) {
				risk += 150000 / float64(future)
			}
		}
		for _, shot := range w.Projectiles {
			sx, sy, alive := demoProjectilePosition(w, shot, future, w.ScrollDelta)
			if alive && absDemo(player.X-sx) < 15 && absDemo(player.Y-sy) < 19 {
				risk += 100000 / float64(future)
			}
		}
		distance += float64(absDemo(player.X-x)+absDemo(player.Y-y)) / horizon
	}
	return risk, distance
}

// Target selection follows the same moving-list callbacks as player weapons.
// Body sections that consume hits without taking damage are poor aim targets.
func presentationTargetBounds(w *World, actor *WorldActor) (CollisionRect, bool) {
	if actor != nil && (actor.fourthIndex > 0 || actor.fifthIndex > 0) {
		return presentationGuardianTargetBounds(w, actor)
	}
	if actor != nil && actor.Active && actor.fifthTile != nil && actor.fixedTileArt != nil && actor.fixedTileArt.Kind == 1 {
		// Barrier posts are drawn into terrain, so their live damage callback
		// remains targetable without a sprite. The linking band cannot be damaged.
		part := actor.fifthTile.Part
		damageable := part >= 0 && part < len(actor.fixedTileArt.Parts) && actor.fixedTileArt.Parts[part].Damageable
		return actor.Collision, damageable && actor.Health > 0 && !actor.Collision.Empty() && actor.Collision.Left < 320 && actor.Collision.Right >= 0
	}
	if actor != nil && actor.Active && actor.thirdCannon != nil {
		return actor.Collision, actor.Health > 0 && !actor.Collision.Empty()
	}
	if actor != nil && actor.Active && actor.secondNode != nil {
		return actor.Collision, w.secondScheduler != nil && w.secondScheduler.DefenseFlags != 3 && actor.Health > 0 && !actor.Collision.Empty()
	}
	if actor == nil || !actor.Active || !actor.Visible || actor.Materializing || actor.ActorList != "moving" || actor.Collision.Empty() {
		return CollisionRect{}, false
	}
	if actor.firstGuardian {
		if w.FirstGuardian == nil {
			return CollisionRect{}, false
		}
		r := w.FirstGuardian.WeakPoint(w.ScrollY)
		return r, !r.Empty()
	}
	if actor.thirdFinalMember != nil {
		return actor.Collision, actor.thirdPart != nil && actor.thirdPart.Index == 0
	}
	if actor.thirdMiddlePart > 0 {
		return actor.Collision, actor.thirdPart != nil && (actor.thirdPart.Index == 3 || actor.thirdPart.Index == 4)
	}
	if actor.part != nil && actor.part.DamageMode == "block-shot" {
		return CollisionRect{}, false
	}
	if actor.part != nil && actor.part.Linked && actor.leader != nil {
		return CollisionRect{}, false
	}
	return actor.Collision, true
}

func (p *PresentationPilot) selectiveFire(w *World) bool {
	return p.selectiveFireForMotion(w, MotionInput{})
}

func (p *PresentationPilot) selectiveFireForMotion(w *World, motion MotionInput) bool {
	if w.Frame < p.restUntil && p.burstUntil == 0 {
		return false
	}
	if !presentationShotOpportunityWithForecast(w, motion, &p.guardianAimForecast, p.PALRefreshes) {
		p.burstUntil = 0
		return false
	}
	if w.Frame < p.restUntil {
		return false
	}
	if p.burstUntil == 0 {
		// Hold long enough for native autofire to contribute multiple shots.
		// The pause after each burst leaves time to reassess the moving target.
		period := max(1, w.Equipment.FirePeriod/max(1, w.Equipment.FireAdvance))
		p.burstUntil = w.Frame + uint64(max(8, min(20, period*2+2)))
	}
	if w.Frame >= p.burstUntil {
		p.burstUntil, p.restUntil = 0, w.Frame+4
		return false
	}
	return true
}

// Ordinary forward bullets are point hits: forecast the actual nine-pixel ray
// instead of firing at every visible sprite or at arbitrary solid terrain.
func presentationShotOpportunity(w *World) bool {
	return presentationShotOpportunityForMotion(w, MotionInput{})
}

func presentationShotOpportunityForMotion(w *World, motion MotionInput) bool {
	if !presentationGuardianAimSupported(w) {
		return presentationShotOpportunityWithForecast(w, motion, nil, 3)
	}
	var guardianForecast WorldForecast
	return presentationShotOpportunityWithForecast(w, motion, &guardianForecast, 3)
}

func presentationShotOpportunityWithForecast(w *World, motion MotionInput, guardianForecast *WorldForecast, palRefreshes int) bool {
	if opportunity, supported := presentationGuardianShotOpportunity(w, motion, guardianForecast, palRefreshes); supported {
		return opportunity
	}
	forecast := newDemoMotionForecast(w)
	// Terrain contact can start a rewind without suppressing this pass's
	// weapon phase. Its resulting gun position still determines the shot ray.
	forecast.advance(w, motion)
	ship := forecast.player
	for _, actor := range w.Actors {
		bounds, ok := presentationTargetBounds(w, actor)
		if !ok || bounds.Bottom < 0 || bounds.Top >= ship.Y-6 {
			continue
		}
		for future := 1; future <= 18; future++ {
			predicted := bounds
			if actor.thirdCannon != nil {
				camera := w.ScrollY - (future-1)*w.BaseScrollStep
				predicted = actor.thirdCannon.CollisionAt(camera)
			} else if actor.secondNode != nil {
				camera := w.ScrollY - (future-1)*w.BaseScrollStep
				predicted = CollisionRect{Left: actor.secondNode.TileX * 16, Right: actor.secondNode.TileX*16 + 15, Top: actor.secondNode.TileY*16 - camera, Bottom: actor.secondNode.TileY*16 - camera + 15}
			} else if view, supported := demoActorPrediction(w, actor, future, w.ScrollY-(future-1)*w.BaseScrollStep); supported {
				if !view.Active || !view.Visible {
					continue
				}
				predicted = view.Bounds
			} else {
				ox, oy := int(math.Round(actor.X-actor.PreviousX))*future, int(math.Round(actor.Y-actor.PreviousY))*future
				predicted.Left, predicted.Right = predicted.Left+ox, predicted.Right+ox
				predicted.Top, predicted.Bottom = predicted.Top+oy, predicted.Bottom+oy
			}
			y := ship.Y - 6 - 9*future
			if y < 0 {
				break
			}
			if y < predicted.Top || y > predicted.Bottom {
				continue
			}
			x := ship.X
			if x >= predicted.Left && x <= predicted.Right || w.Equipment.Primary.Item == ItemDoubleShot && (x-5 >= predicted.Left && x-5 <= predicted.Right || x+6 >= predicted.Left && x+6 <= predicted.Right) {
				return true
			}
		}
	}
	// Level two exposes a genuine bullet callback for these intact cells. No
	// other ordinary terrain tile awards damage or justifies holding fire.
	if w.Level.Number == 2 && w.secondTerrainCells != nil {
		for future := 1; future <= 18; future++ {
			y := ship.Y - 6 - 9*future
			if y < 0 {
				break
			}
			if w.secondTerrainCells.FindBullet(CollisionRect{Left: ship.X, Right: ship.X, Top: y, Bottom: y}, w.ScrollY-(future-1)*w.BaseScrollStep) >= 0 {
				return true
			}
		}
	}
	return false
}
