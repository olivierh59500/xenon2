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
	return s.advanceWithTouchCache(w, input, nil)
}

func (s *demoMotionForecast) advanceWithTouchCache(w *World, input MotionInput, cache *nativeMotionTouchCache) bool {
	if w.Level.Number == 5 && w.FifthFinal != nil && w.fifthFinalArt != nil && w.fifthFinalActors[0] != nil && w.fifthFinalActors[0].Active {
		// The final controller renews its source arena bound before scrolling,
		// independently of the sixteen-pixel buffer retained by the last pass.
		s.scroll.Maximum = w.fifthFinalArt.MotionParameters["maximum_scroll"]
	}
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
		if w.Dive.Phase != 0 || w.Coverage == nil || w.Level.PlayerStencil == nil {
			return false
		}
		if cache != nil {
			return cache.touches(w, s.player.X, s.player.Y, s.scroll.Y)
		}
		return w.Coverage.Touches(s.player.X, s.player.Y, s.scroll.Y, *w.Level.PlayerStencil)
	}
	handled, crushed := false, false
	if w.Dive.Phase == 0 {
		// Advance only consults contact while Timer < 0.
		contact := false
		if s.rewind.Timer < 0 {
			contact = touching()
		}
		handled, crushed = s.rewind.Advance(&s.player, s.scroll.Y, w.BaseScrollStep, contact)
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
	if w.Level.Number == 4 && w.FourthMiddle != nil && !w.FourthMiddle.Defeated && !w.FourthMiddle.Parts[4].Disabled && w.fourthMiddleActors[4] != nil && w.fourthMiddleActors[4].Active {
		s.scroll.Maximum = max(s.scroll.Maximum, 2480)
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
	if avoidShots {
		return demoRouteMotionAvoidingShots(w, x, worldY, comfortY, 0)
	}
	return demoRouteMotionSearch(w, x, worldY, comfortY, nil, 0)
}

func demoRouteMotionWithClearance(w *World, x, worldY, comfortY, clearance int) MotionInput {
	return demoRouteMotionAvoidingShots(w, x, worldY, comfortY, clearance)
}

type demoRouteRisk struct {
	bounds            CollisionRect
	active, supported bool
}

type demoRouteRisks struct {
	actors, shots *[6][ActorPoolCapacity]demoRouteRisk
}

func demoRouteActorCacheable(actor *WorldActor) bool {
	if actor.fixedKind != nil {
		return false
	}
	// An unselected world-anchor entry clip can depend on the candidate camera.
	return actor.part == nil || actor.part.MotionMode != "world-anchored" || actor.entrySelected || len(actor.part.EntryAnimations) == 0
}

func demoRouteMotionAvoidingShots(w *World, x, worldY, comfortY, clearance int) MotionInput {
	var actorRisks, shotRisks [6][ActorPoolCapacity]demoRouteRisk
	risks := demoRouteRisks{actors: &actorRisks, shots: &shotRisks}
	for index, actor := range w.Actors {
		if index >= ActorPoolCapacity {
			break
		}
		if !demoActorHazard(actor) || !demoRouteActorCacheable(actor) {
			continue
		}
		for depth := range risks.actors {
			view, supported := demoActorPrediction(w, actor, depth+1, w.ScrollY)
			risks.actors[depth][index] = demoRouteRisk{bounds: view.Bounds, active: view.Active, supported: supported}
		}
	}
	for index, shot := range w.Projectiles {
		if index >= ActorPoolCapacity {
			break
		}
		for depth := range risks.shots {
			sx, sy, active := demoProjectilePosition(w, shot, depth+1, w.ScrollDelta)
			risks.shots[depth][index] = demoRouteRisk{bounds: CollisionRect{Left: sx, Top: sy}, active: active, supported: true}
		}
	}
	return demoRouteMotionSearch(w, x, worldY, comfortY, &risks, clearance)
}

func demoRouteMotionSearch(w *World, x, worldY, comfortY int, risks *demoRouteRisks, clearance int) MotionInput {
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
				previousPlayer := candidate.motion.player
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
				if clearance > 0 && !bounds.Empty() {
					// Preserve a small reaction clearance around the source ship
					// prefix; gameplay still uses the unchanged exact collider.
					bounds.Left, bounds.Right = bounds.Left-clearance, bounds.Right+clearance
					bounds.Top, bounds.Bottom = bounds.Top-clearance, bounds.Bottom+clearance
				}
				for actorIndex, actor := range w.Actors {
					if !demoActorHazard(actor) {
						continue
					}
					dx, dy := int(actor.X-actor.PreviousX)*(depth+1), int(actor.Y-actor.PreviousY)*(depth+1)
					other := actor.Collision
					other.Left, other.Right = other.Left+dx, other.Right+dx
					other.Top, other.Bottom = other.Top+dy, other.Bottom+dy
					var predicted demoRouteRisk
					if risks != nil && actorIndex < ActorPoolCapacity && demoRouteActorCacheable(actor) {
						predicted = risks.actors[depth][actorIndex]
						if predicted.supported && actor.part != nil && actor.part.MotionMode == "world-anchored" {
							change := w.ScrollY - actorCamera
							predicted.bounds.Top, predicted.bounds.Bottom = predicted.bounds.Top+change, predicted.bounds.Bottom+change
						}
					} else {
						view, supported := demoActorPrediction(w, actor, depth+1, actorCamera)
						predicted = demoRouteRisk{bounds: view.Bounds, active: view.Active, supported: supported}
					}
					if predicted.supported {
						if !predicted.active {
							continue
						}
						other = predicted.bounds
					}
					contact := bounds
					if actor.fifthColumn != nil {
						contact = thirdMiddlePlayerBounds(w, previousPlayer)
					}
					if contact.Intersects(other) {
						candidate.score += 100000
					}
				}
				if risks != nil {
					for shotIndex, shot := range w.Projectiles {
						var sx, sy int
						var active bool
						if shotIndex < ActorPoolCapacity {
							predicted := risks.shots[depth][shotIndex]
							sx, sy, active = predicted.bounds.Left, predicted.bounds.Top, predicted.active
						} else {
							sx, sy, active = demoProjectilePosition(w, shot, depth+1, w.ScrollDelta)
						}
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
