package app

import (
	"testing"
	"xenon2/internal/engine"
	"xenon2/internal/presentation"
)

func TestCheatMenuDefaultsNavigationAndStartingLevel(t *testing.T) {
	g := frontendGame(t)
	if g.Config.Cheats.Enabled() {
		t.Fatal("ordinary menu enabled a trainer option")
	}
	advanceFrontend(t, g, inputFrame{cheatMenu: true})
	if g.Screen != CheatScreen {
		t.Fatal("F3 did not open the cheat menu")
	}
	renderIntegrationPixels(t, g, "cheat-menu-defaults")
	for row := 0; row < 5; row++ {
		advanceFrontend(t, g, inputFrame{confirm: true})
		advanceFrontend(t, g, inputFrame{downPressed: true})
	}
	advanceFrontend(t, g, inputFrame{rightPressed: true})
	if g.Config.Level != 2 || !g.Config.Cheats.InfiniteLives || !g.Config.Cheats.InfiniteCredits || !g.Config.Cheats.InfiniteMoney || !g.Config.Cheats.InfiniteEnergy || !g.Config.Cheats.KeyFunctions {
		t.Fatal("trainer menu omitted an original option or level choice")
	}
	advanceFrontend(t, g, inputFrame{escape: true})
	if g.Screen != TitleScreen {
		t.Fatal("cheat menu did not return to its caller")
	}
	advanceFrontend(t, g, inputFrame{menuConfirm: true})
	awaitFrontendBoundary(t, g, 800, "selected level READY", func() bool { return g.readyRunning })
	d := g.Driver.(*worldDriver)
	if d.world.Level.Number != 2 || d.world.Cheats != g.Config.Cheats || d.world.Equipment.Lives != 3 {
		t.Fatal("starting level or trainer options were not admitted through the ordinary menu")
	}
}

func TestCheatMenuSuspendsGameplayAndDisabledKeysDoNothing(t *testing.T) {
	g := menuAdmittedGame(t, 1, 1)
	d := g.Driver.(*worldDriver)
	before := d.world.Equipment
	advanceFrontend(t, g, inputFrame{cheatItem: engine.ItemCannon})
	if d.world.Equipment != before {
		t.Fatal("disabled trainer key granted equipment")
	}
	frame := d.world.Frame
	advanceFrontend(t, g, inputFrame{cheatMenu: true})
	for range 120 {
		advanceFrontend(t, g, inputFrame{})
	}
	if d.world.Frame != frame {
		t.Fatal("trainer settings advanced the gameplay world")
	}
	g.cheatRow = 4
	advanceFrontend(t, g, inputFrame{confirm: true})
	advanceFrontend(t, g, inputFrame{cheatMenu: true})
	advanceFrontend(t, g, inputFrame{cheatItem: engine.ItemCannon})
	found := false
	for _, slot := range d.world.Equipment.Mounts {
		found = found || slot.Item == engine.ItemCannon
	}
	if g.Screen != LevelScreen || !found {
		t.Fatal("enabled trainer equipment key did not work after returning to the game")
	}
	advanceFrontend(t, g, inputFrame{cheatMenu: true})
	g.cheatRow = 4
	advanceFrontend(t, g, inputFrame{confirm: true})
	advanceFrontend(t, g, inputFrame{escape: true})
	before = d.world.Equipment
	advanceFrontend(t, g, inputFrame{cheatItem: engine.ItemLaser})
	if d.world.Equipment != before {
		t.Fatal("turning off trainer keys retained permission to grant equipment")
	}
}

func TestUnlimitedContinueMenuStillOffersZeroCreditSession(t *testing.T) {
	g := menuAdmittedGame(t, 1, 1)
	d := g.Driver.(*worldDriver)
	// This is a final-loss admission fixture. The chosen aid must be checked
	// by the frontend as well as the world's continue debit rule.
	d.world.GameOver = true
	d.world.ContinueCredits = 0
	d.world.Equipment.Lives = 0
	d.world.PlayerAlive = false
	for row := range g.director.Scores {
		g.director.Scores[row].Points = 1000000
	}
	g.Config.Cheats.InfiniteCredits = true
	g.applyCheatOptions()
	g.View = d.Frame()
	advanceFrontend(t, g, inputFrame{})
	if g.director.Phase != presentation.ContinueIn {
		t.Fatal("unlimited credits were rejected by the frontend's finite-credit gate")
	}
}
