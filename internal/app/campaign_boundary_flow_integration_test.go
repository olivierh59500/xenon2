package app

import (
	"reflect"
	"testing"

	"xenon2/internal/engine"
	"xenon2/internal/presentation"
	"xenon2/internal/shopui"
)

func awaitFrontendBoundary(t *testing.T, g *Game, limit int, description string, predicate func() bool) {
	t.Helper()
	for pass := 0; pass < limit; pass++ {
		if predicate() {
			return
		}
		advanceFrontend(t, g, inputFrame{})
	}
	phase := shopui.Phase("")
	if g.shop != nil {
		phase = g.shop.Phase
	}
	t.Fatalf("%s not reached: screen%d presentation%s shop%s", description, g.Screen, g.director.Phase, phase)
}

func menuAdmittedGame(t *testing.T, level, players int) *Game {
	t.Helper()
	g := frontendGame(t)
	// A nonfirst starting level is an explicit stage-boundary fixture. Admission
	// still passes through the production menu, loading and READY directors.
	g.Config.Level = level
	if players == 2 {
		advanceFrontend(t, g, inputFrame{downPressed: true, anyKey: true})
	}
	advanceFrontend(t, g, inputFrame{menuConfirm: true, anyKey: true})
	awaitFrontendBoundary(t, g, 800, "waiting READY caption", func() bool {
		return g.readyRunning && g.director.Phase == presentation.ReadyMessage && g.director.Data.MessageSteps[g.director.Step] == 17
	})
	advanceFrontend(t, g, inputFrame{confirm: true, firePressed: true, anyKey: true})
	awaitFrontendBoundary(t, g, 240, "playable menu admission", func() bool { return g.Screen == LevelScreen && !g.backdropOnly && !g.View.Ready })
	d, ok := g.Driver.(*worldDriver)
	if !ok || d.session == nil || d.session.PlayerCount != players || g.View.Diagnostic {
		t.Fatal("menu admission bypassed the normal session")
	}
	return g
}

func collisionDeathBoundaryGame(t *testing.T, qualifyingScore bool) *Game {
	t.Helper()
	g := menuAdmittedGame(t, 1, 1)
	w := g.Driver.(*worldDriver).world
	// Only the last-life/ranking boundary is arranged. The death itself must
	// result from ordinary controls and the real terrain/contact simulation.
	w.Equipment.Lives = 1
	if qualifyingScore {
		w.Score, w.DisplayScore = 500, 500
	} else {
		// Collision play can still earn points. Arrange an explicitly
		// nonqualifying table instead of assuming an unearned zero score.
		for row := range g.director.Scores {
			g.director.Scores[row].Points = 1000000 - row*1000
		}
	}
	deathSeen := false
	for pass := 0; pass < 2400; pass++ {
		advanceFrontend(t, g, inputFrame{gameMotion: engine.MotionInput{Left: true, Up: true}})
		deathSeen = deathSeen || !w.PlayerAlive
		if g.Screen == PresentationScreen && g.gameOverRunning {
			if !deathSeen || !w.GameOver || w.Equipment.Lives != 0 {
				t.Fatal("UI game-over route did not follow the collision death and final ship loss")
			}
			t.Logf("Real collision death reached final-loss UI after%d display updates, simulation pass%d, score%d", pass+1, w.Frame, w.Score)
			return g
		}
	}
	t.Fatal("bounded ordinary movement did not produce a final collision death")
	return nil
}

func TestNormalMenuAdmissionAndControlsAreReproducible(t *testing.T) {
	first, second := menuAdmittedGame(t, 1, 1), menuAdmittedGame(t, 1, 1)
	for pass := 0; pass < 240; pass++ {
		controls := inputFrame{fire: true, gameMotion: engine.MotionInput{Right: pass < 60, Left: pass >= 120 && pass < 180}}
		advanceFrontend(t, first, controls)
		advanceFrontend(t, second, controls)
		a, b := first.Driver.(*worldDriver).world, second.Driver.(*worldDriver).world
		if first.Screen != second.Screen || first.View.Player != second.View.Player || a.Frame != b.Frame || a.ScrollY != b.ScrollY || a.Score != b.Score || a.Equipment != b.Equipment || a.RandomState() != b.RandomState() || !reflect.DeepEqual(a.Level.Terrain.Map, b.Level.Terrain.Map) {
			t.Fatalf("identical menu/control sequence diverged at display pass%d", pass)
		}
	}
}

