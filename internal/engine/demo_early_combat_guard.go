package engine

// forecastEarlyCombatGuard is shared by the basic and presentation controllers.
// It rehearses ordinary first/second-level commands through isolated callbacks;
// the live world, equipment and random stream remain untouched.
func forecastEarlyCombatGuard(w *World, planned Input, forecast *WorldForecast, pal int) Input {
	second := w.Level.Number == 2 && (w.ScrollY > 1280 || w.secondMiddleReleased)
	if w.Level.Number != 1 && !second || w.Ready || !w.PlayerAlive {
		return planned
	}
	if pal <= 0 {
		pal = 3
	}
	const horizon = 12
	if err := forecast.Load(w); err != nil {
		return planned
	}
	unsafe := false
	for pass := 0; pass < horizon; pass++ {
		for range pal {
			forecast.AdvancePALTick()
		}
		result, err := forecast.Advance(planned)
		if err != nil {
			return planned
		}
		if result.Shield < w.Equipment.Shield || !result.Alive {
			unsafe = true
			if !second || !result.Alive {
				break
			}
		}
		if result.Boundary != ForecastRunning || pass == horizon-1 {
			if unsafe {
				break
			}
			return planned
		}
	}
	intended := forecast.State()
	intendedX, intendedY := intended.Player.X, intended.Player.Y+intended.ScrollY
	if second && w.secondMiddleReleased && w.ScrollY <= 352 {
		intendedX, intendedY = 160, 416
		if absDemo(w.Player.X-160) > 36 {
			intendedY = 448
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
		var result ForecastResult
		for range horizon {
			for range pal {
				forecast.AdvancePALTick()
			}
			var err error
			result, err = forecast.Advance(input)
			if err != nil {
				return planned
			}
			if result.Boundary != ForecastRunning {
				break
			}
		}
		end := forecast.State()
		distance := absDemo(end.Player.X-w.Player.X) + absDemo(end.Player.Y-120)
		if second {
			distance = absDemo(end.Player.X-intendedX) + absDemo(end.Player.Y+end.ScrollY-intendedY)*2
		}
		progress := end.ScrollY
		if result.Alive && !bestAlive || result.Alive == bestAlive && (result.Shield > bestShield || result.Shield == bestShield && (distance < bestDistance || distance == bestDistance && progress < bestProgress)) {
			best, bestAlive, bestShield, bestProgress, bestDistance = input, result.Alive, result.Shield, progress, distance
		}
	}
	return best
}
