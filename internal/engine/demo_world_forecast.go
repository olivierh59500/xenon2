package engine

// forecastOpeningGuard evaluates the first level, third routes/final encounter
// and fourth opening/middle encounter through isolated World.Step callbacks.
// It retains the planned action unless its existing lookahead is unsafe.
func (p *PresentationPilot) forecastOpeningGuard(w *World, planned Input) (answer Input) {
	defer func() {
		if answer.Motion != planned.Motion {
			p.captureFourthOpeningBranch(w, answer)
		}
	}()
	opening := w.Checkpoint.ScrollY > 3408 && w.ThirdMiddle == nil
	corridor := w.ThirdMiddle != nil && w.ThirdMiddle.Defeated && w.PendingExitDrops == 0 && !w.ShopReady && w.ScrollY > 208
	final := w.ThirdFinal != nil && !w.ThirdFinal.Defeated && w.ThirdFinal.LaunchCount > 0 && w.ScrollY <= 208
	third := w.Level.Number == 3 && (opening || corridor || final)
	fourthOpening := w.Level.Number == 4 && w.FourthMiddle == nil && w.ScrollY > 176
	fourthMiddle := w.Level.Number == 4 && w.FourthMiddle != nil && !w.FourthMiddle.Defeated
	early := w.Level.Number == 1
	if !early && !third && !fourthOpening && !fourthMiddle || w.Ready || !w.PlayerAlive {
		return planned
	}
	forecast := &p.forecast
	pal := p.PALRefreshes
	if pal <= 0 {
		pal = 3
	}
	var sequence [6]MotionInput
	useSequence := false
	if corridor || final || fourthOpening {
		sequence, useSequence = p.planner.nativeMotion.guardSequence(w, planned.Motion)
	}
	// Keep the practiced action unless the real callback lookahead predicts damage.
	if err := forecast.Load(w); err != nil {
		return planned
	}
	for pass := 0; pass < 6; pass++ {
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
		if r.Shield < w.Equipment.Shield || !r.Alive || fourthMiddle && fourthAdmissionTerrainUnsafe(forecast.State()) {
			break
		}
		if fourthOpening && pass == 5 && fourthOpeningTerminalContactUnsafe(forecast.State()) {
			break
		}
		if r.Boundary != ForecastRunning || pass == 5 {
			return planned
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
		for pass := 0; pass < 6; pass++ {
			for range pal {
				forecast.AdvancePALTick()
			}
			var err error
			r, err = forecast.Advance(input)
			if err != nil {
				return planned
			}
			if fourthMiddle && fourthAdmissionTerrainUnsafe(forecast.State()) {
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
