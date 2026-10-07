package engine

import "math"

func fifthBarrierTarget(w *World) *WorldActor {
	if w == nil || w.Level.Number != 5 {
		return nil
	}
	var target *WorldActor
	distance := math.MaxInt
	for _, actor := range w.Actors {
		if actor.fifthTile == nil || actor.fixedTileArt == nil || actor.fixedTileArt.Kind != 1 {
			continue
		}
		bounds, damageable := presentationTargetBounds(w, actor)
		if !damageable || bounds.Bottom < 0 || bounds.Top >= w.Player.Y-6 {
			continue
		}
		x := (bounds.Left + bounds.Right) / 2
		if d := absDemo(w.Player.X - x); d < distance {
			target, distance = actor, d
		}
	}
	return target
}

// FifthBarrierInput aligns with a real post while its firing lane is still
// below the barrier. Native motion copies account for release inertia; holding
// down at the bottom requests only the source's available reverse allowance.
func (p *DemoPilot) FifthBarrierInput(w *World) (Input, bool) {
	if w == nil || w.Ready || w.GameOver || !w.PlayerAlive || p.Config.DisableBossAlignment || w.Dive.Phase != 0 {
		return Input{}, false
	}
	target := fifthBarrierTarget(w)
	if target == nil {
		return Input{}, false
	}
	x := (target.Collision.Left + target.Collision.Right) / 2
	best, rank, found := MotionInput{}, math.MaxInt, false
	for _, motion := range demoDirections {
		forecast := newDemoMotionForecast(w)
		safe := true
		for range 3 {
			if !forecast.advance(w, motion) {
				safe = false
				break
			}
		}
		if !safe {
			continue
		}
		score := absDemo(forecast.player.X-x) + absDemo(forecast.player.Y-176)*2
		if !found || score < rank || score == rank && motion.Down && !best.Down {
			best, rank, found = motion, score, true
		}
	}
	if !found {
		return Input{}, false
	}
	return Input{Motion: best, Fire: (w.Frame+1)%2 != 0 && !w.blockedFireUntilRelease && presentationShotOpportunityForMotion(w, best)}, true
}
