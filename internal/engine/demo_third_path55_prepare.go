package engine

// thirdPath55Preparation uses the known bottom-entry formation only after the
// last original two-stage cannon has actually written its destroyed map patch.
func thirdPath55Preparation(w *World) (MotionInput, bool) {
	if w == nil || w.Level.Number != 3 || w.ScrollY > 608 || w.Ready || w.GameOver || !w.PlayerAlive || w.Dive.Phase != 0 || w.ThirdMiddle == nil || !w.ThirdMiddle.Defeated || w.PendingExitDrops != 0 || w.ShopReady || w.Coverage == nil || w.Level.PlayerStencil == nil || w.Level.FixedSprites == nil || w.Level.FixedSprites.Third == nil || w.cursor.FixedHighWater > 736 {
		return MotionInput{}, false
	}
	for _, actor := range w.Actors {
		if actor.Active && actor.thirdCannon != nil && actor.thirdCannon.X == 128 && actor.thirdCannon.WorldY == 640 {
			return MotionInput{}, false
		}
	}
	patch := w.Level.FixedSprites.Third.Cannon.Destroyed
	if patch.Columns != 4 || patch.Rows != 6 || len(patch.Tiles) != 24 || w.Coverage.Columns != 20 || len(w.Coverage.Map) < 46*20 {
		return MotionInput{}, false
	}
	for row := 0; row < patch.Rows; row++ {
		for column := 0; column < patch.Columns; column++ {
			if w.Coverage.Map[(40+row)*20+8+column] != patch.Tiles[row*patch.Columns+column] {
				return MotionInput{}, false
			}
		}
	}
	pending := w.cursor.MovingHighWater > 576
	active := false
	for _, actor := range w.Actors {
		if actor.Active && actor.path != nil && actor.path.ID == 55 && actor.part != nil && actor.part.ResourceTag == 228 {
			active = true
		}
	}
	if !pending && !active || w.Coverage.Touches(160, 100, w.ScrollY, *w.Level.PlayerStencil) {
		return MotionInput{}, false
	}
	return demoRouteMotionWithOptions(w, 160, w.ScrollY+100, 100, true), true
}
