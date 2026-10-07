package app

import (
	"testing"

	"xenon2/internal/controls"
	"xenon2/internal/engine"
	"xenon2/internal/presentation"
)

func TestTitleIdleStartsExpertDemoAtSixtySeconds(t *testing.T) {
	g := frontendGame(t)
	g.Config.Level = 5
	g.Config.Cheats = engine.CheatOptions{InfiniteLives: true, InfiniteCredits: true, InfiniteMoney: true, InfiniteEnergy: true, KeyFunctions: true}
	g.applyCheatOptions()
	g.menu = 1
	for range titleDemoIdleUpdates - 1 {
		advanceFrontend(t, g, inputFrame{})
	}
	if g.DemoActive() || g.Screen != TitleScreen || g.titleIdleUpdates != titleDemoIdleUpdates-1 {
		t.Fatal("title demo started before sixty idle display seconds")
	}
	advanceFrontend(t, g, inputFrame{})
	if !g.DemoActive() || !g.Config.HumanDemo || g.Config.Cheats.Enabled() || g.Config.Level != 1 || g.pendingPlayers != 1 || g.Screen == TitleScreen {
		t.Fatal("idle expiry did not start an expert, unaided single-player admission")
	}
	awaitFrontendBoundary(t, g, 1800, "idle demo ordinary READY admission", func() bool {
		d, ok := g.Driver.(*worldDriver)
		return ok && d.session != nil && g.Screen == LevelScreen && !g.View.Ready && !g.backdropOnly && d.world.Frame > 20
	})
	d := g.Driver.(*worldDriver)
	if d.session.PlayerCount != 1 || d.world.Level.Number != 1 || d.world.Cheats.Enabled() || d.diagnostic || !g.DemoActive() {
		t.Fatal("idle demo or its generated interface controls bypassed normal admission")
	}
}

func TestTitleIdleRestartsAfterManualActivity(t *testing.T) {
	g := frontendGame(t)
	advanceFrontend(t, g, inputFrame{})
	heldTouch := controls.Frame{}
	heldTouch.Held[controls.Dive] = true
	for _, fixture := range []struct {
		name  string
		input inputFrame
	}{
		{"keyboard edge", inputFrame{anyKey: true}},
		{"held device or wheel", inputFrame{deviceActivity: true}},
		{"held motion", inputFrame{gameMotion: engine.MotionInput{Right: true}}},
		{"held fire", inputFrame{fire: true}},
		{"mouse click", inputFrame{mousePressed: true, mouseX: -1, mouseY: -1}},
		{"pointer movement", inputFrame{mouseX: 37, mouseY: 42}},
		{"held touch", mergeTouchInput(inputFrame{}, heldTouch)},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			g.titleIdleUpdates = titleDemoIdleUpdates - 1
			advanceFrontend(t, g, fixture.input)
			if g.DemoActive() || g.titleIdleUpdates != 0 || g.Screen != TitleScreen {
				t.Fatal("manual activity did not reset the title idle interval")
			}
		})
	}
	// Returning the pointer to its recorded-test origin is itself activity.
	advanceFrontend(t, g, inputFrame{mouseX: 1})
	advanceFrontend(t, g, inputFrame{})
	for range titleDemoIdleUpdates - 1 {
		advanceFrontend(t, g, inputFrame{})
	}
	if g.DemoActive() {
		t.Fatal("title demo reused elapsed time from before manual activity")
	}
	advanceFrontend(t, g, inputFrame{})
	if !g.DemoActive() {
		t.Fatal("title demo did not start after a fresh sixty idle seconds")
	}
}

func TestTitleIdleDoesNotCountSuspendedScreens(t *testing.T) {
	fade := presentation.NewPaletteFadeIn()
	for _, fixture := range []struct {
		name   string
		screen Screen
		paused bool
		fade   *presentation.PaletteFade
	}{
		{"fade", TitleScreen, false, &fade},
		{"pause", TitleScreen, true, nil},
		{"cheat menu", CheatScreen, false, nil},
		{"shop", ShopScreen, false, nil},
		{"presentation", PresentationScreen, false, nil},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			g := &Game{Screen: fixture.screen, paused: fixture.paused, fade: fixture.fade, titleIdleUpdates: titleDemoIdleUpdates - 1}
			g.advanceTitleIdle(inputFrame{})
			if g.DemoActive() || g.titleIdleUpdates != 0 {
				t.Fatal("inactive title state counted toward automatic demo admission")
			}
		})
	}
}

func TestExpertDemoTakeoverAppliesTheSameKeyboardOrTouchAction(t *testing.T) {
	touch := controls.Frame{X: 1, AnyPressed: true}
	touch.Pressed[controls.Fire], touch.Held[controls.Fire] = true, true
	for _, fixture := range []struct {
		name  string
		input inputFrame
	}{
		{"keyboard", inputFrame{gameMotion: engine.MotionInput{Right: true}, fire: true, firePressed: true}},
		{"touch", mergeTouchInput(inputFrame{}, touch)},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			g := menuAdmittedGame(t, 1, 1)
			awaitFrontendBoundary(t, g, 120, "gameplay fade completion", func() bool { return g.fade == nil || g.fade.Done })
			g.Config.Demo, g.Config.HumanDemo = true, true
			g.demo = &demoDirector{}
			g.clock = engine.NewFrameClock(1, 1)
			d := g.Driver.(*worldDriver)
			world, frame, x := d.world, d.world.Frame, d.world.Player.X
			advanceFrontend(t, g, fixture.input)
			if g.DemoActive() || g.demo != nil || d.world != world || d.world.Frame != frame+1 || d.world.Player.X <= x || g.pendingFire {
				t.Fatal("takeover consumed the action or replaced the ordinary active world")
			}
		})
	}
}

func TestMenuDemoSelectsExpertProfileAndHeldFireTakesOver(t *testing.T) {
	g := frontendGame(t)
	g.menu = 4
	advanceFrontend(t, g, inputFrame{menuConfirm: true})
	if !g.DemoActive() || !g.Config.HumanDemo {
		t.Fatal("menu demo did not select the expert controller")
	}
	manual := inputFrame{fire: true}
	if got := g.demoControls(manual); got != manual || g.DemoActive() || g.demo != nil {
		t.Fatal("held fire did not return the same controls to the human player")
	}
}
