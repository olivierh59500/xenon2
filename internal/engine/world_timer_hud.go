package engine

// TimerMeterFrame uses the source ten-image meter after the countdown decrement.
// The extra denominator step keeps a newly started meter below image ten.
func TimerMeterFrame(remaining, maximum int) int {
	return int(uint16(remaining)) * 10 / (maximum + 1)
}

func (w *World) appendTimerMeter(remaining, maximum, x, y int) {
	if w.Level.Common == nil {
		return
	}
	frame := TimerMeterFrame(remaining, maximum)
	if frame >= len(w.Level.Common.TimerFrames) {
		return
	}
	w.HUD = append(w.HUD, WorldSpriteAttachment{Atlas: "common", Sprite: w.Level.Common.TimerFrames[frame], X: float64(x), Y: float64(y)})
}

func (w *World) advanceTimedEquipment() {
	superWasActive := w.Equipment.SuperFrames != 0
	suiteWasInstalled := w.Equipment.SuperLoadoutActive
	diveWasActive := w.Dive.Remaining != 0
	w.Equipment.AdvanceTimers()
	if suiteWasInstalled && !w.Equipment.SuperLoadoutActive && w.Weapons != nil {
		if err := w.Weapons.RestoreSuperEquipment(w.weaponContext(Input{}, false)); err != nil {
			w.poolError = err
		}
	}
	w.Dive.Tick()
	if diveWasActive {
		w.appendTimerMeter(w.Dive.Remaining, 136, 152, 8)
	}
	if superWasActive {
		w.appendTimerMeter(w.Equipment.SuperFrames, 170, 152, 160)
	}
}
