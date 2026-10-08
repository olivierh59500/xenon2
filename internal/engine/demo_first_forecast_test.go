package engine

import "testing"

func firstLevelGuardOriginalScene(t testing.TB, level int) *World {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, level))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 4500, 4516, 4516
	w.Player.X, w.Player.Y = 160, 120
	w.Rewind = NewTerrainRewind(4500, 160, 120)
	w.cursor = EncounterCursor{MovingHighWater: 0, FixedHighWater: 0}
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("original source scene starts in covered terrain")
	}
	return w
}
func TestFirstLevelGuardAvoidsOriginalProjectileWithoutChangingSourceOptional(t *testing.T) {
	w := firstLevelGuardOriginalScene(t, 1)
	w.spawnEnemyShot(160, 80, EnemyShot{Direction: 4, Speed: 6})
	if len(w.Projectiles) != 1 || w.Projectiles[0].Sprite != w.Level.Rules.DefaultEnemyShot {
		t.Fatal("original projectile factory selected another bank")
	}
	before := forecastIsolationDigest(w)
	pilot := PresentationPilot{}
	planned := Input{}
	guarded := pilot.forecastOpeningGuard(w, planned)
	if guarded.Motion == planned.Motion || guarded.Fire != planned.Fire || guarded.Dive != planned.Dive {
		t.Fatal("first-level eligibility did not replace unsafe movement alone")
	}
	held, heldContacts := sixThirdFinalGuardPasses(t, w, planned)
	safe, safeContacts := sixThirdFinalGuardPasses(t, w, guarded)
	if !held.Alive || held.Shield != 35 || heldContacts != 0 || !safe.Alive || safe.Shield != 39 || safeContacts != 0 {
		t.Fatalf("source projectile avoidance differs: held%+v/%d guarded%+v/%d", held, heldContacts, safe, safeContacts)
	}
	if w.Cheats.Enabled() || forecastIsolationDigest(w) != before {
		t.Fatal("first-level guard changed source state, equipment or randomness")
	}
}
func TestFirstLevelGuardPreservesSecondLevelExclusionOptional(t *testing.T) {
	w := firstLevelGuardOriginalScene(t, 2)
	w.spawnEnemyShot(160, 80, EnemyShot{Direction: 4, Speed: 6})
	before := forecastIsolationDigest(w)
	pilot := PresentationPilot{}
	planned := Input{Motion: MotionInput{Right: true}, Fire: true}
	if got := pilot.forecastOpeningGuard(w, planned); got != planned || pilot.forecast.State() != nil {
		t.Fatal("first-level experiment changed the second-level beam or guard behavior")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("excluded second-level guard changed source state")
	}
}
func TestFirstLevelGuardPreservesLifecycleAndOtherLevelScopeOptional(t *testing.T) {
	original := firstLevelGuardOriginalScene(t, 1)
	for _, state := range []string{"ready", "destroyed", "level-five"} {
		copy := *original
		switch state {
		case "ready":
			copy.Ready = true
		case "destroyed":
			copy.PlayerAlive = false
		case "level-five":
			copy.Level.Number = 5
		}
		before := forecastIsolationDigest(&copy)
		pilot := PresentationPilot{}
		planned := Input{Motion: MotionInput{Left: true}, Fire: true, Dive: true}
		if got := pilot.forecastOpeningGuard(&copy, planned); got != planned || pilot.forecast.State() != nil {
			t.Fatalf("first-level experiment entered excluded scope%s", state)
		}
		if forecastIsolationDigest(&copy) != before {
			t.Fatal("excluded lifecycle probe changed source state")
		}
	}
}
