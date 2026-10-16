package engine

// The source callback replay, not this compact key, establishes future safety.
type retainedGuardKey struct {
	frame                                                     uint64
	player                                                    PlayerMotionState
	camera, minimum, maximum, deviation, delta, base          int
	rewind                                                    TerrainRewind
	equipment                                                 Equipment
	random                                                    RandomState
	divePhase, diveRemaining, material, invulnerable, credits int
	movingWater, fixedWater, nextID                           int
	tiles                                                     uint64
}

type retainedGuardPlan struct {
	forecast       WorldForecast
	world          *World
	before         [7]retainedGuardKey
	input          [6]Input
	at, count, pal int
}

func fourthBranchWindow(w *World) bool {
	return w != nil && w.Level.Number == 4 && w.FourthMiddle == nil && w.ScrollY > 176 && w.PlayerAlive && !w.GameOver && !w.Ready && w.ScreenClearFrames == 0 && !w.ShopReady && !w.ExitReady && !w.LevelFinished && w.PendingExitDrops == 0 && !w.stepContinuation.active && w.Coverage != nil && w.Level.PlayerStencil != nil
}

func retainedGuardStateKey(w *World) retainedGuardKey {
	hash := uint64(14695981039346656037)
	for _, tile := range w.Coverage.Map {
		hash ^= uint64(tile)
		hash *= 1099511628211
	}
	return retainedGuardKey{w.Frame, w.Player, w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.ScrollDeviationPasses, w.ScrollDelta, w.BaseScrollStep, w.Rewind, w.Equipment, w.RandomState(), w.Dive.Phase, w.Dive.Remaining, w.MaterializationFrames, w.InvulnerableFrames, w.ContinueCredits, w.cursor.MovingHighWater, w.cursor.FixedHighWater, w.nextActorID, hash}
}

func fourthBranchSafe(w *World, result ForecastResult, previous int) bool {
	return fourthBranchWindow(w) && result.Boundary == ForecastRunning && result.Alive && result.Shield >= previous && w.Rewind.Timer == 0 && !w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil)
}

// Only an already chosen, changing fallback can install a branch. The exact
// held ordinary input is checked for all six source passes before retention.
func (p *PresentationPilot) captureFourthOpeningBranch(w *World, input Input) {
	if !fourthBranchWindow(w) || input.Dive {
		return
	}
	if p.fourthBranch == nil {
		p.fourthBranch = &retainedGuardPlan{}
	}
	plan := p.fourthBranch
	plan.count = 0
	if err := plan.forecast.Load(w); err != nil {
		return
	}
	pal := thirdMiddlePALRefreshes(p.PALRefreshes)
	var keys [7]retainedGuardKey
	keys[0] = retainedGuardStateKey(w)
	for pass := 0; pass < 6; pass++ {
		previous := plan.forecast.State().Equipment.Shield
		for range pal {
			plan.forecast.AdvancePALTick()
		}
		result, err := plan.forecast.Advance(input)
		if err != nil || !fourthBranchSafe(plan.forecast.State(), result, previous) {
			return
		}
		keys[pass+1] = retainedGuardStateKey(plan.forecast.State())
	}
	if fourthOpeningTerminalContactUnsafe(plan.forecast.State()) {
		return
	}
	plan.world, plan.before, plan.at, plan.count, plan.pal = w, keys, 1, 6, pal
	for i := range plan.input {
		plan.input[i] = input
	}
}

// Repeated calls for the same before-state return the same command. Every new
// live step verifies ownership/key, then replays all remaining exact callbacks.
func (p *PresentationPilot) continueFourthOpeningBranch(w *World) (Input, bool) {
	plan := p.fourthBranch
	if plan == nil || plan.count == 0 {
		return Input{}, false
	}
	if !fourthBranchWindow(w) || plan.world != w || plan.pal != thirdMiddlePALRefreshes(p.PALRefreshes) {
		plan.count = 0
		return Input{}, false
	}
	key, index := retainedGuardStateKey(w), plan.at
	repeated := index > 0 && key == plan.before[index-1]
	if repeated {
		index--
	}
	if index >= plan.count || key != plan.before[index] {
		plan.count = 0
		return Input{}, false
	}
	if err := plan.forecast.Load(w); err != nil {
		plan.count = 0
		return Input{}, false
	}
	for pass := index; pass < plan.count; pass++ {
		previous := plan.forecast.State().Equipment.Shield
		for range plan.pal {
			plan.forecast.AdvancePALTick()
		}
		result, err := plan.forecast.Advance(plan.input[pass])
		state := plan.forecast.State()
		if err != nil || !fourthBranchSafe(state, result, previous) || retainedGuardStateKey(state) != plan.before[pass+1] {
			plan.count = 0
			return Input{}, false
		}
	}
	if fourthOpeningTerminalContactUnsafe(plan.forecast.State()) {
		plan.count = 0
		return Input{}, false
	}
	if !repeated {
		plan.at = index + 1
	}
	return plan.input[index], true
}

func (p *PresentationPilot) clearFourthOpeningBranch() {
	if p.fourthBranch != nil {
		p.fourthBranch.count = 0
		p.fourthBranch.world = nil
	}
}
