package app

import (
	"testing"

	"xenon2/internal/controls"
	"xenon2/internal/engine"
	"xenon2/internal/presentation"
)

func TestTitleIdleStartsAfterTouchOpensStartupMenu(t *testing.T) {
	bundle := frontendGame(t).Bundle
	g, err := NewConfiguredGame(bundle, Config{Level: 1, StartScreen: PresentationScreen, Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	var pad controls.Pad
	pad.Place(controls.NewLayout(1344))
	button := pad.Layout.Buttons[controls.Enter].Bounds
	contact := controls.Touch{ID: 7, X: button.X + button.Width/2, Y: button.Y + button.Height/2, Pressed: true}
	advanceFrontend(t, g, mergeTouchInput(inputFrame{}, pad.Update([]controls.Touch{contact})))
	contact.Pressed = false
	for range 12 {
		advanceFrontend(t, g, mergeTouchInput(inputFrame{}, pad.Update([]controls.Touch{contact})))
		if g.titleIdleUpdates != 0 || g.DemoActive() {
			t.Fatal("held menu contact counted as inactivity")
		}
	}
	for update := 0; update < 180 && g.Screen != TitleScreen; update++ {
		advanceFrontend(t, g, mergeTouchInput(inputFrame{}, pad.Update(nil)))
	}
	if g.Screen != TitleScreen || g.DemoActive() {
		t.Fatal("released touch did not open the ordinary startup menu")
	}
	remaining := titleDemoIdleUpdates - g.titleIdleUpdates
	for range remaining - 1 {
		advanceFrontend(t, g, mergeTouchInput(inputFrame{}, pad.Update(nil)))
	}
	if g.DemoActive() || g.titleIdleUpdates != titleDemoIdleUpdates-1 {
		t.Fatal("released menu contact did not retain a full idle interval")
	}
	advanceFrontend(t, g, mergeTouchInput(inputFrame{}, pad.Update(nil)))
	if !g.DemoActive() || !g.Config.HumanDemo {
		t.Fatal("menu inactivity did not activate the expert controller")
	}
	awaitFrontendBoundary(t, g, 1800, "touch-opened menu idle demo READY admission", func() bool {
		d, ok := g.Driver.(*worldDriver)
		return ok && d.session != nil && g.Screen == LevelScreen && !g.View.Ready && !g.backdropOnly && d.world.Frame > 20 && (g.fade == nil || g.fade.Done)
	})
	d := g.Driver.(*worldDriver)
	if !g.DemoActive() || d.diagnostic || d.session.PlayerCount != 1 || d.world.Level.Number != 1 || d.world.Cheats.Enabled() {
		t.Fatal("idle demo after a released menu touch bypassed ordinary gameplay")
	}
}

func TestTitleIdleStartsFromStartupAttractAfterSixtySeconds(t *testing.T) {
	bundle := frontendGame(t).Bundle
	g, err := NewConfiguredGame(bundle, Config{Level: 1, StartScreen: PresentationScreen, Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for range titleDemoIdleUpdates - 1 {
		advanceFrontend(t, g, inputFrame{})
	}
	if g.DemoActive() || g.titleIdleUpdates != titleDemoIdleUpdates-1 {
		t.Fatalf("startup attract did not retain idle time: demo%v idle%d phase%s", g.DemoActive(), g.titleIdleUpdates, g.director.Phase)
	}
	advanceFrontend(t, g, inputFrame{})
	if !g.DemoActive() || !g.Config.HumanDemo || g.Config.Level != 1 || g.Config.Cheats.Enabled() || g.pendingPlayers != 1 {
		t.Fatal("startup idle deadline did not start the ordinary expert profile")
	}
	awaitFrontendBoundary(t, g, 1800, "startup idle demo READY admission", func() bool {
		d, ok := g.Driver.(*worldDriver)
		return ok && d.session != nil && g.Screen == LevelScreen && !g.View.Ready && !g.backdropOnly && d.world.Frame > 20 && (g.fade == nil || g.fade.Done)
	})
	d := g.Driver.(*worldDriver)
	if d.diagnostic || d.session.PlayerCount != 1 || d.world.Level.Number != 1 || d.world.Cheats.Enabled() {
		t.Fatal("startup idle demo bypassed normal single-player admission")
	}
	g.clock = engine.NewFrameClock(1, 1)
	frame, x := d.world.Frame, d.world.Player.X
	advanceFrontend(t, g, inputFrame{gameMotion: engine.MotionInput{Right: true}, fire: true, firePressed: true})
	if g.DemoActive() || g.demo != nil || d.world.Frame != frame+1 || d.world.Player.X <= x {
		t.Fatal("manual takeover after startup demo lost the same ordinary action")
	}
}

func TestAttractIdleExcludesInteractivePresentationPhasesAndFlags(t *testing.T) {
	g := frontendGame(t)
	g.Screen = PresentationScreen
	for _, phase := range []presentation.Phase{presentation.ReadyMessage, presentation.GameOverMessage, presentation.ContinueIn, presentation.ContinueHold, presentation.ContinueOut, presentation.InitialsIn, presentation.Initials, presentation.InitialsHold, presentation.InitialsOut, presentation.MenuIn, presentation.MenuOut, presentation.HeaderIn, presentation.HeaderHold, presentation.HeaderOut} {
		g.director.Phase, g.titleIdleUpdates = phase, titleDemoIdleUpdates-1
		g.advanceTitleIdle(inputFrame{})
		if g.DemoActive() || g.titleIdleUpdates != 0 {
			t.Fatalf("interactive phase counted toward attract idle: %s", phase)
		}
	}
	g.director.Phase = presentation.Credits
	for _, flag := range []*bool{&g.readyRunning, &g.gameOverRunning, &g.continueAfterScores} {
		*flag = true
		g.titleIdleUpdates = titleDemoIdleUpdates - 1
		g.advanceTitleIdle(inputFrame{})
		if g.DemoActive() || g.titleIdleUpdates != 0 {
			t.Fatal("active game presentation counted toward attract idle")
		}
		*flag = false
	}
}

func TestAttractIdleManualActivityResetsAcrossPassivePhases(t *testing.T) {
	g := frontendGame(t)
	g.Screen = PresentationScreen
	for _, phase := range []presentation.Phase{presentation.LogoDelay, presentation.LogoIn, presentation.Credits, presentation.LogoOut, presentation.ScoresIn, presentation.ScoresHold, presentation.ScoresOut} {
		g.director.Phase = phase
		g.advanceTitleIdle(inputFrame{})
	}
	if g.titleIdleUpdates != 7 {
		t.Fatal("passive phase transition reset elapsed idle time")
	}
	g.advanceTitleIdle(inputFrame{deviceActivity: true})
	if g.titleIdleUpdates != 0 || g.DemoActive() {
		t.Fatal("manual activity did not reset passive attract inactivity")
	}
}
