package engine

import "testing"

func TestCombatWeaponExplosionNativeTraceOptional(t *testing.T) {
	var random RandomState
	var effects [4]WeaponExplosion
	var actual []WeaponExplosion
	nativeCombatRows(t, "combat-weapon-explosion-trace.csv", func(v []int64) {
		if v[2] == 0 {
			random = RandomState{A: uint32(0x630c1592 + v[1]*7919), B: uint32(0x35358979 - v[1]*173)}
			var err error
			actual, err = AppendWeaponExplosions(effects[:0], []string{"mine-small", "mine-large", "bomb"}[v[0]], 160+int(v[1]), 100-int(v[1]), random.Next)
			if err != nil {
				t.Fatal(err)
			}
		}
		effect := actual[v[2]]
		if effect.X != int(v[3]) || effect.Y != int(v[4]) || random.A != uint32(v[5]) || random.B != uint32(v[6]) {
			t.Fatalf("explosion %v: got %+v random=%+v", v, effect, random)
		}
	})
}

func TestMineReturnsToCurrentShip(t *testing.T) {
	mine := MineState{X: 30, Y: 50, Mode: 3}
	for range 30 {
		mine.Advance(160, 100, 40, 60, MotionInput{}, false, false, false, nil)
	}
	if mine.X != 160 || mine.Y != 125 || mine.Mode != 0 {
		t.Fatalf("mine did not return to ship: %+v", mine)
	}
}
