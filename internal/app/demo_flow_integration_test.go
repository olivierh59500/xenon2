package app

import (
	"os"
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
	minScroll := 4608
	levels := map[int]bool{}
	for update := 0; update < 60*300; update++ {
		advanceFrontend(t, g, inputFrame{})
		if d, ok := g.Driver.(*worldDriver); ok && d.session != nil {
			levels[d.world.Level.Number] = true
			minScroll = min(minScroll, d.world.ScrollY)
			if update%3600 == 0 {
				t.Logf("update%d level%d scroll%d lives%d shield%d cash%d screen%d", update, d.world.Level.Number, d.world.ScrollY, d.world.Equipment.Lives, d.world.Equipment.Shield, d.world.Money, g.Screen)
			}
		}
	}
	d := g.Driver.(*worldDriver)
	t.Logf("Bounded 300 s pilot: stages%v minimumcamera%d level%d lives%d shield%d cash%d; full campaign remains unproven", levels, minScroll, d.world.Level.Number, d.world.Equipment.Lives, d.world.Equipment.Shield, d.world.Money)
}
