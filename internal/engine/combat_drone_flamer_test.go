package engine

import "testing"

func TestCombatDroneNativeTraceOptional(t *testing.T) {
	var drone DroneMountState
	nativeCombatRows(t, "combat-drone-trace.csv", func(v []int64) {
		if v[1] == 0 {
			drone = DroneMountState{}
		}
		count := 0
		if drone.Tick(v[2] != 0, v[3] != 0) {
			count = len(dronePatterns[v[0]])
		}
		if drone.Cooldown != int(v[4]) || count != int(v[7]) {
			t.Fatalf("drone %v: got %+v emissions=%d", v, drone, count)
		}
	})
	var shots []SparkShot
	nativeCombatRows(t, "combat-drone-shots-trace.csv", func(v []int64) {
		if v[1] == 0 {
			var err error
			shots, err = AppendDroneSparks(shots[:0], int(v[0]), 160, 125)
			if err != nil {
				t.Fatal(err)
			}
		}
		shot := shots[v[1]]
		if shot.X != int(v[2]) || shot.Y != int(v[3]) || shot.VelocityX != int(v[4]) || shot.VelocityY != int(v[5]) {
			t.Fatalf("drone spark %v: got %+v", v, shot)
		}
	})
}

func TestCombatFlamerNativeTraceOptional(t *testing.T) {
	var random RandomState
	states := map[int][]FlameShot{}
	residues := map[int][]uint16{}
	expected := map[int][]int64{}
	nativeCombatRows(t, "combat-flamer-shots-trace.csv", func(v []int64) {
		frame := int(v[0])
		residues[frame] = append(residues[frame], uint16(v[4]))
		expected[frame*2+int(v[1])] = append([]int64(nil), v...)
	})
	nativeCombatRows(t, "combat-flamer-trace.csv", func(v []int64) {
		frame := int(v[0])
		if frame == 0 {
			random = RandomState{A: 0x12345678, B: 0x6abcdef1}
		}
		var shots []FlameShot
		if v[1] != 0 && v[2] == 0 {
			used := 0
			residue := func() uint16 {
				value := residues[frame][1-used]
				used++
				return value
			}
			var err error
			shots, err = AppendFlamerShotsWithResidue(nil, 2, 160, 176, random.Next, residue)
			if err != nil {
				t.Fatal(err)
			}
		}
		states[frame] = shots
		if len(shots) != int(v[3]) || random.A != uint32(v[4]) || random.B != uint32(v[5]) {
			t.Fatalf("flamer frame %v: count=%d random=%+v", v, len(shots), random)
		}
	})
	for key, v := range expected {
		shot := states[key/2][1-key%2]
		if uint32(shot.X) != uint32(v[2]) || shot.Y != int(v[3]) || shot.VelocityX != int32(v[4])>>6 || shot.Tier != int(v[5]) || shot.Damaging != (v[6] != 0) {
			t.Fatalf("flamer particle %v: got %+v", v, shot)
		}
	}
}
