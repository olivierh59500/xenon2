package engine

// The earned equipment profile and native tail gate bound this specialist.
// It supplies ordinary input; source callbacks retain all health and hit rules.
func fourthMiddleSpecialistEligible(w *World, pal int) bool {
	if w == nil || w.Level.Number != 4 || w.FourthMiddle == nil || w.FourthMiddle.Defeated || !w.FourthMiddle.Parts[15].Disabled || !w.PlayerAlive || w.GameOver || w.Ready || w.ScreenClearFrames != 0 || w.stepContinuation.active || w.Dive.Phase != 0 || w.ShopReady || w.ExitReady || w.LevelFinished || w.PendingExitDrops != 0 || w.Coverage == nil || w.Level.PlayerStencil == nil || w.Level.Ships == nil || w.Level.Paths == nil || w.Weapons == nil || thirdMiddlePALRefreshes(pal) != 3 {
		return false
	}
	e := w.Equipment
	if e.Primary.Item != ItemForwardShot || e.Primary.Tier != 1 || e.Mounts[0].Item != ItemCannon || e.Mounts[0].Tier != 0 || e.Rear.Item != ItemRearShot || e.Rear.Tier != 0 || e.Side.Item != ItemNone || e.SpeedTier != 2 || e.FirePeriod != 8 || e.FireAdvance != 3 || e.SuperLoadoutActive || e.SuperFrames != 0 {
		return false
	}
	for _, slot := range e.Mounts[1:] {
		if slot.Item != ItemNone {
			return false
		}
	}
	if w.fourthMiddleArt == nil || len(w.fourthMiddleArt.Components) != 20 || w.fourthMiddleActors[18] == nil || w.fourthMiddleActors[16] == nil {
		return false
	}
	for _, index := range []int{17, 19} {
		if _, _, _, ok := fourthMiddleStableAim(w, index); !ok {
			return false
		}
	}
	animation, ok := w.Weapons.animations["cannon-ball"]
	if !ok || len(animation.Animation.Frames) == 0 {
		return false
	}
	box, ok := w.Weapons.boxes[animation.Animation.Frames[0].Sprite]
	return ok && box.Width > 0 && box.Height > 0
}

func (p *PresentationPilot) fourthMiddleSpecialistInput(w *World) (Input, bool) {
	if !fourthMiddleSpecialistEligible(w, p.PALRefreshes) {
		p.fourthMiddleUpper = nil
		p.fourthMiddleRight = nil
		p.fourthMiddleLeft = nil
		return Input{}, false
	}
	m := w.FourthMiddle
	if !m.Parts[17].Disabled || !m.Parts[19].Disabled {
		if p.fourthMiddleUpper == nil {
			p.fourthMiddleUpper = &fourthMiddleUpperPilot{}
		}
		return p.fourthMiddleUpper.Input(w), true
	}
	if !m.Parts[18].Disabled {
		if p.fourthMiddleRight == nil {
			p.fourthMiddleRight = &fourthMiddleRightPilot{}
		}
		return p.fourthMiddleRight.Input(w), true
	}
	if !m.Parts[16].Disabled {
		if p.fourthMiddleLeft == nil {
			p.fourthMiddleLeft = &fourthMiddleLeftPilot{}
		}
		return p.fourthMiddleLeft.Input(w), true
	}
	return Input{}, false
}
