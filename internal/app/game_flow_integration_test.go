package app

import (
	"os"
	"testing"
	"xenon2/internal/engine"
	"xenon2/internal/presentation"
)

func frontendGame(t *testing.T) *Game {
	t.Helper()
	dir := os.Getenv("XENON2_RUNTIME_TEST_DIR")
	if dir == "" {
		t.Skip("local exported resources not supplied")
	}
	bundle, err := LoadFS(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	g, err := New(bundle)
	if err != nil {
		t.Fatal(err)
	}
	g.Config.Mute = true
	g.Screen = TitleScreen
	return g
}

func advanceFrontend(t *testing.T, g *Game, input inputFrame) {
	t.Helper()
	if err := g.advanceWithInput(input); err != nil {
		t.Fatal(err)
	}
}

func TestNormalMenuReadyGameplayPauseAndAttract(t *testing.T) {
	g := frontendGame(t)
	advanceFrontend(t, g, inputFrame{menuConfirm: true, anyKey: true})
	ready := false
	for pass := 0; pass < 600; pass++ {
		advanceFrontend(t, g, inputFrame{})
		if g.readyRunning && g.director.Phase == presentation.ReadyMessage && g.director.Data.MessageSteps[g.director.Step] == 17 {
			ready = true
			break
		}
	}
	if !ready {
		t.Fatal("normal menu did not reach its waiting READY director")
	}
	for pass := 0; pass < 160 && (g.Screen != LevelScreen || g.backdropOnly); pass++ {
		advanceFrontend(t, g, inputFrame{confirm: pass == 0, firePressed: pass == 0, anyKey: pass == 0})
	}
	driver := g.Driver.(*worldDriver)
	if g.Screen != LevelScreen || g.View.Ready || driver.session == nil || g.View.Diagnostic {
		t.Fatal("READY did not admit an ordinary player session")
	}
	advanceFrontend(t, g, inputFrame{})
	for range 30 {
		advanceFrontend(t, g, inputFrame{fire: true, gameMotion: engine.MotionInput{Right: true}})
	}
	if driver.world.Frame == 0 || driver.world.Player.X <= 160 {
		t.Fatal("normal sampled controls did not advance movement")
	}
	frame, random, scroll := driver.world.Frame, driver.world.RandomState(), driver.world.ScrollY
	advanceFrontend(t, g, inputFrame{pause: true, anyKey: true})
	for range 60 {
		advanceFrontend(t, g, inputFrame{})
	}
	if !g.paused || driver.world.Frame != frame || driver.world.RandomState() != random || driver.world.ScrollY != scroll {
		t.Fatal("pause advanced world state")
	}
	advanceFrontend(t, g, inputFrame{anyKey: true})
	if g.paused {
		t.Fatal("ordinary key did not resume paused gameplay")
	}
	advanceFrontend(t, g, inputFrame{escape: true, anyKey: true})
	if g.Screen != PresentationScreen || g.director.Phase != presentation.LogoDelay {
		t.Fatal("gameplay Escape did not return to attract")
	}
}
