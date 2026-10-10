package engine

// demoEnemyShotPrediction copies the ordinary projectile callback, including
// its animation-before-motion order, fixed-point fractions and native bounds.
// scrollDelta is supplied by the caller; turning controllers remain separate.
func demoEnemyShotPrediction(w *World, shot *WorldProjectile, passes, scrollDelta int) (demoActorView, bool) {
	if passes < 0 || passes > 18 {
		return demoActorView{}, false
	}
	var steps [18]int
	for index := range passes {
		steps[index] = scrollDelta
	}
	return demoEnemyShotPredictionSteps(w, shot, steps[:passes])
}

// demoEnemyShotPredictionSteps accepts the camera displacement used by each
// future projectile phase. Reversing or clamped scrolling cannot be replaced
// by repeating the live pass's displacement throughout the horizon.
func demoEnemyShotPredictionSteps(w *World, shot *WorldProjectile, steps []int) (demoActorView, bool) {
	if w == nil || shot == nil || shot.turning != nil || len(steps) > 18 {
		return demoActorView{}, false
	}
	state := *shot
	for _, scrollDelta := range steps {
		if !state.Active {
			break
		}
		if len(state.animation.Frames) != 0 {
			state.animationState.Advance(state.animation)
			state.Sprite = state.animationState.Sprite(state.animation)
		}
		var err error
		state.Active, err = state.Motion.Advance(scrollDelta)
		if err != nil {
			return demoActorView{}, false
		}
		state.X, state.Y = float64(state.Motion.X>>16), float64(state.Motion.Y>>16)
	}
	x, y := int(state.X), int(state.Y)
	view := demoActorView{X: x, Y: y, Sprite: state.Sprite, Active: state.Active, Visible: state.Active, Bounds: CollisionRect{Right: -1, Bottom: -1}}
	if !state.Active {
		return view, true
	}
	box, ok := w.enemyShotCollisionBox(state.Sprite, state.Atlas)
	if !ok {
		return demoActorView{}, false
	}
	view.Bounds = ActorCollisionRect(box, x, y)
	return view, true
}

// All ordinary projectile callbacks use the ship prefix published before this
// pass moves it. A clear movement endpoint cannot undo that earlier contact.
func demoPublishedEnemyShotContact(w *World, shot *WorldProjectile, passes int, player PlayerMotionState) bool {
	if w == nil || w.Level.Number < 1 || w.Level.Number > 5 {
		return false
	}
	view, supported := demoEnemyShotPrediction(w, shot, passes, w.ScrollDelta)
	return supported && view.Active && thirdMiddlePlayerBounds(w, player).Intersects(view.Bounds)
}

// demoProjectilePredictionSteps adds the separate turning controller while
// preserving the collision image used before its heading/image change.
func demoProjectilePredictionSteps(w *World, shot *WorldProjectile, steps []int) (demoActorView, bool) {
	if shot == nil || shot.turning == nil {
		return demoEnemyShotPredictionSteps(w, shot, steps)
	}
	if w == nil || len(steps) > 18 || w.Level.FixedSprites == nil || w.Level.FixedSprites.Projectile == nil {
		return demoActorView{}, false
	}
	state := *shot.turning
	bounds := CollisionRect{Right: -1, Bottom: -1}
	for _, delta := range steps {
		if state.Removed {
			break
		}
		box, ok := w.enemyShotCollisionBox(state.Sprite, shot.Atlas)
		if !ok {
			return demoActorView{}, false
		}
		event, err := state.Advance(w.Level.FixedSprites.Projectile, FixedProjectileInputs{ScrollDelta: delta}, box)
		if err != nil {
			return demoActorView{}, false
		}
		bounds = event.Collision
	}
	return demoActorView{X: int(state.Motion.X >> 16), Y: int(state.Motion.Y >> 16), Sprite: state.Sprite, Active: shot.Active && !state.Removed, Visible: shot.Active && !state.Removed, Bounds: bounds}, true
}
