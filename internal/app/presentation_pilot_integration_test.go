package app

import (
	"os"
	"testing"
	"xenon2/internal/engine"
)

// This runs ordinary frontend controls from the real intro. Motion measurements
// establish activity, not human appearance or complete campaign success; video
// inspection remains separate from this bounded regression.
func TestPresentationPilotMovesAndSelectsFireThroughRealFrontendOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the real presentation pilot check explicitly")
	}
	g := frontendGame(t)
	g.Config.Demo, g.Config.HumanDemo = true, true
	g.BeginAttract()
	var lastFrame uint64 = ^uint64(0)
	var lastPlayer engine.PlayerMotionState
	passes, moving, firing, resting := 0, 0, 0, 0
	quietRun, longestQuietRun := 0, 0
	minimumX, maximumX, minimumY, maximumY := ScreenWidth, 0, PlayfieldHeight, 0
	for update := 0; update < 60*160; update++ {
		advanceFrontend(t, g, inputFrame{})
		if g.Screen != LevelScreen || g.backdropOnly || g.View.Ready {
			continue
		}
		w := g.Driver.(*worldDriver).world
		if w.Frame == lastFrame {
			continue
		}
		if w.Cheats.Enabled() || w.Level.Number != 1 || w.GameOver || !g.DemoActive() {
			t.Fatal("presentation did not retain the ordinary first-stage rules")
		}
		if passes > 0 && (w.Player.X != lastPlayer.X || w.Player.Y != lastPlayer.Y) {
			moving++
		}
		if g.demo.controls.fire {
			firing++
			quietRun = 0
		} else {
			resting++
			quietRun++
			longestQuietRun = max(longestQuietRun, quietRun)
		}
		minimumX, maximumX = min(minimumX, w.Player.X), max(maximumX, w.Player.X)
		minimumY, maximumY = min(minimumY, w.Player.Y), max(maximumY, w.Player.Y)
		passes++
		lastFrame, lastPlayer = w.Frame, w.Player
	}
	if passes < 500 || moving < passes/5 || maximumX-minimumX < 80 || resting < passes/4 || firing == 0 || longestQuietRun < 12 {
		t.Fatalf("presentation activity insufficient: passes%d moving%d firing%d resting%d x%d..%d y%d..%d", passes, moving, firing, resting, minimumX, maximumX, minimumY, maximumY)
	}
	t.Logf("Ordinary presentation: %d passes, %d moving, %d firing, %d resting; longest quiet%d; x%d..%d y%d..%d", passes, moving, firing, resting, longestQuietRun, minimumX, maximumX, minimumY, maximumY)
}
