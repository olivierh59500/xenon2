package engine

import "math"

// DemoPilotConfig adjusts the deterministic controller without changing game
// rules, collision data, health, inventory or the shared random stream.
type DemoPilotConfig struct {
	TargetX, TargetY                                                       int
	Lookahead, SafetyMargin, FireReleasePeriod                             int
	DisableOpeningRoute, DisableBonuses, DisableBossAlignment, DisableDive bool
}

// DemoPilot produces ordinary player commands. Its zero value uses the verified
// second-level opening policy and conservative short-horizon obstacle avoidance.
// It is a development controller, not a guarantee of completing every stage.
type DemoPilot struct {
	Config             DemoPilotConfig
	navigation         *demoNavigation
	retreatGoal        int
	practicedRoute     bool
	retreatX, retreatY int
	secondArenaScratch []demoSecondDefenseView
}

var demoDirections = [9]MotionInput{{}, {Left: true}, {Right: true}, {Up: true}, {Down: true}, {Left: true, Up: true}, {Right: true, Up: true}, {Left: true, Down: true}, {Right: true, Down: true}}

func (p *DemoPilot) NormalInput(w *World) Input {
	if w == nil || w.GameOver {
		return Input{}
	}
	if w.Ready {
		p.retreatGoal = 0
		return Input{Fire: true}
	}
	if p.navigation != nil && p.navigation.world != w {
		p.retreatGoal = 0
	}
	if input, handled := p.FourthFinalInput(w); handled {
		return input
	}
	if input, handled := p.ThirdMiddleInput(w); handled {
		return input
	}
	if input, handled := p.ThirdGuardianInput(w); handled {
		return input
	}
	if input, handled := p.SecondCorridorInput(w); handled {
		return input
	}
	if input, handled := p.SecondFinalInput(w); handled {
		return input
	}
	if input, handled := p.StageInput(w); handled {
		return input
	}
	c := firstStageConfig(w, thirdOpeningConfig(w, p.Config))
	if w.Level.Number == 2 && w.Checkpoint.ScrollY <= 4032 {
		if c.TargetY == 0 {
			c.TargetY = 166
		}
		if c.FireReleasePeriod == 0 {
			c.FireReleasePeriod = 2
		}
	}
	if c.Lookahead <= 0 {
		c.Lookahead = 5
	}
	c.Lookahead = min(8, c.Lookahead)
	if c.SafetyMargin <= 0 {
		c.SafetyMargin = 18
	}
	if c.FireReleasePeriod <= 0 {
		c.FireReleasePeriod = 4
	}
	x, y := c.TargetX, c.TargetY
	if x == 0 {
		x = 160
	}
	if y == 0 {
		y = 120
	}
	opening := !c.DisableOpeningRoute && w.Level.Number == 2 && w.ScrollY >= 4032 && w.Checkpoint.ScrollY >= 4032
	if opening {
		x, y = 250, 40
	}
	if !c.DisableBossAlignment && w.Level.Number == 1 && w.FirstGuardian != nil && w.FirstGuardian.Active && !w.FirstGuardian.Defeated {
		weak := w.FirstGuardian.WeakPoint(w.ScrollY)
		x, y = (weak.Left+weak.Right)/2+9, 166
	}
	routeX, routeY, route := 0, 0, false
	retreat := false
	firstApproach := p.practicedRoute && w.Level.Number == 1 && (w.FirstGuardian == nil || !w.FirstGuardian.Active)
	if !opening && w.Coverage != nil && w.Level.PlayerStencil != nil && (w.ScrollY > 640 || firstApproach) {
		goal := w.ScrollY + w.Player.Y - 128
		if firstApproach && w.FirstMiddle != nil && w.FirstMiddle.Crossed && w.ScrollY <= 1024 && p.retreatGoal == 0 {
			goal = 384
		}
		if p.retreatGoal != 0 {
			if w.ScrollY+w.Player.Y <= p.retreatGoal+6 || w.ScrollY+w.Player.Y <= p.retreatY+6 && absDemo(w.Player.X-p.retreatX) >= 48 {
				p.retreatGoal = 0
			} else {
				goal = p.retreatGoal
			}
		}
		if w.Level.Number == 1 && w.FirstMiddle != nil && !w.FirstMiddle.Crossed && w.ScrollY < 3456 {
			// A practiced player knows the arena exit before choosing a branch.
			// A nearby row alone can accept a pocket that closes farther ahead.
			if p.retreatGoal == 0 && (p.practicedRoute || w.ScrollY < 2750) {
				goal = 2685
			}
			y = max(25, min(80, 2685-w.ScrollY))
			if p.Config == (DemoPilotConfig{}) && (!p.practicedRoute || w.ScrollY >= 2750) {
				y = c.TargetY
			}
		}
		if p.navigation == nil {
			p.navigation = &demoNavigation{}
		}
		p.navigation.practiced = p.practicedRoute
		p.navigation.targetX = 0
		if p.practicedRoute && w.Level.Number == 1 && w.FirstMiddle != nil && !w.FirstMiddle.Crossed && w.ScrollY < 3456 && w.ScrollY > 3160 && w.Player.X > 140 && p.retreatGoal == 0 {
			// Prepare the known left junction before the right pocket closes.
			// Its position comes from the original full-stencil terrain route.
			goal, p.navigation.targetX = 3422, 80
		}
		routeX, routeY, route = p.navigation.waypoint(w, goal)
		if p.practicedRoute && !route && p.retreatGoal == 0 && w.Level.Number == 1 && w.FirstMiddle != nil && !w.FirstMiddle.Crossed {
			// A ship already in the wrong pocket may need a local rear escape
			// before the bounded search can reach the full arena destination.
			goal = w.ScrollY + w.Player.Y - 128
			routeX, routeY, route = p.navigation.waypoint(w, goal)
		}
		retreat = route && p.navigation.retreat
		if retreat && p.retreatGoal == 0 {
			rear := w.ScrollY + w.Player.Y
			for _, point := range p.navigation.path {
				rear = max(rear, point.y)
			}
			// Keep the forward destination while returning to the junction;
			// moving the goal backward can accept the same closed pocket again.
			if rear > w.ScrollY+w.Player.Y+32 {
				p.retreatGoal, p.retreatX, p.retreatY = goal, w.Player.X, w.ScrollY+w.Player.Y
			}
		}
		retreat = route && p.retreatGoal != 0
		if retreat {
			y = 176
		}
	}
	if !opening && !retreat && !c.DisableBonuses {
		best := math.Inf(1)
		for _, item := range w.Collectibles {
			if !item.Active || item.Y < 0 || item.Y > 168 || item.Y < float64(w.Player.Y-50) {
				continue
			}
			if w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(int(item.X), int(item.Y), w.ScrollY, *w.Level.PlayerStencil) {
				continue
			}
			distance := math.Abs(item.X-float64(w.Player.X)) + math.Abs(item.Y-float64(w.Player.Y))*2
			if distance < best {
				best = distance
				x, y = int(item.X), int(item.Y)
				route = false
			}
		}
	}
	maximumY := 170
	if p.practicedRoute {
		maximumY = 176
	}
	x, y = max(20, min(300, x)), max(25, min(maximumY, y))
	best, bestScore, bestThreat := 0, math.Inf(1), false
	for action, motion := range demoDirections {
		// Holding down at the bottom requests reverse scrolling. Short-horizon
		// risk scoring must not turn that escape into a stationary campaign.
		if motion.Down && w.Player.Y >= 168 && w.Rewind.Timer == 0 && !retreat {
			continue
		}
		player := w.Player
		scroll := ScrollState{Y: w.ScrollY, Minimum: w.MinimumScrollY, Maximum: w.MaximumScrollY, DeviationPasses: w.ScrollDeviationPasses}
		score := 0.0
		immediateThreat := false
		forecast := newDemoMotionForecast(w)
		for future := 1; future <= c.Lookahead; future++ {
			if retreat || p.practicedRoute && w.Rewind.Timer != 0 {
				if !forecast.advance(w, motion) {
					score += 10000000
					break
				}
				player, scroll = forecast.player, forecast.scroll
			} else {
				player.Advance(motion, MotionContext{ScrollY: scroll.Y, VisitedScrollY: scroll.Maximum, BaseScrollStep: w.BaseScrollStep})
				if w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(player.X, player.Y, scroll.Y, *w.Level.PlayerStencil) {
					score += 10000000
					break
				}
				scroll.Advance(player.ScrollStep, w.BaseScrollStep, motion.Down)
				// Scrolling happens after the ship update, so a clear movement endpoint
				// can still touch terrain at the next pass's camera position.
				if w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(player.X, player.Y, scroll.Y, *w.Level.PlayerStencil) {
					score += 10000000
					break
				}
			}
			dx, dy := float64(player.X-x), float64(player.Y-y)
			if route {
				rx, ry := float64(player.X-routeX), float64(player.Y+scroll.Y-routeY)
				weight := .002
				if retreat {
					weight = .06
				}
				score += (rx*rx*.006 + ry*ry*weight + dy*dy*.008) / float64(c.Lookahead)
			} else {
				score += (dx*dx*.003 + dy*dy*.005) / float64(c.Lookahead)
			}
			for _, actor := range w.Actors {
				if !actor.Active || actor.Collision.Empty() || actor.ActorList != "moving" && actor.ActorList != "scenery" {
					continue
				}
				ox, oy := int(math.Round(actor.X-actor.PreviousX))*future, int(math.Round(actor.Y-actor.PreviousY))*future
				r := actor.Collision
				if player.X+c.SafetyMargin >= r.Left+ox && player.X-c.SafetyMargin <= r.Right+ox && player.Y+c.SafetyMargin >= r.Top+oy && player.Y-c.SafetyMargin <= r.Bottom+oy {
					score += 150000 / float64(future)
					immediateThreat = immediateThreat || future <= 2
				}
			}
			for _, shot := range w.Projectiles {
				if !shot.Active {
					continue
				}
				sx, sy, alive := demoProjectilePosition(w, shot, future, w.ScrollDelta)
				if alive && absDemo(player.X-sx) < c.SafetyMargin && absDemo(player.Y-sy) < c.SafetyMargin+4 {
					score += 100000 / float64(future)
					immediateThreat = immediateThreat || future <= 2
				}
			}
		}
		score += float64(scroll.Y-w.ScrollY) * .1
		if score < bestScore {
			bestScore, best, bestThreat = score, action, immediateThreat
		}
	}
	input := Input{Motion: demoDirections[best], Fire: (w.Frame+1)%uint64(c.FireReleasePeriod) != 0}
	if w.blockedFireUntilRelease {
		input.Fire = false
	}
	if retreat {
		input.Motion = demoRouteMotion(w, routeX, routeY)
	}
	if !c.DisableDive && bestThreat && w.Dive.Phase == 0 && w.Equipment.DiveCharges > 0 && w.MaterializationFrames == 0 && w.InvulnerableFrames == 0 {
		// Avoid requesting a dive whose unchanged ship location would resurface
		// inside known terrain at the ordinary 136-pass travel distance.
		safe := w.Coverage == nil || w.Level.PlayerStencil == nil || !w.Coverage.Touches(w.Player.X, w.Player.Y, max(w.MinimumScrollY, w.ScrollY-136), *w.Level.PlayerStencil)
		input.Dive = safe
	}
	return input
}

func absDemo(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
