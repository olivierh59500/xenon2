package app

import (
	"os"
	"testing"
)

func TestIdleStartedExpertEscapesThirdCannonPocketOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the actual idle-start campaign regression explicitly")
	}
	bundle := frontendGame(t).Bundle
	g, err := NewConfiguredGame(bundle, Config{Level: 1, StartScreen: PresentationScreen, Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	third := false
	for update := 0; update < 60*1500; update++ {
		advanceFrontend(t, g, inputFrame{})
		d, ok := g.Driver.(*worldDriver)
		if !ok || d.session == nil {
			continue
		}
		w := d.world
		if w.Level.Number == 3 && !third {
			third = true
			t.Logf("Actual idle entry: level3 t%.2f ships%d credits%d score%d", float64(update)/60, w.Equipment.Lives, w.ContinueCredits, w.Score)
		}
		if third && (w.GameOver || !w.PlayerAlive || w.Level.Number != 3 && w.Level.Number != 4) {
			t.Fatalf("idle third corridor lost its carried ship: level%d F%d C%d HP%d", w.Level.Number, w.Frame, w.ScrollY, w.Equipment.Shield)
		}
		if third && w.Level.Number == 3 && w.ThirdMiddle != nil && w.ThirdMiddle.Defeated && w.ScrollY < 1400 {
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
