package engine

// presentationGuardianShotOpportunity follows the first new primary volley
// reaching a point query under the current held-fire cadence. The selected
// motion applies this pass; subsequent passes coast while holding fire.
// Every point query is inspected
// before its native callback, including harmless armor and older-shot effects.
// Worlds without the original weapon runtime retain the general aiming path.
func presentationGuardianShotOpportunity(w *World, motion MotionInput, forecast *WorldForecast, palRefreshes int) (opportunity, supported bool) {
	if !presentationGuardianAimSupported(w) {
		return false, false
	}
	if err := forecast.Load(w); err != nil {
		return false, true
	}
	if palRefreshes <= 0 {
		palRefreshes = 3
	}
	var ids [2]int
	count := 0
	var volleyFrame uint64
	useful := false
	observe := func(event WeaponPointImpact) {
		if event.Kind != "small-shot" || event.OwnerSlot != 0 || event.ProjectileID <= w.nextActorID {
			return
		}
		state := forecast.State()
		if volleyFrame == 0 {
			volleyFrame = state.Frame
		}
		tracked := false
		for _, id := range ids[:count] {
			tracked = tracked || id == event.ProjectileID
		}
		if !tracked && state.Frame == volleyFrame && count < len(ids) {
			ids[count], count, tracked = event.ProjectileID, count+1, true
		}
		if tracked {
			_, target := presentationFirstPointImpact(state, event.X, event.Y)
			useful = useful || target
		}
	}
	for future := 0; future < 18; future++ {
		for tick := 0; tick < palRefreshes; tick++ {
			forecast.AdvancePALTick()
		}
		input := Input{Fire: true}
		if future == 0 {
			input.Motion = motion
		}
		result, err := forecast.AdvanceObserved(input, observe)
		if useful {
			return true, true
		}
		if err != nil || result.Boundary != ForecastRunning {
			return false, true
		}
		if count != 0 {
			active := false
			for _, projectile := range forecast.State().Weapons.projectiles {
				for _, id := range ids[:count] {
					active = active || projectile.Render.ID == id && projectile.Render.Active
				}
			}
			if !active {
				return false, true
			}
		}
	}
	return false, true
}

func presentationGuardianAimSupported(w *World) bool {
	if w == nil || w.Weapons == nil || w.Equipment.Primary.Item != ItemForwardShot && w.Equipment.Primary.Item != ItemDoubleShot {
		return false
	}
	return w.Level.Number == 3 && w.ScrollY <= 208 && w.ThirdFinal != nil && !w.ThirdFinal.Defeated && w.ThirdFinal.LaunchCount > 0 ||
		w.Level.Number == 4 && (w.FourthMiddle != nil && !w.FourthMiddle.Defeated || w.FourthFinal != nil && !w.FourthFinal.Defeated) ||
		w.Level.Number == 5 && (w.FifthMiddle != nil && !w.FifthMiddle.Defeated || w.FifthFinal != nil && !w.FifthFinal.Defeated)
}

// presentationFirstPointImpact follows weaponHitPoint's physical moving-list
// order. Harmless armor still consumes an ordinary bullet at its first contact.
func presentationFirstPointImpact(w *World, x, y int) (hit, useful bool) {
	var storage [ActorPoolCapacity]*WorldActor
	for _, actor := range w.orderedMovingActors(&storage) {
		if !actor.Active || actor.ActorList != "moving" || !actor.Collision.Contains(x, y) {
			continue
		}
		bounds, target := presentationTargetBounds(w, actor)
		return true, target && bounds.Contains(x, y)
	}
	return false, false
}
