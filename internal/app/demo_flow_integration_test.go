package app

import (
	"os"
	"strconv"
	"testing"
	"xenon2/internal/engine"
	"xenon2/internal/shopui"
)

func TestDemoAdmitsNormalSessionAndManualInputTakesOver(t *testing.T) {
	g := frontendGame(t)
	g.Config.Demo = true
	for update := 0; update < 3600; update++ {
		advanceFrontend(t, g, inputFrame{})
		if g.Screen == LevelScreen && !g.View.Ready && !g.backdropOnly && g.View.PlayerAlive && g.Driver.(*worldDriver).world.Frame > 20 {
			break
		}
	}
	d, ok := g.Driver.(*worldDriver)
	if !ok || d.session == nil || d.diagnostic || g.Screen != LevelScreen || d.world.Frame <= 20 || !g.DemoActive() {
		t.Fatal("demo bypassed or failed normal menu/READY admission")
	}
	frame := d.world.Frame
	advanceFrontend(t, g, inputFrame{anyKey: true, gameMotion: engine.MotionInput{Right: true}})
	if g.DemoActive() || g.demo != nil || d.world.Frame < frame {
		t.Fatal("human input did not immediately recover the active session")
	}
}

func TestDemoShopUsesRealQuotesPurchasesAndWallet(t *testing.T) {
	g := menuAdmittedGame(t, 1, 1)
	w := g.Driver.(*worldDriver).world
	// This is an explicit merchant boundary fixture, not a demo victory.
	w.Money = 3000
	w.Equipment.Shield = 7
	if err := g.EnterShop(false); err != nil {
		t.Fatal(err)
	}
	g.Config.Demo = true
	quoteSeen := false
	original := w.Money
	for update := 0; update < 12000; update++ {
		advanceFrontend(t, g, inputFrame{})
		if g.shop != nil && g.shop.QuoteValid && g.shop.Phase == shopui.Buying {
			quoteSeen = true
		}
		if g.Screen == LevelScreen && !g.backdropOnly && (g.fade == nil || g.fade.Done) {
			break
		}
	}
	if !quoteSeen || g.Screen != LevelScreen || w.Money >= original || w.Equipment.Shield <= 7 || w.Equipment.FireAdvance <= 1 {
		t.Fatalf("demo merchant did not complete real quotes/trades: quote%v screen%d cash%d shield%d fire%d", quoteSeen, g.Screen, w.Money, w.Equipment.Shield, w.Equipment.FireAdvance)
	}
	if w.Money < 0 {
		t.Fatal("demo granted spending beyond the wallet")
	}
}

// This optional bounded run reports actual progress from the ordinary frontend.
// A passing short run is not a five-stage victory or a complete demo-mode gate.
func TestDemoBoundedCampaignProgressOptional(t *testing.T) {
	if os.Getenv("XENON2_DEMO_PROGRESS_CHECK") == "" {
		t.Skip("enable the local bounded pilot progress run explicitly")
	}
	g := frontendGame(t)
	g.Config.Demo = true
	g.Config.LogicPALRefreshes = 3
	duration := 600
	if os.Getenv("XENON2_DEMO_LONG_RUN") != "" {
		duration = 7200
	}
	minScroll := 4608
	levels := map[int]bool{}
	for update := 0; update < 60*duration; update++ {
		advanceFrontend(t, g, inputFrame{})
		if d, ok := g.Driver.(*worldDriver); ok && d.session != nil {
			levels[d.world.Level.Number] = true
			minScroll = min(minScroll, d.world.ScrollY)
			if update%3600 == 0 {
				t.Logf("update%d level%d scroll%d xy%d,%d rewind%d frame%d alive%v material%d lives%d shield%d cash%d screen%d input%+v delta%d max%d min%d", update, d.world.Level.Number, d.world.ScrollY, d.world.Player.X, d.world.Player.Y, d.world.Rewind.Timer, d.world.Frame, d.world.PlayerAlive, d.world.MaterializationFrames, d.world.Equipment.Lives, d.world.Equipment.Shield, d.world.Money, g.Screen, g.demo.controls.gameMotion, d.world.ScrollDelta, d.world.MaximumScrollY, d.world.MinimumScrollY)
			}
		}
	}
	d := g.Driver.(*worldDriver)
	t.Logf("Bounded %d s pilot: stages%v minimumcamera%d level%d lives%d shield%d cash%d; full campaign remains unproven", duration, levels, minScroll, d.world.Level.Number, d.world.Equipment.Lives, d.world.Equipment.Shield, d.world.Money)
}

