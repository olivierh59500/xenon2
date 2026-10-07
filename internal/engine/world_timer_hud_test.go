package engine

import "testing"

func TestTimerMeterNativeImageSelectionOptional(t *testing.T) {
	common := originalWorldData(t, 1).Common
	rows := 0
	nativeCombatRows(t, "timer-meter-trace.csv", func(v []int64) {
		if frame := TimerMeterFrame(int(v[1]), int(v[0])); frame != int(v[2]) {
			t.Fatalf("source meter frame differs: %v got%d", v, frame)
		}
		w := testWorld(t)
		w.Level.Common = common
		w.appendTimerMeter(int(v[1]), int(v[0]), int(v[3]), int(v[4]))
		if len(w.HUD) != 1 || w.HUD[0].Sprite != w.Level.Common.TimerFrames[v[2]] || w.HUD[0].X != float64(v[3]) || w.HUD[0].Y != float64(v[4]) {
			t.Fatal("meter snapshot lost its original image or anchor")
		}
		rows++
	})
	if rows != 306 {
		t.Fatalf("incomplete meter comparison: %d", rows)
	}
}

func TestTimedEquipmentKeepsZeroMeterOnExpiryOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	w.Equipment.SuperFrames, w.Dive.Remaining = 1, 1
	w.advanceTimedEquipment()
	if w.Equipment.SuperFrames != 0 || w.Dive.Remaining != 0 || len(w.HUD) != 2 || w.HUD[0].Sprite != w.Level.Common.TimerFrames[0] || w.HUD[1].Sprite != w.Level.Common.TimerFrames[0] {
		t.Fatal("expiry omitted either source zero-meter frame")
	}
	w.HUD = w.HUD[:0]
	w.advanceTimedEquipment()
	if len(w.HUD) != 0 {
		t.Fatal("expired effects kept drawing meters")
	}
}
