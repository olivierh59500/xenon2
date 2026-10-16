package engine

// demoProjectilePosition predicts a live shot without advancing its source
// state. Turning shots own a separate controller and must retain its thresholds,
// fractional movement and removal boundaries rather than the generic motion.
func demoProjectilePosition(w *World, shot *WorldProjectile, passes, scrollDelta int) (x, y int, alive bool) {
	if shot == nil || !shot.Active || passes < 0 {
		return 0, 0, false
	}
	if shot.turning != nil {
		if w == nil || w.Level.FixedSprites == nil || w.Level.FixedSprites.Projectile == nil {
			return 0, 0, false
		}
		state := *shot.turning
		for range passes {
			if _, err := state.Advance(w.Level.FixedSprites.Projectile, FixedProjectileInputs{ScrollDelta: scrollDelta}, w.movingSpriteBoxes[state.Sprite]); err != nil || state.Removed {
				return 0, 0, false
			}
		}
		return int(state.Motion.X >> 16), int(state.Motion.Y >> 16), !state.Removed
	}
	motion := shot.Motion
	for range passes {
		// Keep the existing conservative linear projection for ordinary shots.
		// Only the specialized controller needs a new lifetime/turning policy.
		if _, err := motion.Advance(scrollDelta); err != nil {
			return 0, 0, false
		}
	}
	return int(motion.X >> 16), int(motion.Y >> 16), true
}
