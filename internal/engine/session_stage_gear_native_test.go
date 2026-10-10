package engine

import "testing"

func TestStagePoolEquipmentMatchesOriginalTransitionsOptional(t *testing.T) {
	profiles := [][]Item{nil, {ItemCannon, ItemPowerup}, {ItemDoubleShot, ItemRearShot, ItemPowerup}, {ItemLaser, ItemSideShot, ItemProtection}, {ItemCannon, ItemLaser, ItemPowerup, ItemSuperNashwan}}
	sessions := make(map[[3]int]*Session)
	rows := 0
	nativeCombatRows(t, "stage-pool-gear.csv", func(v []int64) {
		if len(v) != 13 || v[0] < 0 || int(v[0]) >= len(profiles) {
			t.Fatal("invalid original equipment-transition row")
		}
		key := [3]int{int(v[0]), int(v[1]), int(v[2])}
		s := sessions[key]
		if s == nil {
			var err error
			s, err = NewSession(originalWorldData(t, key[1]), key[2], NewRandomState())
			if err != nil {
				t.Fatal(err)
			}
			s.ActiveWorld().Pool.Slot(100).Residue.XFraction = 0x1234
			s.ActiveWorld().Pool.Slot(100).Residue.Counter = 0x456
			for player := 0; player < s.PlayerCount; player++ {
				w := s.ActiveWorld()
				for _, item := range profiles[key[0]] {
					if item == ItemSuperNashwan {
						continue
					}
					w.Equipment.ApplyItem(item)
					if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
						t.Fatal(err)
					}
				}
				w.Checkpoint.Loadout = w.Equipment.WeaponLoadout
				for _, item := range profiles[key[0]] {
					if item != ItemSuperNashwan {
						continue
					}
					w.Equipment.ApplyItem(item)
					w.Equipment.BeginSuperLoadout()
					if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
						t.Fatal(err)
					}
				}
				w.LevelFinished = true
				if _, err := s.CompleteStage(originalWorldData(t, int(v[4]))); err != nil {
					t.Fatal(err)
				}
			}
			sessions[key] = s
		}
		w := s.Players[int(v[3])]
		r := w.Pool.Slot(100).Residue
		got := [6]int{w.Pool.FreeFirst(), w.Pool.First(ActorPoolMoving), w.Pool.First(ActorPoolEquipment), w.Pool.First(ActorPoolScenery), int(r.XFraction), int(r.Counter)}
		want := [6]int{int(v[6]), int(v[7]), int(v[8]), int(v[9]), int(v[11]), int(v[12])}
		if w.Level.Number != int(v[4]) || s.Difficulty != int(v[5]) || got != want {
			t.Errorf("equipment profile%v view%d: Go%v original%v", key, v[3], got, want)
		}
		if int(v[3]) == s.Current && w.Weapons.mounts[0].Binding.Slot != int(v[10]) {
			t.Errorf("equipment profile%v active primary: Go%d original%d", key, w.Weapons.mounts[0].Binding.Slot, v[10])
		}
		rows++
	})
	if rows != 75 || len(sessions) != 50 {
		t.Fatal("incomplete original stage/equipment coverage")
	}
	t.Logf("Compared %d original transitions with basic, cannon, rear, side, laser and temporary-suite equipment", rows)
}