func TestCollisionDeathInitialsAndAcceptedContinueBoundary(t *testing.T) {
	g := collisionDeathBoundaryGame(t, true)
	d := g.Driver.(*worldDriver)
	oldWorld := d.world
	awaitFrontendBoundary(t, g, 240, "initials input", func() bool { return g.director.Phase == presentation.Initials })
	row, score := g.director.InitialRow, g.director.Scores[g.director.InitialRow].Points
	for character := 0; character < 3; character++ {
		for pass := 0; pass < 8 && g.director.InitialLetter == 0; pass++ {
			advanceFrontend(t, g, inputFrame{right: true})
		}
		advanceFrontend(t, g, inputFrame{confirm: true, anyKey: true})
		awaitFrontendBoundary(t, g, 8, "confirmed initials character", func() bool { return g.director.InitialCharacter == character+1 })
	}
	if g.director.Scores[row].Initials != "BBB" || g.director.Scores[row].Points != score {
		t.Fatal("initials editing changed score order or failed to retain three selected letters")
	}
	renderIntegrationPixels(t, g, "flow-initials-confirmed")
	awaitFrontendBoundary(t, g, 400, "continue offer", func() bool { return g.director.Phase == presentation.ContinueHold })
	credits := oldWorld.ContinueCredits
	advanceFrontend(t, g, inputFrame{confirm: true, anyKey: true})
	awaitFrontendBoundary(t, g, 800, "continued READY caption", func() bool {
		return g.readyRunning && g.director.Phase == presentation.ReadyMessage && g.director.Data.MessageSteps[g.director.Step] == 17
	})
	if d.world != oldWorld || d.world.GameOver || d.world.Equipment.Lives != 3 || d.world.ContinueCredits != credits-1 || d.world.Score != 0 || g.director.Scores[row].Initials != "BBB" {
		t.Fatal("continue failed to restore the same session while retaining the entered high score")
	}
	advanceFrontend(t, g, inputFrame{confirm: true, firePressed: true, anyKey: true})
	awaitFrontendBoundary(t, g, 240, "continued gameplay", func() bool { return g.Screen == LevelScreen && !g.View.Ready && !g.backdropOnly })
	if g.View.Diagnostic || !d.world.PlayerAlive {
		t.Fatal("accepted continue returned to a reference scene or a dead ship")
	}
}

func TestCollisionDeathContinueTimeoutReturnsToAttractBoundary(t *testing.T) {
	g := collisionDeathBoundaryGame(t, false)
	awaitFrontendBoundary(t, g, 240, "nonranking continue offer", func() bool { return g.director.Phase == presentation.ContinueHold })
	w := g.Driver.(*worldDriver).world
	credits := w.ContinueCredits
	renderIntegrationPixels(t, g, "flow-continue-offer")
	seenGameOver := false
	for pass := 0; pass < 600; pass++ {
		advanceFrontend(t, g, inputFrame{})
		seenGameOver = seenGameOver || g.director.Phase == presentation.GameOverMessage
		if seenGameOver && !g.gameOverRunning && g.Screen == PresentationScreen && g.director.Phase == presentation.LogoDelay {
			break
		}
	}
	if !seenGameOver || g.gameOverRunning || g.director.Phase != presentation.LogoDelay || !w.GameOver || w.ContinueCredits != credits {
		t.Fatal("unanswered continue did not finish the player game without consuming a continue credit")
	}
}

func awaitShopMode(t *testing.T, g *Game, phase shopui.Phase) {
	t.Helper()
	awaitFrontendBoundary(t, g, 2400, "usable merchant mode", func() bool {
		return g.Screen == ShopScreen && g.shop != nil && g.shop.Phase == phase && !g.shop.Busy() && g.shop.HandRemaining <= 0 && g.shop.Revealed >= len(g.shop.Dialogue) && g.shop.DisplayMoney == *g.shop.Money && (g.fade == nil || g.fade.Done)
	})
}

