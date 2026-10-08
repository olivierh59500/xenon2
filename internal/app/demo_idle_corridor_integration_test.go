package app

import (
	"os"
	"testing"

	"xenon2/internal/engine"
)

func TestIdleStartedExpertEscapesThirdCannonPocketOptional(t *testing.T) {
	verifyThreeLevelDemoJourney(t, false, false)
}

func TestIdleStartedExpertCompletesThirdStageOptional(t *testing.T) {
	verifyThreeLevelDemoJourney(t, true, false)
}

func TestExplicitExpertDemoReturnsToMenuAfterThirdStageOptional(t *testing.T) {
	verifyThreeLevelDemoJourney(t, true, true)
}

func verifyThreeLevelDemoJourney(t *testing.T, complete, explicit bool) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the actual idle-start campaign regression explicitly")
	}
	bundle := frontendGame(t).Bundle
	g, err := NewConfiguredGame(bundle, Config{Level: 1, StartScreen: PresentationScreen, Mute: true, Demo: explicit, HumanDemo: explicit})
	if err != nil {
		t.Fatal(err)
	}
	third := false
	var thirdWorld *engine.World
	lives, credits := 0, 0
	defeated := false
	duration := 1500
	if complete {
		duration = 2400
	}
	for update := 0; update < 60*duration; update++ {
		advanceFrontend(t, g, inputFrame{})
		d, ok := g.Driver.(*worldDriver)
		if !ok || d.session == nil {
			continue
		}
		w := d.world
		if w.Level.Number == 3 && !third {
			third = true
			thirdWorld, lives, credits = w, w.Equipment.Lives, w.ContinueCredits
			t.Logf("Actual idle entry: level3 t%.2f ships%d credits%d score%d", float64(update)/60, w.Equipment.Lives, w.ContinueCredits, w.Score)
		}
		if third && (w.GameOver || !w.PlayerAlive || w.Level.Number != 3) {
			t.Fatalf("idle third corridor lost its carried ship: level%d F%d C%d HP%d", w.Level.Number, w.Frame, w.ScrollY, w.Equipment.Shield)
		}
		if third && (w.Equipment.Lives != lives || w.ContinueCredits != credits) {
			t.Fatal("third-stage journey changed its entry ship or credits")
		}
		if complete && third && w.Level.Number == 3 && w.ThirdFinal != nil && w.ThirdFinal.Defeated && !defeated {
			defeated = true
			t.Logf("IDLE_THIRD_GUARDIAN_DEFEATED F%d HP%d score%d", w.Frame, w.Equipment.Shield, w.Score)
			renderIntegrationPixels(t, g, "idle-third-guardian-defeated")
		}
		if complete && third && g.headerAction == headerNextStage {
			t.Fatal("three-level tour started loading a later stage")
		}
		if complete && third && !g.DemoActive() && g.Screen == TitleScreen {
			if !defeated || thirdWorld != w || w.Level.Number != 3 || w.ThirdFinal == nil || !w.ThirdFinal.Defeated || !w.LevelFinished || !w.ExitReady || w.PendingExitDrops != 0 || d.diagnostic || w.Cheats.Enabled() || g.demo != nil || g.shop != nil {
				t.Fatal("three-level tour omitted its real guardian, rewards, final merchant or menu return")
			}
			t.Logf("THREE_LEVEL_TOUR_COMPLETE explicit%v t%.2f F%d HP%d ships%d credits%d score%d", explicit, float64(update)/60, w.Frame, w.Equipment.Shield, w.Equipment.Lives, w.ContinueCredits, w.Score)
			renderIntegrationPixels(t, g, "three-level-tour-menu")
			remaining := titleDemoIdleUpdates - g.titleIdleUpdates
			for range remaining - 1 {
				advanceFrontend(t, g, inputFrame{})
			}
			if g.DemoActive() {
				t.Fatal("completed tour restarted before sixty idle menu seconds")
			}
			advanceFrontend(t, g, inputFrame{})
			if !g.DemoActive() || g.Config.Level != 1 || !g.Config.HumanDemo {
				t.Fatal("completed tour did not restart level-one expert admission after menu inactivity")
			}
			awaitFrontendBoundary(t, g, 1800, "three-level tour restart READY", func() bool {
				d, ok := g.Driver.(*worldDriver)
				return ok && d.world != thirdWorld && d.world.Level.Number == 1 && g.Screen == LevelScreen && !g.View.Ready && !g.backdropOnly && d.world.Frame > 20
			})
			return
		}

		if !complete && third && w.Level.Number == 3 && w.ThirdMiddle != nil && w.ThirdMiddle.Defeated && w.ScrollY < 1400 {
			if !g.DemoActive() || d.diagnostic || w.Cheats.Enabled() || w.Rewind.Timer != 0 || w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
				t.Fatal("corridor admission bypassed ordinary controls")
			}
			t.Logf("IDLE_CORRIDOR_ESCAPED t%.2f F%d C%d P%+v HP%d ships%d credits%d score%d RNG%+v", float64(update)/60, w.Frame, w.ScrollY, w.Player, w.Equipment.Shield, w.Equipment.Lives, w.ContinueCredits, w.Score, w.RandomState())
			renderIntegrationPixels(t, g, "idle-third-corridor-escaped")
			return
		}
		if update%3600 == 0 && third {
			t.Logf("progress F%d C%d P%+v HP%d score%d", w.Frame, w.ScrollY, w.Player, w.Equipment.Shield, w.Score)
		}
	}
	w := g.Driver.(*worldDriver).world
	t.Fatalf("idle route failed to exit third corridor: level%d F%d C%d P%+v HP%d score%d", w.Level.Number, w.Frame, w.ScrollY, w.Player, w.Equipment.Shield, w.Score)
}
