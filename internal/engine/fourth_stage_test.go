package engine

import "testing"

func TestFourthStageSourcePreludeOptional(t *testing.T) {
	art, _ := fourthStageTestArt(t)
	cases := 0
	nativeCombatRows(t, "fourth-stage-trace.csv", func(v []int64) {
		decision := FourthStagePrelude(uint64(v[1]), int(v[2]), int(v[3]))
		random := RandomState{A: uint32(v[4]), B: uint32(v[5])}
		if decision.MaximumScrollY != int(v[6]) || decision.Spawn != (v[7] != 0) {
			t.Fatalf("stage prelude %v: %+v", v, decision)
		}
		if decision.Spawn {
			variant := decision.FamilyOffset + int(random.Next()&1)
			state := NewFourthFallingActor(variant, art, ActorResidue{}, random.Next())
			if art.Variants[variant].ResourceTag != int(v[8]) || state.X != int(v[9]) || state.Y != int(v[10]) || state.PrimaryClock != uint8(v[11]) {
				t.Fatalf("stage birth %v: %+v", v, state)
			}
		}
		if random.A != uint32(v[12]) || random.B != uint32(v[13]) {
			t.Fatalf("stage random %v: %+v", v, random)
		}
		cases++
	})
	if cases != 147 {
		t.Fatalf("stage prelude coverage:%d", cases)
	}
}
