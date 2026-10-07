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
type DemoPilot struct{ Config DemoPilotConfig }

var demoDirections = [9]MotionInput{{}, {Left: true}, {Right: true}, {Up: true}, {Down: true}, {Left: true, Up: true}, {Right: true, Up: true}, {Left: true, Down: true}, {Right: true, Down: true}}

func (p *DemoPilot) NormalInput(w *World) Input {
	if w == nil || w.GameOver {
		return Input{}
	}
	if w.Ready {
		return Input{Fire: true}
	}
	c := p.Config
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
	if !opening && !c.DisableBonuses {
		best := math.Inf(1)
		for _, item := range w.Collectibles {
			if !item.Active || item.Y < 0 || item.Y > 168 || item.Y < float64(w.Player.Y-50) {
				continue
			}
			distance := math.Abs(item.X-float64(w.Player.X)) + math.Abs(item.Y-float64(w.Player.Y))*2
			if distance < best {
				best = distance
				x, y = int(item.X), int(item.Y)
			}
		}
	}
	x, y = max(20, min(300, x)), max(25, min(170, y))
	best, bestScore, bestThreat := 0, math.Inf(1), false
	for action, motion := range demoDirections {
		player := w.Player
		scroll := ScrollState{Y: w.ScrollY, Minimum: w.MinimumScrollY, Maximum: w.MaximumScrollY, DeviationPasses: w.ScrollDeviationPasses}
		score := 0.0
		immediateThreat := false
		for future := 1; future <= c.Lookahead; future++ {
			player.Advance(motion, MotionContext{ScrollY: scroll.Y, VisitedScrollY: scroll.Maximum, BaseScrollStep: w.BaseScrollStep})
			if w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(player.X, player.Y, scroll.Y, *w.Level.PlayerStencil) {
				score += 10000000
				break
			}
			scroll.Advance(player.ScrollStep, w.BaseScrollStep, motion.Down)
			dx, dy := float64(player.X-x), float64(player.Y-y)
			score += (dx*dx*.003 + dy*dy*.005) / float64(c.Lookahead)
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
				prediction := shot.Motion
				for step := 0; step < future; step++ {
					_, _ = prediction.Advance(w.ScrollDelta)
				}
				if absDemo(player.X-int(prediction.X>>16)) < c.SafetyMargin && absDemo(player.Y-int(prediction.Y>>16)) < c.SafetyMargin+4 {
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