func TestDemoCompletesFirstLevelFromNormalMenuOptional(t *testing.T) {
	if os.Getenv("XENON2_DEMO_PROGRESS_CHECK") == "" {
		t.Skip("enable the local current-engine demo victory regression explicitly")
	}
	g := frontendGame(t)
	g.Config.Demo = true
	g.Config.LogicPALRefreshes = 3
	middle, final, guardian := false, false, false
	var previousShop *shopui.State
	for update := 0; update < 60*600; update++ {
		advanceFrontend(t, g, inputFrame{})
		d, ok := g.Driver.(*worldDriver)
		if !ok || d.session == nil {
			continue
		}
		if d.world.Level.Number == 1 && d.world.FirstGuardian != nil && d.world.FirstGuardian.Defeated {
			guardian = true
		}
		if g.Screen == ShopScreen && g.shop != nil && g.shop != previousShop {
			if d.world.Level.Number != 1 {
				t.Fatal("advanced beyond the first level before validating its shop route")
			}
			previousShop = g.shop
			if g.shopFinal {
				final = true
				if !guardian || d.world.PendingExitDrops != 0 || !d.world.ExitReady {
					t.Fatal("final merchant admission bypassed guardian destruction or exit coins")
				}
			} else {
				middle = true
			}
		}
		if d.world.Level.Number == 2 {
			if !middle || !final || !guardian || d.world.GameOver || d.diagnostic || !g.DemoActive() {
				t.Fatalf("first-stage route incomplete: middle%v final%v guardian%v", middle, final, guardian)
			}
			t.Logf("Normal-menu pilot completed level1 and both merchants after%d display updates; level2 ships%d shield%d cash%d", update+1, d.world.Equipment.Lives, d.world.Equipment.Shield, d.world.Money)
			return
		}
	}
	t.Fatal("bounded normal-menu pilot did not complete the first level")
}

func TestDemoCompletesFirstTwoLevelsFromNormalMenuOptional(t *testing.T) {
	if os.Getenv("XENON2_DEMO_PROGRESS_CHECK") == "" {
		t.Skip("enable current-engine normal-menu campaign check explicitly")
	}
	for _, refreshes := range []int{2, 3} {
		t.Run(strconv.Itoa(refreshes)+"PAL", func(t *testing.T) {
			verifyDemoFirstTwoLevels(t, refreshes)
		})
	}
}

func verifyDemoFirstTwoLevels(t *testing.T, refreshes int) {
	t.Helper()
	g := frontendGame(t)
	g.Config.Demo = true
	g.Config.LogicPALRefreshes = refreshes
	shops := [3][2]bool{}
	defeated := [3]bool{}
	var lastShop *shopui.State
	for update := 0; update < 60*1200; update++ {
		advanceFrontend(t, g, inputFrame{})
		d, ok := g.Driver.(*worldDriver)
		if !ok || d.session == nil {
			continue
		}
		w := d.world
		level := w.Level.Number
		if level == 1 && w.FirstGuardian != nil && w.FirstGuardian.Defeated {
			defeated[1] = true
		}
		if level == 2 && w.LevelFinished {
			defeated[2] = true
		}
		if g.Screen == ShopScreen && g.shop != nil && g.shop != lastShop {
			lastShop = g.shop
			if level <= 2 {
				slot := 0
				if g.shopFinal {
					slot = 1
					if !defeated[level] || w.PendingExitDrops != 0 || !w.ExitReady {
						t.Fatal("final merchant bypassed actual guardian/exit drops")
					}
				}
				shops[level][slot] = true
				if level == 1 && !g.shopFinal && (w.Equipment.Lives != 3 || w.ContinueCredits != 2) {
					t.Fatal("first defense streams consumed a ship or continue")
				}
			}
		}
		if level == 3 {
			if shops[1] != ([2]bool{true, true}) || shops[2] != ([2]bool{true, true}) || !defeated[1] || !defeated[2] || w.Cheats.Enabled() || w.GameOver || d.diagnostic || w.Equipment.Lives < 2 {
				t.Fatalf("two-stage route incomplete: shops%v defeated%v", shops, defeated)
			}
			t.Logf("Normal-menu pilot completed levels1 and2 after%d display updates; next stage ships%d shield%d cash%d", update+1, w.Equipment.Lives, w.Equipment.Shield, w.Money)
			return
		}
	}
	t.Fatal("normal-input pilot did not complete both original stages")
}

func TestDemoMenuStartsAnUnaidedSinglePlayerSession(t *testing.T) {
	g := frontendGame(t)
	g.Config.Cheats = engine.CheatOptions{InfiniteLives: true, InfiniteEnergy: true, InfiniteMoney: true}
	g.applyCheatOptions()
	for range 4 {
		advanceFrontend(t, g, inputFrame{downPressed: true})
	}
	advanceFrontend(t, g, inputFrame{menuConfirm: true})
	if !g.DemoActive() || g.Config.Cheats.Enabled() || g.pendingPlayers != 1 {
		t.Fatal("demo menu reused trainer state or two-player admission")
	}
	for update := 0; update < 3600; update++ {
		advanceFrontend(t, g, inputFrame{})
		if d, ok := g.Driver.(*worldDriver); ok && d.session != nil && !g.View.Ready && !g.backdropOnly && d.world.Frame > 10 {
			if d.session.PlayerCount != 1 || d.world.Cheats.Enabled() || d.diagnostic {
				t.Fatal("demo menu bypassed ordinary unaided admission")
			}
			return
		}
	}
	t.Fatal("demo menu did not reach ordinary gameplay")
}
