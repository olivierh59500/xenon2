package engine

import "testing"

func TestStagePlayerPosesMatchOriginalAdmissionOptional(t *testing.T) {
	sessions := make(map[[3]int]*Session)
	rows := 0
	nativeCombatRows(t, "stage-pool-poses.csv", func(v []int64) {
		if len(v) != 9 {
			t.Fatal("invalid original stage-pose row")
		}
		key := [3]int{int(v[0]), int(v[1]), int(v[2])}
		s := sessions[key]
		if s == nil {
			var err error
			s, err = NewSession(originalWorldData(t, key[0]), key[1], NewRandomState())
			if err != nil {
				t.Fatal(err)
			}
			for player := 0; player < s.PlayerCount; player++ {
				w := s.ActiveWorld()
				w.Player.X, w.Player.Y = key[2]+player*16, 80+player*16
				w.LevelFinished = true
				if _, err := s.CompleteStage(originalWorldData(t, int(v[4]))); err != nil {
					t.Fatal(err)
				}
			}
			sessions[key] = s
		}
		w := s.Players[int(v[3])]
		if w.Level.Number != int(v[4]) || w.Checkpoint.PlayerX != int(v[5]) || w.Checkpoint.WorldY != int(v[6]) || w.Player.X != int(v[7]) || w.Player.Y != int(v[8]) {
			t.Errorf("stage pose%v: Go checkpoint%+v ship%+v", v, w.Checkpoint, w.Player)
		}
		if int(v[3]) == s.Current {
			for _, trail := range w.shipTrail {
				if trail.X != w.Player.X || trail.Y != w.Player.Y {
					t.Fatal("READY retained an outgoing thrust-history position")
				}
			}
		}
		rows++
	})
	if rows != 45 || len(sessions) != 30 {
		t.Fatal("incomplete original pose coverage across levels and players")
	}
	t.Logf("Compared %d original retained positions, checkpoint coordinates and READY poses", rows)
}
