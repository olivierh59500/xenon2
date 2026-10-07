package engine

import "math"

// SecondCorridorInput chooses the left exit before the second level's right
// passage closes. The original full-ship stencil has a horizontal crossing
// around world Y1168; targeting a point beyond the closure makes the bounded
// route traverse that crossing rather than descend into the right pocket.
func (p *DemoPilot) SecondCorridorInput(w *World) (Input, bool) {
	if w == nil || w.Level.Number != 2 || w.Ready || w.GameOver || !w.PlayerAlive || !w.secondMiddleReleased || w.Checkpoint.ScrollY > 1792 || w.ScrollY > 1280 || w.ScrollY < 800 || p.Config.DisableBossAlignment {
		return Input{}, false
	}
	worldY := w.Player.Y + w.ScrollY
	// An already trapped ship cannot reverse past its visited camera bound.
	// This policy starts early, rather than pretending the closed pocket has
	// a physically reachable escape after the crossing has scrolled away.
	if worldY < 1120 && w.Player.X > 200 && w.MaximumScrollY+176 < 1168 {
		return Input{}, false
	}
	goal := 1072
	if worldY <= 1072 {
		goal = 928
	}
	x, y, found := p.secondFinalWaypoint(w, goal)
	if !found {
		return Input{}, false
	}
	return Input{Motion: secondCorridorRouteMotion(w, x, y), Fire: (w.Frame+1)%2 != 0 && !w.blockedFireUntilRelease}, true
}

// Screen-space comfort keeps the ship below the arriving formations while the
// world-coordinate route crosses the thin horizontal passage.
func secondCorridorRouteMotion(w *World, x, y int) MotionInput {
	type branch struct {
		player PlayerMotionState
		scroll ScrollState
		first  int
		score  float64
	}
	current := []branch{{player: w.Player, scroll: ScrollState{Y: w.ScrollY, Minimum: w.MinimumScrollY, Maximum: w.MaximumScrollY, DeviationPasses: w.ScrollDeviationPasses}, first: -1}}
	for depth := 0; depth < 8; depth++ {
		next := make([]branch, 0, len(current)*9)
		for _, before := range current {
			for action, motion := range demoDirections {
				b := before
				b.player.Advance(motion, MotionContext{ScrollY: b.scroll.Y, VisitedScrollY: b.scroll.Maximum, BaseScrollStep: w.BaseScrollStep})
				if w.Coverage.Touches(b.player.X, b.player.Y, b.scroll.Y, *w.Level.PlayerStencil) {
					continue
				}
				b.scroll.Advance(b.player.ScrollStep, w.BaseScrollStep, motion.Down)
				if w.Coverage.Touches(b.player.X, b.player.Y, b.scroll.Y, *w.Level.PlayerStencil) {
					continue
				}
				if b.first < 0 {
					b.first = action
				}
				b.score += float64(absDemo(b.player.X-x) + absDemo(b.player.Y+b.scroll.Y-y)*2 + absDemo(b.player.Y-136)*2)
				rect := w.playerCollision
				if !rect.Empty() {
					dx, dy := b.player.X-w.Player.X, b.player.Y-w.Player.Y
					rect.Left += dx
					rect.Right += dx
					rect.Top += dy
					rect.Bottom += dy
					for _, a := range w.Actors {
						if !a.Active || a.Collision.Empty() || a.ActorList != "moving" && a.ActorList != "scenery" {
							continue
						}
						other := a.Collision
						ox, oy := int(math.Round(a.X-a.PreviousX))*(depth+1), int(math.Round(a.Y-a.PreviousY))*(depth+1)
						other.Left += ox
						other.Right += ox
						other.Top += oy
						other.Bottom += oy
						if rect.Intersects(other) || (CollisionRect{Left: rect.Left - 10, Top: rect.Top - 10, Right: rect.Right + 10, Bottom: rect.Bottom + 10}).Intersects(other) {
							b.score += 100000 / float64(depth+1)
						}
					}
					for _, shot := range w.Projectiles {
						if !shot.Active {
							continue
						}
						pred := shot.Motion
						for step := 0; step <= depth; step++ {
							pred.Advance(w.ScrollDelta)
						}
						if int(pred.X>>16) >= rect.Left-4 && int(pred.X>>16) <= rect.Right+4 && int(pred.Y>>16) >= rect.Top-4 && int(pred.Y>>16) <= rect.Bottom+4 {
							b.score += 100000 / float64(depth+1)
						}
					}
				}
				next = append(next, b)
			}
		}
		if len(next) == 0 {
			if len(current) > 0 && current[0].first >= 0 {
				return demoDirections[current[0].first]
			}
			return MotionInput{}
		}
		for i := 0; i < min(12, len(next)); i++ {
			best := i
			for j := i + 1; j < len(next); j++ {
				if next[j].score < next[best].score {
					best = j
				}
			}
			next[i], next[best] = next[best], next[i]
		}
		current = next[:min(12, len(next))]
	}
	return demoDirections[current[0].first]
}
