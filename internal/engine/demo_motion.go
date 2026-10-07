package engine

// demoMotionForecast copies player, camera and terrain-rewind state. Predictions
// use the same motion routines as World.Step without changing the live game.
type demoMotionForecast struct {
	player      PlayerMotionState
	scroll      ScrollState
	rewind      TerrainRewind
	firstMiddle FirstMiddleState
	firstArena  bool
}

func newDemoMotionForecast(w *World) demoMotionForecast {
	forecast := demoMotionForecast{player: w.Player, rewind: w.Rewind,
		scroll: ScrollState{Y: w.ScrollY, Minimum: w.MinimumScrollY, Maximum: w.MaximumScrollY, DeviationPasses: w.ScrollDeviationPasses}}
	if w.Level.Number == 1 && w.FirstMiddle != nil {
		forecast.firstArena, forecast.firstMiddle = true, *w.FirstMiddle
	}
	return forecast
}

func (s *demoMotionForecast) advance(w *World, input MotionInput) bool {
	if w.Level.Number == 3 && s.scroll.Y > 208 && (w.ThirdMiddle == nil || w.ThirdMiddle.Defeated) {
		s.scroll.Maximum = demoScrollMaximum(w, s.scroll.Y, s.scroll.Maximum)
	}
	if w.Level.Number == 2 && w.secondBackward && w.secondDefenseRemaining > 0 {
		// Living source node callbacks reopen this reverse bound every pass.
		s.scroll.Maximum = max(s.scroll.Maximum, 2880)
	}
	if s.firstArena {
		event := s.firstMiddle.Advance(s.scroll.Y, s.scroll.Maximum, s.scroll.Y+s.player.Y, [16]int{})
		s.scroll.Y, s.scroll.Maximum = event.Scroll, event.Maximum
	}
	s.player.SpeedTier = w.Equipment.SpeedTier
	s.player.ScrollStep = w.BaseScrollStep
	touching := func() bool {
		return w.Dive.Phase == 0 && w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(s.player.X, s.player.Y, s.scroll.Y, *w.Level.PlayerStencil)
	}
	handled, crushed := false, false
	if w.Dive.Phase == 0 {
		handled, crushed = s.rewind.Advance(&s.player, s.scroll.Y, w.BaseScrollStep, touching())
	}
	if crushed {
		return false
	}
	if !handled {
		s.player.Advance(input, MotionContext{ScrollY: s.scroll.Y, VisitedScrollY: s.scroll.Maximum, BaseScrollStep: w.BaseScrollStep})
		s.rewind.Record(s.scroll.Y, s.player.X, s.player.Y)
		if touching() {
			s.rewind.Timer, s.player.Inertia = 1, 0
			s.scroll.Maximum = max(s.scroll.Maximum, s.scroll.Y+16)
			return false
		}
	}
	s.scroll.Advance(s.player.ScrollStep, w.BaseScrollStep, input.Down)
	return !touching()
}

// demoRouteMotion can turn within the horizon instead of requiring one held
// direction to clear a narrow corner. The small fixed beam bounds planner work.
func demoRouteMotion(w *World, x, worldY int) MotionInput {
	return demoRouteMotionWithOptions(w, x, worldY, 0, false)
}

func demoRouteMotionWithOptions(w *World, x, worldY, comfortY int, avoidShots bool) MotionInput {
	type branch struct {
		motion demoMotionForecast
		first  int
		score  float64
	}
	var current, next [8]branch
	current[0] = branch{motion: newDemoMotionForecast(w), first: -1}
	count := 1
	for depth := 0; depth < 6; depth++ {
		nextCount := 0
		for index := 0; index < count; index++ {
			for action, input := range demoDirections {
				candidate := current[index]
				actorCamera := candidate.motion.scroll.Y
				if !candidate.motion.advance(w, input) {
					continue
				}
				if candidate.first < 0 {
					candidate.first = action
				}
				player, scroll := candidate.motion.player, candidate.motion.scroll
				candidate.score += float64(absDemo(player.X-x) + absDemo(player.Y+scroll.Y-worldY)*2)
				if comfortY != 0 {
					candidate.score += float64(absDemo(player.Y-comfortY) * 2)
				}
				bounds := thirdMiddlePlayerBounds(w, player)
				for _, actor := range w.Actors {
					if !actor.Active || actor.Collision.Empty() || actor.ActorList != "moving" && actor.ActorList != "scenery" {
						continue
					}
					dx, dy := int(actor.X-actor.PreviousX)*(depth+1), int(actor.Y-actor.PreviousY)*(depth+1)
					other := actor.Collision
					other.Left, other.Right = other.Left+dx, other.Right+dx
					other.Top, other.Bottom = other.Top+dy, other.Bottom+dy
					if predicted, supported := demoActorPrediction(w, actor, depth+1, actorCamera); supported {
						if !predicted.Active {
							continue
						}
						other = predicted.Bounds
					}
					if bounds.Intersects(other) {
						candidate.score += 100000
					}
				}
				if avoidShots {
					for _, shot := range w.Projectiles {
						sx, sy, active := demoProjectilePosition(w, shot, depth+1, w.ScrollDelta)
						if active && sx >= bounds.Left-5 && sx <= bounds.Right+5 && sy >= bounds.Top-5 && sy <= bounds.Bottom+5 {
							candidate.score += 100000
						}
					}
				}
				position := nextCount
				if position == len(next) {
					if candidate.score >= next[position-1].score {
						continue
					}
					position--
				} else {
					nextCount++
				}
				for position > 0 && candidate.score < next[position-1].score {
					next[position] = next[position-1]
					position--
				}
				next[position] = candidate
			}
		}
		if nextCount == 0 {
			break
		}
		current, count = next, nextCount
	}
	if current[0].first >= 0 {
		return demoDirections[current[0].first]
	}
	return MotionInput{}
}
