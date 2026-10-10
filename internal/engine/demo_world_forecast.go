package engine

// forecastOpeningGuard evaluates the first level, third routes/final encounter
// and fourth opening/middle/later section through isolated World.Step callbacks.
// It retains the planned action unless its existing lookahead is unsafe.
func (p *PresentationPilot) forecastOpeningGuard(w *World, planned Input) (answer Input) {
	if w.Level.Number == 2 {
		return planned
	}
	return p.forecastCombatGuard(w, planned)
}

func (p *PresentationPilot) forecastSecondGuard(w *World, planned Input) Input {
	if w.Level.Number != 2 {
		return planned
	}
	return p.forecastCombatGuard(w, planned)
}

func (p *PresentationPilot) forecastCombatGuard(w *World, planned Input) (answer Input) {
	defer func() {
		if answer.Motion != planned.Motion {
			p.captureFourthOpeningBranch(w, answer)
		}
	}()
	opening := w.ThirdMiddle == nil
	corridor := w.ThirdMiddle != nil && w.ThirdMiddle.Defeated && w.PendingExitDrops == 0 && !w.ShopReady && w.ScrollY > 208
	final := w.ThirdFinal != nil && !w.ThirdFinal.Defeated && w.ThirdFinal.LaunchCount > 0 && w.ScrollY <= 208
	third := w.Level.Number == 3 && (opening || corridor || final)
	thirdOpening := w.Level.Number == 3 && opening
	fourthOpening := w.Level.Number == 4 && w.FourthMiddle == nil && w.ScrollY > 176
	fourthMiddle := w.Level.Number == 4 && w.FourthMiddle != nil && !w.FourthMiddle.Defeated
	fourthLater := w.Level.Number == 4 && w.FourthMiddle != nil && w.FourthMiddle.Defeated && w.ScrollY > 176
	early := w.Level.Number == 1
	second := w.Level.Number == 2 && (w.ScrollY > 1280 || w.ScrollY < 800 && w.ScrollY > 352 ||
		w.SecondGuardian != nil && !w.SecondGuardian.BodyCollision.Empty() && w.Player.X >= 124 && w.Player.X <= 196)
	if !early && !second && !third && !fourthOpening && !fourthMiddle && !fourthLater || w.Ready || !w.PlayerAlive {
		return planned
	}
	recoverCorridor := corridor && p.thirdCorridorRecoveryNeeded(w)
	forecast := &p.forecast
	pal := p.PALRefreshes
	if pal <= 0 {
		pal = 3
	}
	var sequence [6]MotionInput
	useSequence := false
	horizon := 6
	if w.Level.Number <= 2 {
		// An opportunistic attack or collection turn must leave time to
		// escape the formation that follows it, not only its nearest bullet.
		horizon = 12
	}
	if corridor || final || fourthOpening {
		sequence, useSequence = p.planner.nativeMotion.guardSequence(w, planned.Motion)
	}
	// Keep the practiced action unless the real callback lookahead predicts damage.
	if err := forecast.Load(w); err != nil {
		return planned
	}
	unsafe := false
	for pass := 0; pass < horizon; pass++ {
		for range pal {
			forecast.AdvancePALTick()
		}
		input := planned
		if useSequence {
			input.Motion = sequence[pass]
		}
		r, err := forecast.Advance(input)
		if err != nil {
			return planned
		}
		if r.Shield < w.Equipment.Shield || !r.Alive || (thirdOpening || fourthMiddle) && fourthAdmissionTerrainUnsafe(forecast.State()) {
			unsafe = true
			if !second || !r.Alive {
				break
			}
		}
		if fourthOpening && pass == 5 && fourthOpeningTerminalContactUnsafe(forecast.State()) {
			break
		}
		if r.Boundary != ForecastRunning || pass == horizon-1 {
			if unsafe {
				break
			}
			return planned
		}
	}
	intended := forecast.State()
	intendedX, intendedY := intended.Player.X, intended.Player.Y+intended.ScrollY
	if recoverCorridor {
		if input, ok := p.chooseThirdCorridorBranch(w, planned, sequence, useSequence); ok {
			return input
		}
	}
	best := planned
	bestAlive, bestShield, bestProgress, bestDistance := false, -1, 1000000, 1000000
	for _, motion := range demoDirections {
		if err := forecast.Load(w); err != nil {
			return planned
		}
		input := planned
		input.Motion = motion
		var r ForecastResult
		safe := true
		for pass := 0; pass < horizon; pass++ {
			for range pal {
				forecast.AdvancePALTick()
			}
			var err error
			r, err = forecast.Advance(input)
			if err != nil {
				return planned
			}
			if (thirdOpening || fourthMiddle) && fourthAdmissionTerrainUnsafe(forecast.State()) {
				safe = false
				break
			}
			if r.Boundary != ForecastRunning {
				break
			}
		}
		if !safe {
			continue
		}
		end := forecast.State()
		if fourthOpening && fourthOpeningTerminalContactUnsafe(end) {
			continue
		}
		distance := absDemo(end.Player.X-w.Player.X) + absDemo(end.Player.Y-120)
		if second {
			// Preserve the prepared terrain leg while dodging. Re-centering at
			// screen Y120 can abandon a known junction for the wrong passage.
			distance = absDemo(end.Player.X-intendedX) + absDemo(end.Player.Y+end.ScrollY-intendedY)*2
		}
		progress := end.ScrollY
		if r.Alive && !bestAlive || r.Alive == bestAlive && (r.Shield > bestShield || r.Shield == bestShield && (distance < bestDistance || distance == bestDistance && progress < bestProgress)) {
			best, bestAlive, bestShield, bestProgress, bestDistance = input, r.Alive, r.Shield, progress, distance
		}
	}
	return best
}

func fourthAdmissionTerrainUnsafe(w *World) bool {
	return w.Rewind.Timer != 0 || w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil)
}
