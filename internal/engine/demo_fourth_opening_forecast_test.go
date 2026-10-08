package engine

import "testing"

// These isolated source scenes retain the equipment shape earned by the real
// three-level route. They do not recreate its random stream or claim a campaign.
func fourthOpeningSourceFixture(t testing.TB, camera, x, y, inertia int) *World {
	t.Helper()
	equipment := NewEquipment()
	for _, item := range []Item{ItemDoubleShot, ItemDoubleShot, ItemRearShot, ItemSpeedup, ItemSpeedup, ItemAutofire, ItemAutofire} {
		equipment.ApplyItem(item)
	}
	data := playableOriginalWorldData(t, 4)
	data.InitialEquipment = &equipment
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = camera, camera+16, camera+16
	w.Player.X, w.Player.Y, w.Player.Inertia, w.Player.SpeedTier = x, y, inertia, 2
	w.Rewind = NewTerrainRewind(camera, x, y)
	w.Ready, w.MaterializationFrames = false, 0
	w.cursor = RestartEncounterCursor(camera)
	if w.Coverage.Touches(x, y, camera, *w.Level.PlayerStencil) {
		t.Fatal("original fourth opening pose starts in covered terrain")
	}
	return w
}

func assertFourthOpeningAvoidance(t testing.TB, w *World, heldShield int, contact bool) {
	t.Helper()
	before := forecastIsolationDigest(w)
	planned := Input{}
	pilot := PresentationPilot{}
	guarded := pilot.forecastOpeningGuard(w, planned)
	if guarded.Motion == planned.Motion || guarded.Fire != planned.Fire || guarded.Dive != planned.Dive {
		t.Fatal("fourth opening guard did not replace unsafe movement alone")
	}
	held, heldContacts := sixThirdFinalGuardPasses(t, w, planned)
	safe, safeContacts := sixThirdFinalGuardPasses(t, w, guarded)
	if !held.Alive || held.Shield != heldShield || !safe.Alive || safe.Shield != 39 || (heldContacts > 0) != contact || safeContacts != 0 {
		t.Fatalf("source avoidance: held%+v contacts%d guarded%+v contacts%d", held, heldContacts, safe, safeContacts)
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("fourth opening prediction changed source state, inventory or randomness")
	}
}

func TestFourthOpeningGuardAvoidsObservedDiagonalBulletsOptional(t *testing.T) {
	for _, fixture := range []struct {
		frame, x, inertia int
		impactX, impactY  int32
	}{
		{137, 129, 2, 7901968, 7105776},
		{138, 138, 3, 8956400, 7886352},
	} {
		// The actual reference projectile reached these fixed-point coordinates
		// during its impact pass. Recover its pose four ordinary advances earlier,
		// retaining the original fractional motion and source collision prefix.
		w := fourthOpeningSourceFixture(t, 4608-fixture.frame+4, fixture.x, 111, fixture.inertia)
		w.Frame = uint64(fixture.frame - 4)
		dx := int32(directionX[5]) * 6 * 4
		dy := int32(directionY[5])*6*4 + 1<<16
		x, y := fixture.impactX-4*dx, fixture.impactY-4*dy
		w.spawnEnemyShot(int(x>>16), int(y>>16), EnemyShot{Direction: 5, Speed: 6})
		if len(w.Projectiles) != 1 || w.Projectiles[0].Sprite != w.Level.Rules.DefaultEnemyShot {
			t.Fatal("ordinary source projectile factory did not select the original bank")
		}
		shot := w.Projectiles[0]
		shot.Motion.X, shot.Motion.Y = x, y
		shot.X, shot.Y = float64(x>>16), float64(y>>16)
		check := shot.Motion
		for range 4 {
			if _, err := check.Advance(1); err != nil {
				t.Fatal(err)
			}
		}
		if check.X != fixture.impactX || check.Y != fixture.impactY {
			t.Fatal("captured ordinary projectile no longer matches its original advance")
		}
		assertFourthOpeningAvoidance(t, w, 35, false)
	}
}

func TestFourthOpeningGuardAnticipatesOriginalCurvedWaveContactOptional(t *testing.T) {
	w := fourthOpeningSourceFixture(t, 4444, 129, 91, 0)
	found := false
	for _, wave := range w.Level.Encounters.Moving {
		if wave.TriggerY == 4464 && wave.PathID == 31 {
			if err := w.spawnWave(wave); err != nil {
				t.Fatal(err)
			}
			found = true
			break
		}
	}
	if !found || len(w.Actors) != 3 {
		t.Fatal("original three-member curved formation was not admitted")
	}
	// Replay the source family that caused the observed169 contact into its
	// curved approach. This isolates the wave rather than reconstructing all
	// earlier campaign callbacks, hits and random draws.
	for range 20 {
		w.Frame++
		if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
			t.Fatal(err)
		}
	}
	leading := w.Actors[2]
	if leading.part.ResourceTag != 236 {
		t.Fatal("observed path31 actor did not retain its original resource family")
	}
	w.Frame, w.ScrollY = 164, 4444
	w.MaximumScrollY, w.VisitedScrollY = w.ScrollY+16, w.ScrollY+16
	w.Player.X, w.Player.Y, w.Player.Inertia = 129, 91, 0
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.cursor = RestartEncounterCursor(w.ScrollY)
	assertFourthOpeningAvoidance(t, w, 31, true)
}

func TestFourthOpeningGuardDoesNotEnterOtherArenasOptional(t *testing.T) {
	w := fourthOpeningSourceFixture(t, 4475, 129, 111, 2)
	for _, fixture := range []struct {
		name   string
		modify func(*World)
	}{
		{"middle-admitted", func(w *World) { w.FourthMiddle = &FourthMiddleGuardian{} }},
		{"final-boundary", func(w *World) { w.ScrollY = 176 }},
		{"ready", func(w *World) { w.Ready = true }},
		{"destroyed", func(w *World) { w.PlayerAlive = false }},
		{"another-level", func(w *World) { w.Level.Number = 2 }},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			copy := *w
			fixture.modify(&copy)
			pilot := PresentationPilot{}
			planned := Input{Motion: MotionInput{Left: true}, Fire: true}
			if got := pilot.forecastOpeningGuard(&copy, planned); got != planned || pilot.forecast.State() != nil {
				t.Fatal("fourth opening eligibility changed another arena or lifecycle state")
			}
		})
	}
}
