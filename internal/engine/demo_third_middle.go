package engine

import "math"

// ThirdMiddleInput follows an intact eye during its firing windows and moves
// toward the nearest screen edge before the articulated body descends too low.
// Six copied source-controller passes predict the actual arm geometry; every
// hit, item, life and camera update remains owned by the normal world engine.
func (p *DemoPilot) ThirdMiddleInput(w *World) (Input, bool) {
	if w == nil || w.Level.Number != 3 || w.Ready || w.GameOver || !w.PlayerAlive || p.Config.DisableBossAlignment || w.ThirdMiddle == nil || w.ThirdMiddle.Defeated || w.thirdMiddleArt == nil || w.Coverage == nil || w.Level.PlayerStencil == nil {
		return Input{}, false
	}
	const horizon = 6
	var terrain *demoNavigation
	if w.Coverage.Columns == 20 && w.Coverage.Rows == 300 && len(w.Coverage.Map) == 6000 {
		if p.navigation == nil {
			p.navigation = &demoNavigation{}
		}
		if !p.middleTerrainFrozen || p.navigation.world != w {
			p.navigation.refresh(w)
		}
		terrain = p.navigation
	}
	eye := 3
	if int16(w.ThirdMiddle.EyeHealth[0]) <= 0 {
		eye = 4
	}
	target := w.ThirdMiddle.Parts[eye]
	flight := max(1, min(12, (w.Player.Y-target.Y-6)/9))
	motion, random := w.ThirdMiddle.Flight, w.RandomState()
	for range flight {
		if err := motion.Advance(w.thirdMiddleArt.Path, &w.Level.Paths.SineTable, func() uint16 { return uint16(random.Next()) }); err != nil {
			break
		}
	}
	x, y := max(20, min(300, int(motion.X>>16)+w.thirdMiddleArt.Components[eye].OffsetX)), 176
	futureY := int(motion.Y>>16) + w.thirdMiddleArt.Components[eye].OffsetY
	if target.Y > 136 || futureY > 136 {
		if w.Player.X < 160 {
			x = 32
		} else {
			x = 288
		}
	}
	var futureParts [horizon][17]CollisionRect
	state, nextRandom := *w.ThirdMiddle, w.RandomState()
	for future := 0; future < horizon; future++ {
		if future > 0 {
			if _, err := state.Advance(w.thirdMiddleArt, &w.Level.Paths.SineTable, w.Frame+uint64(future), w.Player.X, w.Player.Y, &nextRandom); err != nil {
				return Input{}, false
			}
		}
		for index, part := range state.Parts {
			futureParts[future][index] = CollisionRect{Right: -1, Bottom: -1}
			if part.Collidable {
				if box, ok := w.movingSpriteBoxes[part.Sprite]; ok {
					futureParts[future][index] = ActorCollisionRect(box, part.X, part.Y)
				}
			}
		}
	}
	best, rank := MotionInput{}, math.Inf(1)
	for _, candidate := range demoDirections {
		player, score := w.Player, 0.0
		for future := 0; future < horizon; future++ {
			player.Advance(candidate, MotionContext{ScrollY: w.ScrollY, VisitedScrollY: w.MaximumScrollY, BaseScrollStep: w.BaseScrollStep})
			touching := false
			if terrain != nil {
				touching = terrain.touching(player.X, player.Y+w.ScrollY)
			} else {
				touching = w.Coverage.Touches(player.X, player.Y, w.ScrollY, *w.Level.PlayerStencil)
			}
			if touching {
				score += 10000000
				break
			}
			rect := thirdMiddlePlayerBounds(w, player)
			score += float64(absDemo(player.X-x)+absDemo(player.Y-y)*2) / horizon
			for _, other := range futureParts[future] {
				if !other.Empty() && rect.Intersects(other) {
					score += 100000 / float64(future+1)
				}
			}
			for _, shot := range w.Projectiles {
				if !shot.Active {
					continue
				}
				sx, sy, alive := demoProjectilePosition(w, shot, future+1, w.ScrollDelta)
				if alive && sx >= rect.Left-5 && sx <= rect.Right+5 && sy >= rect.Top-5 && sy <= rect.Bottom+5 {
					score += 100000 / float64(future+1)
				}
			}
		}
		if score < rank {
			best, rank = candidate, score
		}
	}
	return Input{Motion: best, Fire: (w.Frame+1)%2 != 0}, true
}

func thirdMiddlePlayerBounds(w *World, player PlayerMotionState) CollisionRect {
	names := [5]string{"player-ship-0", "player-ship-1", "player-ship-2", "player-ship-3", "player-ship-4"}
	if w.Level.Ships != nil {
		for _, sprite := range w.Level.Ships.Atlas.Sprites {
			if sprite.Name == names[player.BankFrame()] && sprite.Collision != nil {
				return ActorCollisionRect(*sprite.Collision, player.X, player.Y)
			}
		}
	}
	return CollisionRect{Right: -1, Bottom: -1}
}
