package engine

// presentationGuardianShotOpportunity follows the first new primary volley
// and, in fifth-stage arenas, the first Side Shot volley under the current
// held-fire cadence. The selected motion applies this pass; subsequent passes
// coast while holding fire. Every point query is inspected before its native
// callback, including harmless armor and older-shot effects.
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
	var ids [2][2]int
	var counts [2]int
	var volleyFrames [2]uint64
	side := w.Level.Number == 5 && w.Equipment.Side.Item == ItemSideShot
	horizon := 18
	if side {
		// A horizontal point can cross the full 312-pixel bullet area before
		// retiring. Include the wait for the first pulse if fire is already held.
		horizon = (312+8)/9 + 1
		if w.previousFire && !w.fire.Pending && w.fire.Advance > 0 {
			horizon += max(0, (w.fire.Remaining+w.fire.Advance-1)/w.fire.Advance)
		}
	}
	useful := false
	var mounted [4]int
	observeRectangle := func(event WeaponRectImpact) {
		if w.Level.Number != 5 || event.Kind != "laser" || event.ProjectileID <= w.nextActorID || event.OwnerSlot < 1 || event.OwnerSlot > len(w.Equipment.Mounts) || w.Equipment.Mounts[event.OwnerSlot-1].Item != ItemLaser {
			return
		}
		id := &mounted[event.OwnerSlot-1]
		if *id == 0 {
			*id = event.ProjectileID
		}
		if *id == event.ProjectileID {
			useful = useful || guardianRectImpactUseful(forecast.State(), event)
		}
	}
	observe := func(event WeaponPointImpact) {
		if event.Kind != "small-shot" || event.ProjectileID <= w.nextActorID {
			return
		}
		volley := 0
		if event.OwnerSlot != 0 {
			if !side || event.OwnerSlot != 6 {
				return
			}
			volley = 1
		}
		state := forecast.State()
		if volleyFrames[volley] == 0 {
			volleyFrames[volley] = state.Frame
		}
		tracked := false
		for _, id := range ids[volley][:counts[volley]] {
			tracked = tracked || id == event.ProjectileID
		}
		if !tracked && state.Frame == volleyFrames[volley] && counts[volley] < len(ids[volley]) {
			ids[volley][counts[volley]], counts[volley], tracked = event.ProjectileID, counts[volley]+1, true
		}
		if tracked {
			_, target := presentationFirstPointImpact(state, event.X, event.Y)
			useful = useful || target
		}
	}
	for future := 0; future < horizon; future++ {
		for tick := 0; tick < palRefreshes; tick++ {
			forecast.AdvancePALTick()
		}
		input := Input{Fire: true}
		if future == 0 {
			input.Motion = motion
		}
		var result ForecastResult
		var err error
		if w.Level.Number == 5 {
			result, err = forecast.AdvanceWeaponObserved(input, observe, observeRectangle)
		} else {
			result, err = forecast.AdvanceObserved(input, observe)
		}
		if useful {
			return true, true
		}
		if err != nil || result.Boundary != ForecastRunning {
			return false, true
		}
		if counts[0] != 0 {
			active := false
			for _, projectile := range forecast.State().Weapons.projectiles {
				for _, id := range ids[0][:counts[0]] {
					active = active || projectile.Render.ID == id && projectile.Render.Active
				}
			}
			// A fifth-stage mounted beam or side volley can still be travelling
			// after the primary retires. Its own callback decides useful damage.
			if !active && w.Level.Number != 5 {
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
