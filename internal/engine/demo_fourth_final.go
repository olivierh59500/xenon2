package engine

import (
	"math"
	"xenon2/internal/visualassets"
)

// FourthFinalInput supplies ordinary controls for the original eye/core sequence.
func (p *DemoPilot) FourthFinalInput(w *World) (Input, bool) {
	if w == nil || w.Level.Number != 4 || w.Ready || w.GameOver || !w.PlayerAlive || p.Config.DisableBossAlignment {
		return Input{}, false
	}
	fire := (w.Frame+1)%2 != 0 && !w.blockedFireUntilRelease
	if w.FourthFinal == nil {
		if w.ScrollY <= 176 {
			return Input{Fire: fire}, true
		}
		return Input{}, false
	}
	if w.FourthFinal.Defeated || w.fourthFinalArt == nil || w.Level.Paths == nil {
		return Input{}, false
	}
	index := 0
	if !w.FourthFinal.Parts[1].Disabled {
		index = 1
	} else if !w.FourthFinal.Parts[2].Disabled {
		index = 2
	}
	x := 160
	if index != 0 {
		x = int(w.FourthFinal.Parts[index].Arc.X>>16) + 16
	}
	targetY := 148
	if index == 0 {
		targetY = max(80, min(156, int(w.FourthFinal.Parts[0].Arc.Y>>16)+112))
	}
	const horizon = 8
	best, rank := MotionInput{}, math.Inf(1)
	for _, candidate := range demoDirections {
		if candidate.Down && w.Player.Y >= 168 {
			continue
		}
		player, state, random := w.Player, *w.FourthFinal, w.RandomState()
		scroll := ScrollState{Y: w.ScrollY, Minimum: w.MinimumScrollY, Maximum: w.MaximumScrollY, DeviationPasses: w.ScrollDeviationPasses}
		delta := w.ScrollDelta
		score := 0.0
		for future := 0; future < horizon; future++ {
			player.Advance(candidate, MotionContext{ScrollY: scroll.Y, VisitedScrollY: scroll.Maximum, BaseScrollStep: w.BaseScrollStep})
			if w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(player.X, player.Y, scroll.Y, *w.Level.PlayerStencil) {
				score += 10000000
				break
			}
			rect := thirdMiddlePlayerBounds(w, player)
			for part := 0; part < 19; part++ {
				event, err := state.AdvancePart(part, w.fourthFinalArt, FourthGuardianInput{Frame: w.Frame + uint64(future) + 1, ScrollY: scroll.Y, ScrollDelta: delta, MaximumScrollY: scroll.Maximum, PlayerX: player.X, PlayerY: player.Y}, &w.Level.Paths.SineTable, func(name string) visualassets.CollisionBox { return w.movingSpriteBoxes[name] }, random.Next)
				if err != nil {
					return Input{}, false
				}
				scroll.Maximum = event.MaximumScrollY
				if !state.Parts[part].Collision.Empty() && rect.Intersects(state.Parts[part].Collision) {
					score += 100000 / float64(future+1)
				}
			}
			for _, shot := range w.Projectiles {
				if !shot.Active {
					continue
				}
				sx, sy, alive := demoProjectilePosition(w, shot, future+1, delta)
				if alive && sx >= rect.Left-5 && sx <= rect.Right+5 && sy >= rect.Top-5 && sy <= rect.Bottom+5 {
					score += 100000 / float64(future+1)
				}
			}
			score += float64(absDemo(player.X-x)+absDemo(player.Y-targetY)*2) / horizon
			scroll.Advance(player.ScrollStep, w.BaseScrollStep, candidate.Down)
			delta = scroll.ActualStep
			score += float64(scroll.Y-w.ScrollY) * 2
		}
		if score < rank {
			best, rank = candidate, score
		}
	}
	return Input{Motion: best, Fire: fire}, true
}
