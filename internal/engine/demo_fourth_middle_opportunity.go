package engine

// fourthMiddleRearShotOpportunity observes the first new native Rear volley.
// It is an additional opportunity only; a false result never vetoes primary.
// Only the proposed first motion is applied; later source passes coast.
func fourthMiddleRearShotOpportunity(w *World, motion MotionInput, forecast *WorldForecast, pal int) (bool, bool) {
	if w == nil || w.Level.Number != 4 || w.FourthMiddle == nil || w.FourthMiddle.Defeated || w.Weapons == nil || w.Equipment.Rear.Item != ItemRearShot || w.Equipment.Rear.Tier < 0 || w.Equipment.Rear.Tier > 2 {
		return false, false
	}
	if forecast == nil {
		forecast = &WorldForecast{}
	}
	if err := forecast.Load(w); err != nil {
		return false, true
	}
	if pal <= 0 {
		pal = 3
	}
	baseline := w.nextActorID
	firstID := 0
	useful := false
	observe := func(event WeaponPointImpact) {
		if event.Kind != "small-shot" || event.OwnerSlot != 5 || event.ProjectileID <= baseline {
			return
		}
		if firstID == 0 {
			firstID = event.ProjectileID
		}
		if event.ProjectileID != firstID {
			return
		}
		_, target := presentationFirstPointImpact(forecast.State(), event.X, event.Y)
		useful = useful || target
	}
	for future := 0; future < 18; future++ {
		input := Input{Fire: true}
		if future == 0 {
			input.Motion = motion
		}
		for range pal {
			forecast.AdvancePALTick()
		}
		result, err := forecast.AdvanceObserved(input, observe)
		if useful {
			return true, true
		}
		if err != nil || result.Boundary != ForecastRunning {
			return false, true
		}
		if firstID != 0 {
			active := false
			for _, p := range forecast.State().Weapons.projectiles {
				active = active || p.Render.ID == firstID && p.Render.Active
			}
			if !active {
				return false, true
			}
		}
	}
	return false, true
}

// fourthMiddleCannonShotOpportunity observes the first new native left Cannon volley.
// It is an additional opportunity only; a false result never vetoes primary or Rear.
// Only the proposed first motion is applied; later source passes coast.
func fourthMiddleCannonShotOpportunity(w *World, motion MotionInput, forecast *WorldForecast, pal int) (bool, bool) {
	if w == nil || w.Level.Number != 4 || w.FourthMiddle == nil || w.FourthMiddle.Defeated || w.Weapons == nil || w.Equipment.Mounts[0].Item != ItemCannon {
		return false, false
	}
	if forecast == nil {
		forecast = &WorldForecast{}
	}
	if err := forecast.Load(w); err != nil {
		return false, true
	}
	if pal <= 0 {
		pal = 3
	}
	baseline := w.nextActorID
	firstID := 0
	useful := false
	observe := func(event WeaponRectImpact) {
		if event.Kind != "cannon-ball" || event.OwnerSlot != 1 || event.ProjectileID <= baseline {
			return
		}
		if firstID == 0 {
			firstID = event.ProjectileID
		}
		if event.ProjectileID != firstID {
			return
		}
		useful = useful || guardianRectImpactUseful(forecast.State(), event)
	}
	for future := 0; future < 32; future++ {
		input := Input{Fire: true}
		if future == 0 {
			input.Motion = motion
		}
		for range pal {
			forecast.AdvancePALTick()
		}
		result, err := forecast.AdvanceWeaponObserved(input, nil, observe)
		if useful {
			return true, true
		}
		if err != nil || result.Boundary != ForecastRunning {
			return false, true
		}
		if firstID != 0 {
			active := false
			for _, p := range forecast.State().Weapons.projectiles {
				active = active || p.Render.ID == firstID && p.Render.Active
			}
			if !active {
				return false, true
			}
		}
	}
	return false, true
}
