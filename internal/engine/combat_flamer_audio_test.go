package engine

import "testing"

func TestCombatFlamerSoundNativeTraceOptional(t *testing.T) {
	var state FlamerSoundState
	nativeCombatRows(t, "combat-flamer-sound-trace.csv", func(v []int64) {
		start, stop := state.Advance(v[1] != 0, v[2] != 0, v[3] != 0)
		if state.Started != (v[4] != 0) || start != (v[5] != 0) || stop != (v[6] != 0) {
			t.Fatalf("flamer sound %v: started=%v request=%v stop=%v", v, state.Started, start, stop)
		}
	})
}
