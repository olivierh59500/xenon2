package engine

import "testing"

// TestCombatPickupSelectorsNativeTraceOptional compares every original reward
// selector with the world's named equipment and temporary-effect handling.
func TestCombatPickupSelectorsNativeTraceOptional(t *testing.T) {
	nativeCombatRows(t, "combat-pickup-trace.csv", func(v []int64) {
		world := &World{Equipment: NewEquipment(), random: NewRandomState()}
		world.applyCarrierReward(int(v[0]))
		if v[0] == 18 {
			if world.ScreenClearFrames != 31 {
				t.Fatal("screen clear must pause for thirty-one PAL ticks")
			}
			for range 31 {
				world.AdvancePALTick()
			}
			if world.ScreenClearFrames != 0 || world.ScreenClearPaletteMask != 0 {
				t.Fatal("screen-clear palette must restore after the original duration")
			}
		}
		for index, slot := range world.Equipment.slots() {
			at := 1 + index*3
			if int64(slot.Item) != v[at] || int64(slot.Tier) != v[at+1] || int64(slot.MaxTier) != v[at+2] {
				t.Fatalf("reward %d slot %d: got %+v, expected %v", v[0], index, slot, v)
			}
		}
		e := world.Equipment
		if int64(e.Shield) != v[22] || int64(e.SpeedTier) != v[23] || int64(e.FireAdvance) != v[24] || int64(e.DiveCharges) != v[25] || int64(e.SuperFrames) != v[26] || int64(world.InvulnerableFrames) != v[27] || int64(e.ShadesFrames) != v[28] || e.Protection != (v[29] != 0) {
			t.Fatalf("reward %d: got %+v invulnerability=%d, expected %v", v[0], e, world.InvulnerableFrames, v)
		}
	})
}
