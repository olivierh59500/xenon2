package engine

import "testing"

func TestCombatCannonAndLauncherNativeTraceOptional(t *testing.T) {
	var cannon CannonMountState
	var launcher MissileMountState
	nativeCombatRows(t, "combat-cannon-mount-trace.csv", func(v []int64) {
		if v[1] == 0 {
			cannon, launcher = CannonMountState{}, MissileMountState{}
		}
		phase, pending, emissions := 0, false, 0
		if Item(v[0]) == ItemCannon {
			if cannon.Tick(v[2] != 0, v[3] != 0) {
				emissions = 1
			}
			phase = cannon.Phase
		} else {
			if launcher.Tick(v[2] != 0, v[3] != 0) {
				emissions = 2
			}
			phase, pending = launcher.Phase, launcher.PendingStart
		}
		if phase != int(v[4]) || pending != (v[5] != 0) || emissions != int(v[6]) {
			t.Fatalf("mount %v: cannon=%+v launcher=%+v emissions=%d", v, cannon, launcher, emissions)
		}
		if emissions == 1 && (v[7] != 134 || v[8] != 181) {
			t.Fatalf("cannonball launch position differs: %v", v)
		}
		if emissions == 2 {
			shots := AppendLauncherMissiles(nil, 134, 192, "launcher-missile")
			if int64(shots[1].X) != v[7] || int64(shots[1].Y) != v[8] || int64(shots[0].X) != v[9] || int64(shots[0].Y) != v[10] {
				t.Fatalf("launcher positions %v: got %+v", v, shots)
			}
		}
	})
}
