package engine

// fourthFinalNativePilot compares ordinary six-pass source callback sequences.
type fourthFinalNativePilot struct {
	forecast WorldForecast
	world    *World
	key      fourthFinalNativeKey
	last     Input
}
type fourthFinalNativeKey struct {
	frame                                              uint64
	player                                             PlayerMotionState
	equipment                                          Equipment
	random                                             RandomState
	camera, minimum, maximum, deviation, delta, nextID int
	eyes                                               int
	parts                                              [19]FourthGuardianPart
}

func fourthFinalNativeState(w *World) fourthFinalNativeKey {
	return fourthFinalNativeKey{w.Frame, w.Player, w.Equipment, w.RandomState(), w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.ScrollDeviationPasses, w.ScrollDelta, w.nextActorID, w.FourthFinal.EyesRemaining, w.FourthFinal.Parts}
}

type fourthFinalNativePlan struct {
	valid, alive         bool
	shield, hp, distance int
	first                Input
}

func fourthFinalNativeTarget(w *World) (x, y int) {
	i := 0
	if !w.FourthFinal.Parts[1].Disabled {
		i = 1
	} else if !w.FourthFinal.Parts[2].Disabled {
		i = 2
	}
	r := w.FourthFinal.Parts[i].Collision
	if r.Left >= 320 || r.Empty() {
		return 160, 148
	}
	return (r.Left + r.Right) / 2, 148
}
func fourthFinalNativeBodyUnsafe(w *World) bool {
	if !w.PlayerAlive || w.Dive.Phase != 0 || w.ScreenClearFrames != 0 {
		return false
	}
	r := thirdMiddlePlayerBounds(w, w.Player)
	var order [ActorPoolCapacity]*WorldActor
	for _, a := range w.orderedMovingActors(&order) {
		if !a.Active || a.ActorList != "moving" || !a.Collision.Intersects(r) {
			continue
		}
		if a.part == nil {
			return false
		}
		d := ApplyShieldDamage(w.Equipment.Shield, ContactDamage(a.part.StrongHealth), w.Equipment.Protection, w.InvulnerableFrames != 0 || w.PendingExitDrops != 0 || w.Cheats.InfiniteEnergy)
		return d.Applied && (d.Lost > 0 || d.Destroyed)
	}
	return false
}
func (p *fourthFinalNativePilot) evaluate(w *World, held *MotionInput) fourthFinalNativePlan {
	q := fourthFinalNativePlan{valid: true, shield: w.Equipment.Shield}
	if p.forecast.Load(w) != nil {
		return fourthFinalNativePlan{}
	}
	for pass := 0; pass < 6; pass++ {
		s := p.forecast.State()
		x, y := fourthFinalNativeTarget(s)
		m, _ := fourthMiddleCoastMotion(s.Player, x)
		m.Up = s.Player.Y > y
		m.Down = s.Player.Y < y
		if held != nil {
			m = *held
		}
		in := Input{Motion: m, Fire: true}
		if pass == 0 {
			q.first = in
		}
		for range 3 {
			p.forecast.AdvancePALTick()
		}
		r, e := p.forecast.Advance(in)
		if e != nil {
			return fourthFinalNativePlan{}
		}
		s = p.forecast.State()
		q.shield = min(q.shield, r.Shield)
		if fourthAdmissionTerrainUnsafe(s) || fourthFinalNativeBodyUnsafe(s) {
			q.valid = false
			return q
		}
		if r.Boundary != ForecastRunning || s.FourthFinal.Defeated {
			break
		}
	}
	s := p.forecast.State()
	q.alive = s.PlayerAlive
	for i := 0; i < 3; i++ {
		v := int(int16(s.FourthFinal.Parts[i].Health))
		if v > 0 {
			q.hp += v
		}
	}
	x, y := fourthFinalNativeTarget(s)
	q.distance = absDemo(s.Player.X-x) + absDemo(s.Player.Y-y)
	return q
}
func (p *fourthFinalNativePilot) Input(w *World) Input {
	key := fourthFinalNativeState(w)
	if p.world == w && p.key == key {
		return p.last
	}
	best := p.evaluate(w, nil)
	found := best.valid
	for _, m := range demoDirections {
		q := p.evaluate(w, &m)
		if !q.valid {
			continue
		}
		if !found || q.alive && !best.alive || q.alive == best.alive && (q.shield > best.shield || q.shield == best.shield && (q.hp < best.hp || q.hp == best.hp && q.distance < best.distance)) {
			best = q
			found = true
		}
	}
	if !found {
		x, y := fourthFinalNativeTarget(w)
		m, _ := fourthMiddleCoastMotion(w.Player, x)
		m.Up = w.Player.Y > y
		m.Down = w.Player.Y < y
		best.first = Input{Motion: m, Fire: true}
	}
	p.world, p.key, p.last = w, key, best.first
	return best.first
}

// The final specialist is limited to the earned ordinary equipment profile.
func fourthFinalNativeEligible(w *World, pal int) bool {
	if w == nil || w.Level.Number != 4 || w.FourthFinal == nil || w.FourthFinal.Defeated || !w.PlayerAlive || w.GameOver || w.Ready || w.blockedFireUntilRelease || w.Dive.Phase != 0 || w.ScreenClearFrames != 0 || w.stepContinuation.active || w.ShopReady || w.ExitReady || w.LevelFinished || w.PendingExitDrops != 0 || w.Level.Ships == nil || w.Level.PlayerStencil == nil || w.Level.Paths == nil || w.Coverage == nil || w.Weapons == nil || thirdMiddlePALRefreshes(pal) != 3 {
		return false
	}
	e := w.Equipment
	if e.Primary.Item != ItemForwardShot || e.Primary.Tier != 1 || e.Mounts[0].Item != ItemCannon || e.Mounts[0].Tier != 0 || e.Rear.Item != ItemRearShot || e.Rear.Tier != 1 || e.Side.Item != ItemNone || e.SpeedTier != 2 || e.FirePeriod != 8 || e.FireAdvance != 3 || e.SuperFrames != 0 || e.SuperLoadoutActive {
		return false
	}
	for _, slot := range e.Mounts[1:] {
		if slot.Item != ItemNone {
			return false
		}
	}
	art := w.fourthFinalArt
	if art == nil || len(art.Components) != 19 || len(art.MotionTables["body_vertical_offsets"]) != 16 || len(art.MotionTables["left_eye_headings"]) != 8 || len(art.MotionTables["right_eye_headings"]) != 8 || len(art.MotionTables["eye_shot_offset_x"]) != 8 || len(art.MotionTables["eye_shot_offset_y"]) != 8 {
		return false
	}
	for index := 0; index < 19; index++ {
		if w.fourthFinalActors[index] == nil {
			return false
		}
		if index >= 3 {
			box, ok := w.movingSpriteBoxes[art.Components[index].Sprite]
			if !ok || box.Width <= 0 || box.Height <= 0 {
				return false
			}
		}
	}
	return true
}

func (p *PresentationPilot) fourthFinalNativeInput(w *World) (Input, bool) {
	if !fourthFinalNativeEligible(w, p.PALRefreshes) {
		p.fourthFinalNative = nil
		return Input{}, false
	}
	if p.fourthFinalNative == nil {
		p.fourthFinalNative = &fourthFinalNativePilot{}
	}
	return p.fourthFinalNative.Input(w), true
}
