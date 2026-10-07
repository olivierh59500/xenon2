package engine

import "math"

// PresentationPilot plays through ordinary controls with a short reaction time,
// committed tactical goals and held firing bursts. The game still owns every
// movement, hit, reward, terrain change and random value.
type PresentationPilot struct {
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
	x, y  int
	actor *WorldActor
	bonus *WorldCollectible
}

func (p *PresentationPilot) NormalInput(w *World) Input {
	if w == nil || w.GameOver || !w.PlayerAlive {
		return Input{}
	}
	if w.Ready {
		return Input{Fire: true}
	}
	if p.world != w || w.Frame < p.frame {
		*p = PresentationPilot{world: w, frame: w.Frame, decisionAt: w.Frame + 3}
	}
	p.frame = w.Frame
	base := p.planner.NormalInput(w)
	input := base
	if !presentationSpecialist(w) {
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
	input.Fire = p.selectiveFire(w)
	if w.blockedFireUntilRelease || w.Dive.Phase != 0 {
		input.Fire = false
	}
	return input
}

// Source-specific arena movement retains the already verified controller.
// Trigger decisions still require a live shot opportunity in those arenas.
func presentationSpecialist(w *World) bool {
	return w.Level.Number == 2 && (w.secondScheduler != nil && w.ScrollY >= 2512 && w.ScrollY <= 2896 || w.secondMiddleReleased && w.ScrollY <= 1280) ||
		w.Level.Number == 3 && (w.ThirdMiddle != nil && !w.ThirdMiddle.Defeated || w.ThirdFinal != nil && !w.ThirdFinal.Defeated && w.ScrollY <= 208) ||
		w.Level.Number == 4 && w.ScrollY <= 176 ||
		w.Level.Number == 1 && w.FirstGuardian != nil && w.FirstGuardian.Active && !w.FirstGuardian.Defeated
}

func (g presentationGoal) valid(w *World) bool {
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
		x := (bounds.Left+bounds.Right)/2 + int(math.Round(actor.X-actor.PreviousX))*flight
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
		x, y := int(bonus.X), int(bonus.Y)
		if !presentationReachableBonus(w, x, y) {
			continue
		}
		score := float64(absDemo(x-w.Player.X))*.9 + float64(absDemo(y-w.Player.Y))*1.1 - 18
		if score < best {
			goal, best = presentationGoal{x: x, y: y, bonus: bonus}, score
		}
	}
	return goal
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
		risk, distance := presentationMotionScore(w, motion, p.goal.x, p.goal.y)
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
		risk, distance := presentationMotionScore(w, p.motion, p.goal.x, p.goal.y)
		if risk == 0 && distance <= score+6 {
			return p.motion
		}
	}
	p.motion, p.motionUntil = best, w.Frame+3
	return best
}

func presentationMotionScore(w *World, motion MotionInput, x, y int) (risk, distance float64) {
	player := w.Player
	scroll := ScrollState{Y: w.ScrollY, Minimum: w.MinimumScrollY, Maximum: w.MaximumScrollY, DeviationPasses: w.ScrollDeviationPasses}
	const horizon = 6
	for future := 1; future <= horizon; future++ {
		player.Advance(motion, MotionContext{ScrollY: scroll.Y, VisitedScrollY: scroll.Maximum, BaseScrollStep: w.BaseScrollStep})
		if w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(player.X, player.Y, scroll.Y, *w.Level.PlayerStencil) {
			return 10000000, 0
		}
		scroll.Advance(player.ScrollStep, w.BaseScrollStep, motion.Down)
		if w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(player.X, player.Y, scroll.Y, *w.Level.PlayerStencil) {
			return 10000000, 0
		}
		for _, actor := range w.Actors {
			if !actor.Active || actor.Collision.Empty() || actor.ActorList != "moving" && actor.ActorList != "scenery" {
				continue
			}
			ox, oy := int(math.Round(actor.X-actor.PreviousX))*future, int(math.Round(actor.Y-actor.PreviousY))*future
			r := actor.Collision
			if (CollisionRect{Left: player.X - 14, Top: player.Y - 16, Right: player.X + 14, Bottom: player.Y + 16}).Intersects(CollisionRect{Left: r.Left + ox, Top: r.Top + oy, Right: r.Right + ox, Bottom: r.Bottom + oy}) {
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
	if !presentationShotOpportunity(w) {
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
	for _, actor := range w.Actors {
		bounds, ok := presentationTargetBounds(w, actor)
		if !ok || bounds.Bottom < 0 || bounds.Top >= w.Player.Y-6 {
			continue
		}
		for future := 1; future <= 18; future++ {
			ox, oy := int(math.Round(actor.X-actor.PreviousX))*future, int(math.Round(actor.Y-actor.PreviousY))*future
			y := w.Player.Y - 6 - 9*future
			if y < 0 {
				break
			}
			if y < bounds.Top+oy || y > bounds.Bottom+oy {
				continue
			}
			x := w.Player.X
			if x >= bounds.Left+ox && x <= bounds.Right+ox || w.Equipment.Primary.Item == ItemDoubleShot && (x-5 >= bounds.Left+ox && x-5 <= bounds.Right+ox || x+6 >= bounds.Left+ox && x+6 <= bounds.Right+ox) {
				return true
			}
		}
	}
	// Level two exposes a genuine bullet callback for these intact cells. No
	// other ordinary terrain tile awards damage or justifies holding fire.
	if w.Level.Number == 2 && w.secondTerrainCells != nil {
		for future := 1; future <= 18; future++ {
			y := w.Player.Y - 6 - 9*future
			if y < 0 {
				break
			}
			if w.secondTerrainCells.FindBullet(CollisionRect{Left: w.Player.X, Right: w.Player.X, Top: y, Bottom: y}, w.ScrollY-future*w.BaseScrollStep) >= 0 {
				return true
			}
		}
	}
	return false
}
