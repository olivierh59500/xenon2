package engine

// demoActorView is the actor geometry after a supported controller forecast.
// Unsupported specialized controllers return false instead of a guessed path.
type demoActorView struct {
	X, Y            int
	Sprite          string
	Bounds          CollisionRect
	Active, Visible bool
}

// demoActorPrediction copies ordinary paths, world anchors or sweeper callbacks.
// cameraY is the camera at the future actor phase, before that pass scrolls.
// Path positions are screen-relative; their source callback does not scroll them.
// Random path branches use a copy of the current stream, without forecasting
// unrelated actors' random draws or later combat deaths.
func demoActorPrediction(w *World, actor *WorldActor, passes, cameraY int) (demoActorView, bool) {
	if actor != nil && actor.fifthColumn != nil {
		return demoFifthColumnPrediction(w, actor, passes)
	}
	if w == nil || actor == nil || passes < 0 || passes > 18 || actor.part == nil || actor.ActorList != "moving" {
		return demoActorView{}, false
	}
	if actor.part.HeadingShift < 0 || len(actor.part.EntryAnimations) != 0 && len(actor.part.EntryAnimations) < 4 {
		return demoActorView{}, false
	}
	// These source callbacks take priority over the generic or fixed fields.
	if actor.firstGuardian || actor.secondGuardian || actor.firstSegment != 0 || actor.secondNode != nil || actor.secondSegment != nil || actor.secondMinion != nil || actor.firstMiddleSentinel || actor.firstMiddleAnchor != nil || actor.firstMiddleFollower != nil || actor.thirdMiddlePart != 0 || actor.thirdFinalMember != nil || actor.thirdCrawler != nil || actor.thirdCannon != nil || actor.thirdChainSentinel || actor.thirdChainPart != 0 || actor.fourthIndex != 0 || actor.fourthFalling != nil || actor.fourthChild != nil || actor.fourthCrawler != nil || actor.fifthIndex != 0 || actor.fifthFormation != nil || actor.fifthSeeking != nil || actor.fifthTile != nil || actor.fixedAiming != nil || actor.fixedTileState != nil || actor.fixedHatch != nil || actor.fixedPod != nil || actor.podCreature != nil || actor.hatchCreature != nil {
		return demoActorView{}, false
	}
	if actor.fixedKind != nil {
		if actor.fixed && actor.fixedKind.Behavior == "horizontal-sweeper" {
			return demoSweeperPrediction(w, actor, passes, cameraY)
		}
		return demoActorView{}, false
	}
	path := false
	switch actor.part.MotionMode {
	case "path", "path-heading-frames", "path-entry-edge-frames":
		path = true
		if actor.fixed || actor.path == nil || w.Level.Paths == nil {
			return demoActorView{}, false
		}
	case "world-anchored":
		if !actor.fixed || actor.fixedKind != nil {
			return demoActorView{}, false
		}
	default:
		return demoActorView{}, false
	}
	state, random := *actor, w.RandomState()
	for range passes {
		if !state.Active {
			break
		}
		state.Visible = true
		state.animationState.Advance(state.animation)
		if path {
			if err := state.motion.Advance(state.path, &w.Level.Paths.SineTable, func() uint16 { return uint16(random.Next()) }); err != nil {
				return demoActorView{}, false
			}
			state.X, state.Y = float64(state.motion.X>>16), float64(state.motion.Y>>16)
			state.Active = state.motion.Active
		} else {
			state.Y = float64(state.mapY - cameraY)
		}
		if len(state.part.EntryAnimations) != 0 && !state.entrySelected {
			state.animation = state.part.EntryAnimations[entryEdge(int(state.X), int(state.Y))]
			state.animationState = NewAnimation(state.animation)
			state.entrySelected = true
		}
		state.selectSprite()
		box, ok := w.movingSpriteBoxes[state.Sprite]
		if !ok {
			return demoActorView{}, false
		}
		state.Collision = ActorCollisionRect(box, int(state.X), int(state.Y))
		if path && state.Active {
			// This actor's own firing callback precedes its next path update.
			// Its direction has no effect on the number of random draws.
			if _, _, err := state.fire.Tick(random.Next, w.Player.X-int(state.X), w.Player.Y-int(state.Y)); err != nil {
				return demoActorView{}, false
			}
		}
	}
	return demoActorView{X: int(state.X), Y: int(state.Y), Sprite: state.Sprite, Bounds: state.Collision, Active: state.Active, Visible: state.Visible}, true
}

// demoSweeperPrediction copies the source stop, attack and reversal callbacks.
// It assumes a constant current scroll delta between future actor phases and
// the ordinary sixteen-pixel camera buffer. Shots do not change its motion.
func demoSweeperPrediction(w *World, actor *WorldActor, passes, cameraY int) (demoActorView, bool) {
	if passes > 0 && cameraY != w.ScrollY-(passes-1)*w.ScrollDelta {
		return demoActorView{}, false
	}
	state, kind, random := actor.fixedState, *actor.fixedKind, w.RandomState()
	view := demoActorView{X: int(actor.X), Y: int(actor.Y), Sprite: actor.Sprite, Bounds: actor.Collision, Active: actor.Active, Visible: actor.Visible}
	camera := cameraY + (passes-1)*w.ScrollDelta
	maximum := w.MaximumScrollY
	for range passes {
		if !view.Active {
			break
		}
		maximum = demoScrollMaximum(w, camera, maximum)
		StepFixedSpriteMotion(&state, kind, FixedSpriteInputs{ScrollDelta: w.ScrollDelta, ScrollY: camera, MaximumScrollY: maximum, PlayerX: w.Player.X, PlayerY: w.Player.Y}, &random)
		sprite := state.Animation.Sprite(FixedSpriteAnimation(state, kind))
		box, ok := w.movingSpriteBoxes[sprite]
		if !ok {
			return demoActorView{}, false
		}
		view = demoActorView{X: state.X, Y: state.Y, Sprite: sprite, Bounds: ActorCollisionRect(box, state.X, state.Y), Active: !state.Removed, Visible: true}
		camera -= w.ScrollDelta
		maximum = min(maximum, camera+16)
	}
	return view, true
}
