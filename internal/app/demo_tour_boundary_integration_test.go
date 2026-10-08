package app

import "testing"

// These arranged merchant-boundary fixtures isolate the tour limit. The full
// frontend regression independently earns the third guardian and exit rewards.
func TestDemoTourLimitAppliesOnlyAfterItsCompletedFinalMerchant(t *testing.T) {
	for _, test := range []struct {
		name                   string
		level                  int
		final, demo, completed bool
		limit                  int
		menu                   bool
	}{
		{"first final merchant", 1, true, true, true, 3, false},
		{"second final merchant", 2, true, true, true, 3, false},
		{"third middle merchant", 3, false, true, false, 3, false},
		{"third final merchant", 3, true, true, true, 3, true},
		{"unfinished third boundary", 3, true, true, false, 3, false},
		{"manual third completion", 3, true, false, true, 3, false},
		{"later-stage reference fixture", 3, true, true, true, 5, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := menuAdmittedGame(t, test.level, 1)
			w := g.Driver.(*worldDriver).world
			g.Config.Demo, g.Config.HumanDemo = test.demo, test.demo
			g.demoLastLevel = test.limit
			g.shopFinal = test.final
			w.LevelFinished, w.ExitReady = test.completed, test.completed
			w.PendingExitDrops = 0
			if err := g.leaveShop(); err != nil {
				t.Fatal(err)
			}
			if test.menu {
				if g.DemoActive() || g.demo != nil || g.headerAction != headerNone || g.Screen != PresentationScreen {
					t.Fatal("completed three-level tour did not return through the original menu animation")
				}
				awaitFrontendBoundary(t, g, 180, "completed tour menu", func() bool { return g.Screen == TitleScreen })
				if g.Driver.(*worldDriver).world != w || w.Level.Number != 3 || !w.LevelFinished || w.PendingExitDrops != 0 {
					t.Fatal("menu return changed the completed world or admitted another level")
				}
			} else {
				want := headerNextStage
				if !test.final {
					want = headerReload
				}
				if g.Config.Demo != test.demo || g.headerAction != want {
					t.Fatal("tour limit intercepted a different gameplay boundary")
				}
			}
		})
	}
}
