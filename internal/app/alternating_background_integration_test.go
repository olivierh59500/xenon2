package app

import (
	"testing"

	"xenon2/internal/engine"
	"xenon2/internal/presentation"
)

// Use ordinary menu admission and real collision deaths in both directions.
// No background phase, world completion or death flag is assigned by the test.
func TestAlternatingCollisionDeathsKeepSharedBackdropThroughReady(t *testing.T) {
	g := menuAdmittedGame(t, 1, 2)
	d := g.Driver.(*worldDriver)
	if d.session.Players[0].Pool.Slot(0) != d.session.Players[1].Pool.Slot(0) {
		t.Fatal("two-player menu admission created separate physical reserves")
	}
	for turn := 0; turn < 2; turn++ {
		outgoing, player := d.world, d.session.Current
		deathSeen := false
		for update := 0; update < 2400 && d.session.Current == player; update++ {
			advanceFrontend(t, g, inputFrame{gameMotion: engine.MotionInput{Left: true, Up: true}})
			deathSeen = deathSeen || !outgoing.PlayerAlive
		}
		phase := outgoing.BackgroundY
		display := float64((192 - phase) % 192)
		if !deathSeen || phase == 0 || d.session.Current != player^1 || d.world == outgoing || d.world.GameOver || d.world.Cheats.Enabled() || !g.readyRunning || g.View.PlayerNumber != player^1+1 || g.View.BackgroundY != display || g.previous.BackgroundY != display {
			t.Fatalf("collision turn%d resumed stale backdrop: player%d phase%d view%g previous%g", turn, d.session.Current, phase, g.View.BackgroundY, g.previous.BackgroundY)
		}
		incoming := d.world
		if incoming.Pool.Slot(0) != outgoing.Pool.Slot(0) {
			t.Fatal("collision turn admission detached the shared physical reserve")
		}
		awaitFrontendBoundary(t, g, 800, "incoming player's waiting READY", func() bool {
			return g.director.Phase == presentation.ReadyMessage && g.director.Data.MessageSteps[g.director.Step] == 17
		})
		if incoming.BackgroundY != phase || g.View.BackgroundY != display {
			t.Fatal("READY presentation reset or advanced the shared background")
		}
		frame := incoming.Frame
		advanceFrontend(t, g, inputFrame{confirm: true, firePressed: true, anyKey: true})
		awaitFrontendBoundary(t, g, 240, "incoming player's gameplay", func() bool {
			return g.Screen == LevelScreen && !g.View.Ready && !g.backdropOnly
		})
		if d.world != incoming || incoming.Frame != frame || incoming.BackgroundY != phase || g.View.BackgroundY != display {
			t.Fatal("READY admission or gameplay fade changed the retained backdrop phase")
		}
		awaitFrontendBoundary(t, g, 120, "incoming player's first logic pass", func() bool { return incoming.Frame > frame })
		if incoming.Frame != frame+1 || incoming.PreviousBackgroundY != phase || g.View.BackgroundY != float64((192-incoming.BackgroundY)%192) {
			t.Fatal("first incoming pass did not expose the continued shared backdrop")
		}
	}
}
