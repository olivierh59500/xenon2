package engine

import (
	"math"
	"xenon2/internal/visualassets"
)

// ThirdGuardianInput aims ordinary forward shots at the final worm head.
// Neck, body and tail segments absorb shots, so they are never selected as
// targets. Prediction copies the original path and random state without
// changing either; this helper does not control the unverified middle arena.
func (p *DemoPilot) ThirdGuardianInput(w *World) (Input, bool) {
	if w == nil || w.Level.Number != 3 || w.Ready || w.GameOver || !w.PlayerAlive || p.Config.DisableBossAlignment {
		return Input{}, false
	}
	if w.ThirdFinal == nil || w.ThirdFinal.Defeated || w.ScrollY > 208 {
		return Input{}, false
	}
	var target *WorldActor
	var motion PathMotionState
	var path *visualassets.Path
	for _, actor := range w.Actors {
		if actor.Active && actor.thirdFinalMember != nil && actor.thirdPart.Index == 0 {
			target = actor
			motion = actor.thirdFinalMember.Motion
			path = actor.path
			break
		}
	}
	if target == nil {
		return Input{Fire: (w.Frame+1)%2 != 0 && !w.blockedFireUntilRelease}, true
	}
	flight := max(1, min(12, (w.Player.Y-int(target.Y)-6)/9))
	random := w.RandomState()
	for range flight {
		if err := motion.Advance(path, &w.Level.Paths.SineTable, func() uint16 { return uint16(random.Next()) }); err != nil {
			break
		}
	}
	x := max(20, min(300, int(motion.X>>16)))
	y := 176

	input := Input{Motion: demoAimMotion(w.Player, x, y), Fire: (w.Frame+1)%2 != 0 && !w.blockedFireUntilRelease}
	// These targets can touch the ship; retain a short collision-aware choice
	// whenever the current lead position lies inside a dangerous body segment.
	best, rank := input.Motion, math.Inf(1)
	for _, candidate := range demoDirections {
		player := w.Player
		player.Advance(candidate, MotionContext{ScrollY: w.ScrollY, VisitedScrollY: w.MaximumScrollY, BaseScrollStep: w.BaseScrollStep})
		if w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(player.X, player.Y, w.ScrollY, *w.Level.PlayerStencil) {
			continue
		}
		score := float64(absDemo(player.X-x) + absDemo(player.Y-y)*2)
		rect := w.playerCollision
		dx, dy := player.X-w.Player.X, player.Y-w.Player.Y
		rect.Left += dx
		rect.Right += dx
		rect.Top += dy
		rect.Bottom += dy
		for _, actor := range w.Actors {
			if actor.Active && !actor.Collision.Empty() && rect.Intersects(actor.Collision) {
				score += 100000
			}
		}
		if score < rank {
			best, rank = candidate, score
		}
	}
	input.Motion = best
	return input, true
}
