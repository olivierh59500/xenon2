package engine

import "testing"

func BenchmarkSecondCorridorSourceRoute(b *testing.B) {
	w, err := NewWorld(playableOriginalWorldData(b, 2))
	if err != nil {
		b.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY = 1100, 1116
	w.Player.X, w.Player.Y = 56, 136
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		b.Fatal("original corridor benchmark starts in solid terrain")
	}
	w.updatePlayerCollision()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		secondCorridorRouteMotion(w, 56, 1200)
	}
}

func TestSecondCorridorPlanningKeepsSourceStateAndAllocatesNothingOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY = 1100, 1116
	w.Player.X, w.Player.Y = 56, 136
	w.updatePlayerCollision()
	player, rewind, random, pool := w.Player, w.Rewind, w.RandomState(), *w.Pool
	camera, maximum := w.ScrollY, w.MaximumScrollY
	expected := secondCorridorRouteMotion(w, 56, 1200)
	allocations := testing.AllocsPerRun(20, func() {
		if got := secondCorridorRouteMotion(w, 56, 1200); got != expected {
			t.Fatal("unchanged corridor produced inconsistent controls")
		}
	})
	if allocations != 0 {
		t.Fatalf("corridor planner allocated %.0f objects per decision", allocations)
	}
	if w.Player != player || w.Rewind != rewind || w.RandomState() != random || *w.Pool != pool || w.ScrollY != camera || w.MaximumScrollY != maximum {
		t.Fatal("corridor planning changed live game state")
	}
}

// This arranges an original post-middle checkpoint boundary, not a campaign
// replay. Terrain, enemies, equipment, ship health and collision stay unchanged;
// traversal after admission uses ordinary public controls and PAL ticks only.
func TestDemoSecondCorridorProactiveLeftExitOptional(t *testing.T) {
	session, err := NewSession(playableOriginalWorldData(t, 2), 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := session.ActiveWorld()
	w.secondMiddleReleased, w.secondDefenseRemaining = true, 0
	w.MinimumScrollY = 0
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 1232, 168
	w.RestartCheckpoint()
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("arranged original checkpoint starts in solid terrain")
	}
	pilot := DemoPilot{}
	for pass := 0; pass < 1100; pass++ {
		w = session.ActiveWorld()
		if w.ScrollY < 800 {
			if pass != 909 || w.Equipment.Lives != 1 || w.Equipment.Shield != 23 || w.Cheats.Enabled() {
				t.Fatalf("corridor outcome pass%d ships%d shield%d", pass, w.Equipment.Lives, w.Equipment.Shield)
			}
			t.Logf("Proactive corridor reached camera%d after%d ordinary commands: lives%d shield%d xy%d,%d", w.ScrollY, pass, w.Equipment.Lives, w.Equipment.Shield, w.Player.X, w.Player.Y)
			return
		}
		if w.GameOver {
			t.Fatal("corridor traversal exhausted its original ships")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		input := pilot.NormalInput(w)
		if _, err = session.Advance(input); err != nil {
			t.Fatal(err)
		}
		w = session.ActiveWorld()

	}
	w = session.ActiveWorld()
	t.Fatalf("bounded corridor did not cross: camera%d xy%d,%d lives%d shield%d", w.ScrollY, w.Player.X, w.Player.Y, w.Equipment.Lives, w.Equipment.Shield)
}
