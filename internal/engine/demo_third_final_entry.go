package engine

// thirdFinalEntryPreparation follows the native final checkpoint's actual
// restart coordinate before the first launch can reach the left terrain fork.
func (p *PresentationPilot) thirdFinalEntryPreparation(w *World) (MotionInput, bool) {
	if w == nil || w.Level.Number != 3 || w.ScrollY > 240 || w.ScrollY <= 176 || w.Ready || w.GameOver || !w.PlayerAlive || w.Dive.Phase != 0 || p.planner.Config.DisableBossAlignment || w.ThirdMiddle == nil || !w.ThirdMiddle.Defeated || w.ThirdFinal == nil || w.ThirdFinal.Defeated || w.ThirdFinal.LaunchCount > 1 || w.PendingExitDrops != 0 || w.ShopReady || w.Coverage == nil || w.Level.PlayerStencil == nil || w.Level.FixedSprites == nil || w.Level.FixedSprites.Third == nil || w.Level.Rules == nil || w.cursor.FixedHighWater > 736 || w.cursor.MovingHighWater > 576 {
		return MotionInput{}, false
	}
	for _, actor := range w.Actors {
		if actor.Active && (actor.thirdCannon != nil && actor.thirdCannon.X == 128 && actor.thirdCannon.WorldY == 640 || actor.path != nil && actor.path.ID == 55 && actor.part != nil && actor.part.ResourceTag == 228) {
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
	x, worldY := 0, 0
	for _, checkpoint := range w.Level.Rules.Checkpoints {
		if checkpoint.TriggerY == 176 {
			// RestartCheckpoint uses Y176, independently of the record's 360.
			x, worldY = checkpoint.PlayerX, checkpoint.TriggerY+176
			break
		}
	}
	if x == 0 || w.Coverage.Touches(x, worldY, 0, *w.Level.PlayerStencil) {
		return MotionInput{}, false
	}
	if motion, continued := p.planner.nativeMotion.continueRoute(w, x, worldY); continued {
		return motion, true
	}
	if p.planner.navigation == nil {
		p.planner.navigation = &demoNavigation{}
	}
	n := p.planner.navigation
	n.practiced = true
	waypointX, waypointY, found := n.pointWaypoint(w, x, worldY)
	if !found {
		return MotionInput{}, false
	}
	if motion, found := p.planner.nativeMotion.command(w, waypointX, waypointY, x, worldY); found {
		return motion, true
	}
	return demoRouteMotionWithOptions(w, waypointX, waypointY, 136, true), true
}
