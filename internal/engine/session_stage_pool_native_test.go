package engine

import "testing"

// The private reference arranges sparse completion boundaries, then runs the
// original cleanup, weapon restoration and level initialization. It is not a
// full original playthrough; unused initial RAM contents are not inferred.
func TestStagePoolRetentionMatchesOriginalTransitionsOptional(t *testing.T) {
	sessions := make(map[[2]int]*Session)
	rows := 0
	nativeCombatRows(t, "stage-pool-retention.csv", func(v []int64) {
		if len(v) != 12 {
			t.Fatal("invalid original stage-storage row")
		}
		key := [2]int{int(v[0]), int(v[1])}
		s := sessions[key]
		if s == nil {
			var err error
			s, err = NewSession(originalWorldData(t, key[0]), key[1], NewRandomState())
			if err != nil {
				t.Fatal(err)
			}
			storage := s.ActiveWorld().Pool.storage()
			storage.slots[100].Residue.XFraction = 0x1234
			storage.slots[100].Residue.Counter = 0x456
			for player := 0; player < s.PlayerCount; player++ {
				s.ActiveWorld().LevelFinished = true
				route, err := s.CompleteStage(originalWorldData(t, int(v[3])))
				if err != nil || (route == LoadedNextStage) != (player == s.PlayerCount-1) {
					t.Fatalf("completion route%d error%v", route, err)
				}
			}
			if s.ActiveWorld().Pool.storage() != storage {
				t.Errorf("level%d players%d replaced the original persistent physical reserve", key[0], key[1])
			}
			sessions[key] = s
		}
		w := s.Players[int(v[2])]
		r := w.Pool.Slot(100).Residue
		got := [6]int{w.Pool.FreeFirst(), w.Pool.First(ActorPoolMoving), w.Pool.First(ActorPoolEquipment), w.Pool.First(ActorPoolScenery), int(r.XFraction), int(r.Counter)}
		want := [6]int{int(v[5]), int(v[6]), int(v[7]), int(v[8]), int(v[10]), int(v[11])}
		if w.Level.Number != int(v[3]) || s.Difficulty != int(v[4]) || got != want {
			t.Errorf("level%d players%d view%d: Go level%d difficulty%d storage%v original%v", key[0], key[1], v[2], w.Level.Number, s.Difficulty, got, want)
		}
		// The inactive source game's equipment pointers are not live until its
		// READY restoration. Check the installed primary of the admitted player.
		if int(v[2]) == s.Current && w.Weapons.mounts[0].Binding.Slot != int(v[9]) {
			t.Errorf("level%d players%d active primary: Go slot%d original%d", key[0], key[1], w.Weapons.mounts[0].Binding.Slot, v[9])
		}
		rows++
	})
	if rows != 15 || len(sessions) != 10 {
		t.Fatal("incomplete original one/two-player stage-transition coverage")
	}
	t.Logf("Compared fifteen original stage boundaries, shared storage identity, free/list heads and retained named fields")
}
