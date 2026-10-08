package engine

// fifthOpeningPractice retains a rehearsed ordinary-input route. It admits
// only its earned entry profile and checks each proposed source step against
// the recorded outcome before returning controls. It never changes live state.
type fifthOpeningPractice struct {
	world    *World
	forecast WorldForecast
}

func fifthPracticeMarker(w *World) uint64 {
	h := uint64(14695981039346656037)
	word := func(v uint64) {
		for range 8 {
			h ^= v & 255
			h *= 1099511628211
			v >>= 8
		}
	}
	integers := func(values ...int) {
		for _, v := range values {
			word(uint64(v))
		}
	}
	flag := func(v bool) {
		if v {
			word(1)
		} else {
			word(0)
		}
	}
	loadout := func(e WeaponLoadout) {
		for _, s := range [7]WeaponSlot{e.Primary, e.Mounts[0], e.Mounts[1], e.Mounts[2], e.Mounts[3], e.Rear, e.Side} {
			integers(int(s.Item), s.Tier, s.MaxTier)
			word(uint64(s.Serial))
		}
	}
	e := w.Equipment
	word(w.Frame)
	integers(w.Level.Number, w.Player.X, w.Player.Y, w.Player.Inertia, w.Player.SpeedTier, w.Player.ScrollStep,
		w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY, w.ScrollDelta, w.ScrollDeviationPasses, w.BaseScrollStep,
		w.Checkpoint.ScrollY, w.Checkpoint.PlayerX, w.Checkpoint.WorldY, w.Checkpoint.Money, w.Money, w.Score, w.DisplayScore,
		e.Shield, e.Lives, e.SpeedTier, e.FirePeriod, e.FireAdvance, e.DiveCharges, e.SuperFrames, e.ShadesFrames,
		w.ContinueCredits, w.MaterializationFrames, w.InvulnerableFrames, w.PendingExitDrops, w.ScreenClearFrames,
		w.Dive.Phase, w.Dive.Remaining, w.Rewind.Timer, w.nextActorID, w.fire.Remaining, w.fire.Period, w.fire.Advance,
		w.cursor.MovingHighWater, w.cursor.FixedHighWater)
	word(uint64(e.NextWeaponSerial))
	r := w.RandomState()
	word(uint64(r.A))
	word(uint64(r.B))
	flag(w.Player.ScrollReverseRequested)
	flag(e.Protection)
	flag(e.SuperLoadoutActive)
	flag(w.previousFire)
	flag(w.blockedFireUntilRelease)
	flag(w.fire.Pending)
	loadout(e.WeaponLoadout)
	loadout(e.SavedLoadout)
	loadout(w.Checkpoint.Loadout)
	if w.FifthMiddle == nil {
		word(0)
	} else {
		word(1)
		flag(w.FifthMiddle.Defeated)
		for _, part := range w.FifthMiddle.Parts {
			integers(part.X, part.Y, part.Clock, part.MoveRemaining, part.Health, int(part.Heading), int(part.FireAccumulator), int(part.SecondaryAccumulator))
			flag(part.Active)
			flag(part.Destroyed)
		}
	}
	return h
}

func fifthPracticeWindow(w *World, pal int) bool {
	return w != nil && w.Level.Number == 5 && w.Frame >= 1 && w.Frame <= uint64(len(fifthOpeningControls)) &&
		w.PlayerAlive && !w.GameOver && !w.Ready && !w.Cheats.Enabled() && w.Dive.Phase == 0 &&
		w.ScreenClearFrames == 0 && !w.stepContinuation.active && !w.ShopReady && !w.ExitReady &&
		!w.LevelFinished && w.PendingExitDrops == 0 && (w.FifthMiddle == nil || !w.FifthMiddle.Defeated) && w.FifthFinal == nil &&
		w.Level.Ships != nil && w.Level.PlayerStencil != nil && w.Coverage != nil && w.Weapons != nil && thirdMiddlePALRefreshes(pal) == 3
}

func fifthPracticeControl(code byte) Input {
	return Input{Motion: MotionInput{Up: code&1 != 0, Down: code&2 != 0, Left: code&4 != 0, Right: code&8 != 0}, Fire: code&16 != 0, Dive: code&32 != 0}
}

func (p *PresentationPilot) fifthPracticedOpeningInput(w *World) (Input, bool) {
	if !fifthPracticeWindow(w, p.PALRefreshes) {
		p.fifthPractice = nil
		return Input{}, false
	}
	index := int(w.Frame) - 1
	if fifthPracticeMarker(w) != fifthOpeningMarkers[index] {
		p.fifthPractice = nil
		return Input{}, false
	}
	if p.fifthPractice == nil {
		if index != 0 {
			return Input{}, false
		}
		p.fifthPractice = &fifthOpeningPractice{world: w}
	}
	q := p.fifthPractice
	if q.world != w || q.forecast.Load(w) != nil {
		p.fifthPractice = nil
		return Input{}, false
	}
	input := fifthPracticeControl(fifthOpeningControls[index])
	for range 3 {
		q.forecast.AdvancePALTick()
	}
	result, err := q.forecast.Advance(input)
	if err != nil || result.Boundary != ForecastRunning || !result.Alive || result.Lives != w.Equipment.Lives || fifthPracticeMarker(q.forecast.State()) != fifthOpeningMarkers[index+1] {
		p.fifthPractice = nil
		return Input{}, false
	}
	return input, true
}