func TestMiddleShopPurchaseAndReloadBoundary(t *testing.T) {
	g := menuAdmittedGame(t, 1, 1)
	d := g.Driver.(*worldDriver)
	w := d.world
	// This fixture starts at the engine's middle-shop request boundary; it
	// does not assert that a player defeated the middle arena in this test.
	w.Money = 5000
	w.ShopReady, w.LevelFinished, w.ExitReady = true, false, false
	awaitFrontendBoundary(t, g, 600, "merchant entry", func() bool { return g.Screen == ShopScreen })
	awaitShopMode(t, g, shopui.Selling)
	if g.shop.Ending || g.shopFinal || g.shop.Rules.Level != 1 || g.shop.Rules.StockLimit != w.Level.Terrain.MidShopStockLimit {
		t.Fatal("middle reward used final-shop rules")
	}
	advanceFrontend(t, g, inputFrame{mousePressed: true, mouseX: 20, mouseY: 180})
	awaitShopMode(t, g, shopui.Buying)
	index := -1
	for i, entry := range g.shop.Entries {
		if entry.Item == engine.ItemAutofire && entry.Available {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("affordable original autofire catalogue entry missing")
	}
	var cell visualShopCell
	for _, candidate := range g.Bundle.ShopScene.Cells {
		if candidate.Column == index%5 && candidate.Row == index/5 {
			cell = visualShopCell{candidate.X, candidate.Y}
			break
		}
	}
	if cell.x == 0 && cell.y == 0 {
		t.Fatal("original merchant click geometry missing")
	}
	price, err := g.shop.Rules.Price(engine.ItemAutofire)
	if err != nil {
		t.Fatal(err)
	}
	advanceFrontend(t, g, inputFrame{mousePressed: true, mouseX: cell.x + 2, mouseY: cell.y + 2})
	awaitShopMode(t, g, shopui.Buying)
	if !g.shop.QuoteValid || w.Money != 5000 {
		t.Fatal("merchant quote consumed money before confirmation")
	}
	advanceFrontend(t, g, inputFrame{confirm: true, anyKey: true})
	awaitShopMode(t, g, shopui.Buying)
	if w.Money != 5000-price || w.Equipment.FireAdvance != 2 {
		t.Fatal("confirmed merchant purchase did not update live inventory and wallet")
	}
	renderIntegrationPixels(t, g, "flow-middle-merchant-purchase")
	scroll, player, random := w.ScrollY, w.Player, w.RandomState()
	advanceFrontend(t, g, inputFrame{mousePressed: true, mouseX: 20, mouseY: 180})
	awaitFrontendBoundary(t, g, 700, "same-stage reload", func() bool { return g.Screen == LevelScreen && !g.backdropOnly && (g.fade == nil || g.fade.Done) })
	if d.world != w || w.Level.Number != 1 || w.ShopReady || w.ExitReady || w.Equipment.FireAdvance != 2 || w.Money != 5000-price || w.ScrollY != scroll || w.Player != player || w.RandomState() == random {
		t.Fatal("shop return restarted the game, lost the purchase or omitted its shared presentation RNG work")
	}
}

type visualShopCell struct{ x, y int }

func TestFifthStageMerchantEndingAndSharedNextStageBoundary(t *testing.T) {
	g := menuAdmittedGame(t, 5, 2)
	d := g.Driver.(*worldDriver)
	s := d.session
	first := d.world
	// Both victories are explicit completion-boundary fixtures. The UI must
	// use the normal shared-level gate; this is not a full campaign playthrough.
	first.ShopReady, first.LevelFinished, first.ExitReady = true, true, true
	awaitFrontendBoundary(t, g, 12, "second surviving player's turn", func() bool { return s.Current == 1 && d.world != first })
	if g.shop != nil || !s.Completed[0] || s.Completed[1] || d.world.Level.Number != 5 {
		t.Fatal("first surviving completion incorrectly displayed the final merchant ending")
	}
	awaitFrontendBoundary(t, g, 800, "second player's READY caption", func() bool {
		return g.readyRunning && g.director.Phase == presentation.ReadyMessage && g.director.Data.MessageSteps[g.director.Step] == 17
	})
	advanceFrontend(t, g, inputFrame{confirm: true, firePressed: true, anyKey: true})
	awaitFrontendBoundary(t, g, 240, "second player's gameplay", func() bool { return g.Screen == LevelScreen && !g.backdropOnly && !g.View.Ready })
	second := d.world
	second.Equipment.ApplyItem(engine.ItemCannon)
	second.ShopReady, second.LevelFinished, second.ExitReady = true, true, true
	awaitFrontendBoundary(t, g, 600, "final merchant entry", func() bool { return g.Screen == ShopScreen })
	if !g.shop.Ending || !g.shopFinal || g.shop.Rules.Level != 5 {
		t.Fatal("last surviving completion omitted the merchant ending")
	}
	seen := map[shopui.Phase]bool{}
	for pass := 0; pass < 4000; pass++ {
		if g.Screen == ShopScreen && g.shop != nil {
			seen[g.shop.Phase] = true
			if g.shop.Phase == shopui.Selling || g.shop.Phase == shopui.Buying {
				t.Fatal("final ending exposed a purchase/sale session")
			}
		}
		if d.world.Level.Number == 1 {
			break
		}
		advanceFrontend(t, g, inputFrame{})
	}
	for _, phase := range []shopui.Phase{shopui.MerchantEnding, shopui.EndingFade, shopui.EndingDot, shopui.EndingDotFade, shopui.EndingWait} {
		if !seen[phase] {
			t.Fatalf("merchant ending skipped production phase%s", phase)
		}
	}
	if s.Difficulty != 2 || s.Current != 0 || s.Completed != ([2]bool{}) || s.Players[0].Level.Number != 1 || s.Players[1].Level.Number != 1 || s.Players[0].ContinueCredits != 3 || s.Players[1].ContinueCredits != 3 || s.Players[1].Equipment.Mounts[0].Item != engine.ItemNone || s.Players[1].Equipment.Primary.Item != engine.ItemForwardShot {
		t.Fatal("merchant ending did not reset ordinary weapons and admit both players to the next campaign round")
	}
	if s.Players[0].Pool.Slot(0) != s.Players[1].Pool.Slot(0) {
		t.Fatal("second difficulty round created separate actor reserves for the saved games")
	}
}
