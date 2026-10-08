package engine

import "testing"

func TestFourthCoreRenewsPredictedReverseBoundBeforeScroll(t *testing.T) {
	for _, mode := range []string{"active", "disabled", "absent"} {
		for _, motion := range []MotionInput{{Down: true}, {}} {
			t.Run(mode+map[bool]string{true: "/down", false: "/idle"}[motion.Down], func(t *testing.T) {
				w, err := NewWorld(playableOriginalWorldData(t, 4))
				if err != nil {
					t.Fatal(err)
				}
				if mode != "absent" {
					w.ScrollY = 2480
					if err := w.activateFourthGuardian(false); err != nil {
						t.Fatal(err)
					}
				}
				w.Ready, w.MaterializationFrames = false, 0
				w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 2191, 2176, 2192, 2192
				w.Player.Y, w.Player.Inertia, w.Player.ScrollStep = 176, 0, 1
				found := false
				for x := 14; x <= 304; x++ {
					clear := true
					for camera := 2190; camera <= 2192; camera++ {
						clear = clear && !w.Coverage.Touches(x, 176, camera, *w.Level.PlayerStencil)
					}
					if clear {
						w.Player.X, found = x, true
						break
					}
				}
				if !found {
					t.Fatal("original arena has no clear isolated scroll-control position")
				}
				w.Rewind = NewTerrainRewind(2191, w.Player.X, 176)
				w.ScrollDelta = 0
				w.cursor = EncounterCursor{MovingHighWater: 0, FixedHighWater: 0}
				if mode == "disabled" {
					w.FourthMiddle.Parts[4].Disabled = true
				}
				prediction := newDemoMotionForecast(w)
				before := forecastIsolationDigest(w)
				if !prediction.advance(w, motion) {
					t.Fatal("source clear-scroll prediction rejected")
				}
				if before != forecastIsolationDigest(w) {
					t.Fatal("prediction changed the original core, camera or random stream")
				}
				if err := w.Step(Input{Motion: motion}); err != nil {
					t.Fatal(err)
				}
				if !nativeMotionMatches(w, prediction) {
					t.Fatalf("source core late maximum differs: forecast%+v actualcamera%d maximum%d player%+v", prediction.scroll, w.ScrollY, w.MaximumScrollY, w.Player)
				}
				if mode == "active" && w.MaximumScrollY <= 2192 {
					t.Fatal("active original core did not renew its late reverse limit")
				}
				if mode != "active" && w.MaximumScrollY != 2192 {
					t.Fatal("disabled or absent core changed the existing scroll scope")
				}
			})
		}
	}
}
