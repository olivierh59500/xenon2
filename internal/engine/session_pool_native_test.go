package engine

import "testing"

// These references run the complete original game/level initializers, including
// both saved player games, before observing their one shared physical pool.
func TestSessionPoolsMatchOriginalInitializersOptional(t *testing.T) {
	sessions := make(map[[2]int]*Session)
	rows := 0
	nativeCombatRows(t, "session-pool.csv", func(v []int64) {
		if len(v) != 14 {
			t.Fatal("invalid original session pool row")
		}
		key := [2]int{int(v[0]), int(v[1])}
		s := sessions[key]
		if s == nil {
			var err error
			s, err = NewSession(originalWorldData(t, key[0]), key[1], NewRandomState())
			if err != nil {
				t.Fatal(err)
			}
			sessions[key] = s
		}
		w := s.Players[int(v[2])]
		got := [10]int{w.Pool.FreeFirst(), w.Pool.First(ActorPoolMoving), w.Pool.First(ActorPoolEquipment), w.Pool.First(ActorPoolProjectile), w.Pool.First(ActorPoolScenery)}
		want := [10]int{int(v[3]), int(v[5]), int(v[6]), int(v[7]), int(v[8])}
		// The original player list also includes the dedicated ship, which does
		// not occupy a pooled slot. Its four pooled shadows are observed backward.
		for i, shadow := range w.poolShadows {
			got[5+i], want[5+i] = shadow.Slot, int(v[12-i])
		}
		got[9], want[9] = w.Weapons.mounts[0].Binding.Slot, int(v[13])
		if got != want {
			t.Errorf("level%d players%d current%d pool heads/owners: Go%v original%v", key[0], key[1], v[2], got, want)
		}
		rows++
	})
	if rows != 15 || len(sessions) != 10 {
		t.Fatal("incomplete one/two-player original initialization coverage")
	}
	physical := 0
	failed := make(map[[2]int]bool)
	nativeCombatRows(t, "session-pool-slots.csv", func(v []int64) {
		if len(v) != 4 {
			t.Fatal("invalid original physical slot row")
		}
		key := [2]int{int(v[0]), int(v[1])}
		s := sessions[key]
		if s == nil {
			t.Fatal("physical reference has no matching session")
		}
		for player := 0; player < s.PlayerCount; player++ {
			slot := s.Players[player].Pool.Slot(int(v[2]))
			if slot == nil {
				t.Fatal("original physical slot is outside the pool")
			}
			if int64(slot.ResourceTag) != v[3] && !failed[key] {
				t.Errorf("level%d players%d view%d slot%d: Go tag%d original%d", key[0], key[1], player, v[2], slot.ResourceTag, v[3])
				failed[key] = true
			}
			physical++
		}
	})
	if physical != 2385 {
		t.Fatalf("incomplete shared physical-state coverage: %d", physical)
	}
	t.Logf("Compared fifteen original player admissions and %d physical slot tags across all five levels and both player counts", physical)
}
