package engine

// presentationAuxiliaryShotOpportunity also checks the installed rear and side
// guns. Releasing fire just because the forward ray misses can prevent those
// weapons from clearing a formation passing beside or behind the ship.
func presentationAuxiliaryShotOpportunity(w *World, motion MotionInput) bool {
	var shots [4]SmallShot
	count := 0
	forecast := newDemoMotionForecast(w)
	forecast.advance(w, motion)
	ship := forecast.player
	for _, weapon := range []WeaponSlot{w.Equipment.Rear, w.Equipment.Side} {
		if weapon.Item != ItemRearShot && weapon.Item != ItemSideShot {
			continue
		}
		if weapon.Item == ItemRearShot && ship.Y+18+9 >= 192 {
			continue
		}
		out, err := AppendSmallWeaponShots(shots[:count], weapon, ship.X, ship.Y)
		if err != nil {
			continue
		}
		count = len(out)
	}
	if count == 0 {
		return false
	}
	for _, actor := range w.Actors {
		bounds, ok := presentationTargetBounds(w, actor)
		if !ok {
			continue
		}
		for future := 1; future <= 24; future++ {
			live := false
			for _, shot := range shots[:count] {
				x, y := shot.X+shot.VelocityX*future, shot.Y+shot.VelocityY*future
				live = live || x >= 0 && x < 312 && y >= 0 && y < 192
			}
			if !live {
				break
			}
			predicted := bounds
			if view, supported := demoActorPrediction(w, actor, future, w.ScrollY-(future-1)*w.BaseScrollStep); supported {
				if !view.Active || !view.Visible {
					continue
				}
				predicted = view.Bounds
			} else {
				dx, dy := int(actor.X-actor.PreviousX)*future, int(actor.Y-actor.PreviousY)*future
				predicted.Left, predicted.Right, predicted.Top, predicted.Bottom = predicted.Left+dx, predicted.Right+dx, predicted.Top+dy, predicted.Bottom+dy
			}
			for _, shot := range shots[:count] {
				x, y := shot.X+shot.VelocityX*future, shot.Y+shot.VelocityY*future
				if x >= 0 && x < 312 && y >= 0 && y < 192 && x >= predicted.Left && x <= predicted.Right && y >= predicted.Top && y <= predicted.Bottom {
					return true
				}
			}
		}
	}
	return false
}
