package engine

import "testing"

func TestSecondGuardCoversLateCorridorAndBothFinalFlanksOptional(t *testing.T) {
	for _, sample := range []struct {
		name         string
		camera, left int
		unsafe       bool
	}{{"late corridor", 845, 40, true}, {"left final flank", 200, 40, false}, {"right final flank", 200, 220, true}} {
		t.Run(sample.name, func(t *testing.T) {
			w := firstLevelGuardOriginalScene(t, 2)
			w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = sample.camera, sample.camera+16, sample.camera+16
			found := false
			for y := 100; y <= 140 && !found; y += 4 {
				for x := sample.left; x < sample.left+40; x += 4 {
					if !w.Coverage.Touches(x, y, w.ScrollY, *w.Level.PlayerStencil) {
						w.Player.X, w.Player.Y, found = x, y, true
						break
					}
				}
			}
			if !found {
				t.Fatal("original map has no clear ship pose in the requested flank")
			}
			w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
			w.PreviousPlayer = w.Player
			w.updatePlayerCollision()
			w.secondMiddleReleased = true
			w.SecondGuardian.WaitTimer = 100
			w.spawnEnemyShot(w.Player.X, w.Player.Y-30, EnemyShot{Direction: 4, Speed: 6})
			before := forecastIsolationDigest(w)
			pilot := PresentationPilot{}
			planned := Input{}
			guarded := pilot.forecastSecondGuard(w, planned)
			held, _ := sixThirdFinalGuardPasses(t, w, planned)
			safe, _ := sixThirdFinalGuardPasses(t, w, guarded)
			if !sample.unsafe {
				if held.Shield != 39 || guarded != planned || forecastIsolationDigest(w) != before {
					t.Fatal("a harmless original projectile changed safe flank movement")
				}
				return
			}
			if guarded.Motion == planned.Motion || guarded.Fire != planned.Fire || guarded.Dive != planned.Dive || forecastIsolationDigest(w) != before {
				t.Fatalf("late second-stage guard: pose%+v planned%+v guarded%+v held%+v safe%+v", w.Player, planned, guarded, held, safe)
			}
			if held.Shield >= 39 || !safe.Alive || safe.Shield != 39 {
				t.Fatalf("original projectile: held%+v guarded%+v", held, safe)
			}
		})
	}
}
