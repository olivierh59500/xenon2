package engine

import "testing"

// This is an isolated original-resource scene, not a campaign checkpoint. The
// shot is one pass before its native turn, with its previous rightward velocity.
func fifthGuidedMissileScene(t testing.TB, timer int) *World {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, 5))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	w.Player.X, w.Player.Y = 160, 176
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	event := FixedSpriteEvents{ShotMode: "animated-aiming-projectile", ShotCount: 1, ShotX: 132, ShotY: 143, ShotDelay: timer, ShotDirections: [3]int{2}}
	if !w.spawnSpecializedFixedShot(event) || len(w.Actors) != 1 {
		t.Fatal("original missile constructor was not admitted")
	}
	a := w.Actors[0]
	a.PreviousX, a.PreviousY = a.X-float64(homingX[2]), a.Y-1
	if a.fixedAiming == nil || a.part.ResourceTag != 228 || a.Health != 6 || a.Sprite != "fixed-image-004" || a.Collision.Empty() || w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("fixture lost original missile data or clear terrain")
	}
	return w
}

func directFifthMissileLosses(t testing.TB, w *World, motion MotionInput, horizon, pal int) (damage [8]int) {
	t.Helper()
	shield := w.Equipment.Shield
	for pass := range horizon {
		for range pal {
			w.AdvancePALTick()
		}
		if err := w.Step(Input{Motion: motion}); err != nil {
			t.Fatal(err)
		}
		damage[pass] = max(0, shield-w.Equipment.Shield)
		shield = w.Equipment.Shield
		if forecastBoundary(w) != ForecastRunning {
			break
		}
	}
	return damage
}

func TestFifthGuidedRiskMatchesDirectOriginalCallbacksOptional(t *testing.T) {
	w := fifthGuidedMissileScene(t, 7)
	before := forecastIsolationDigest(w)
	var risk demoFifthNativeRisk
	for _, motion := range demoDirections {
		for _, pal := range []int{2, 3} {
			got, ok := risk.losses(w, motion, 6, pal)
			want := directFifthMissileLosses(t, fifthGuidedMissileScene(t, 7), motion, 6, pal)
			if !ok || got != want {
				t.Fatalf("motion%+v PAL%d: got%v supported%v want%v", motion, pal, got, ok, want)
			}
		}
	}
	// A lethal contact stops at the death boundary rather than predicting a
	// checkpoint restart or spending a life in the isolated future.
	w.Equipment.Shield = 3
	independent := fifthGuidedMissileScene(t, 7)
	independent.Equipment.Shield = 3
	got, ok := risk.losses(w, MotionInput{}, 6, 3)
	want := directFifthMissileLosses(t, independent, MotionInput{}, 6, 3)
	if !ok || got != want || risk.forecast.State().PlayerAlive || !w.PlayerAlive || w.Equipment.Shield != 3 {
		t.Fatalf("native death boundary differs or changed the source: %v want%v", got, want)
	}
	w.Equipment.Shield = 39
	if forecastIsolationDigest(w) != before {
		t.Fatal("candidate forecasts changed live missile, pool, terrain, weapons or randomness")
	}
}

func TestFifthGuidedRiskCatchesTurnAndDropsExpiredMissileOptional(t *testing.T) {
	w := fifthGuidedMissileScene(t, 7)
	var native demoFifthNativeRisk
	damage, ok := native.losses(w, MotionInput{}, 6, 3)
	total := 0
	for _, debit := range damage {
		total += debit
	}
	if !ok || total != 8 {
		t.Fatalf("native pre-movement contact after the eighth-pass turn: %v supported%v", damage, ok)
	}
	old, _ := presentationMotionScore(w, MotionInput{}, 160, 176)
	corrected, _ := presentationMotionScoreWithRisk(w, MotionInput{}, 160, 176, &native, 3)
	if old != 0 || corrected < 100000 {
		t.Fatalf("turning missile regression: oldrisk%g corrected%g", old, corrected)
	}
	// Reload a different world into the same storage: the native lifetime check
	// removes this shot before it could make the turn or reach the ship.
	expired := fifthGuidedMissileScene(t, 79)
	damage, ok = native.losses(expired, MotionInput{}, 6, 3)
	if !ok || damage != ([8]int{}) {
		t.Fatalf("forecast retained a prior-world turn or ignored expiry: %v", damage)
	}
	if expired.Actors[0].fixedAiming.Removed || expired.Actors[0].fixedAiming.Timer != 79 {
		t.Fatal("forecast expired the live missile")
	}
}

func TestFifthGuidedRiskScopeAndPilotIntegrationOptional(t *testing.T) {
	w := fifthGuidedMissileScene(t, 7)
	before := forecastIsolationDigest(w)
	p := DemoPilot{Config: DemoPilotConfig{TargetX: 160, TargetY: 176, DisableDive: true}}
	input := p.NormalInput(w)
	if p.fifthRisk == nil || input.Motion == (MotionInput{}) || forecastIsolationDigest(w) != before {
		t.Fatal("ordinary pilot did not use isolated native guided-missile risk")
	}
	presentation := PresentationPilot{goal: presentationGoal{x: 160, y: 176}, planner: p}
	motion := presentation.tacticalMotion(w, MotionInput{})
	for _, selected := range []MotionInput{input.Motion, motion} {
		damage, ok := p.fifthRisk.losses(w, selected, 6, 3)
		if !ok || damage != ([8]int{}) {
			t.Fatalf("selected motion still takes the avoidable guided hit: %+v %v", selected, damage)
		}
	}
	for _, change := range []func(*World){
		func(q *World) { q.Level.Number = 4 },
		func(q *World) { q.Ready = true },
		func(q *World) { q.GameOver = true },
		func(q *World) { q.PlayerAlive = false },
		func(q *World) { q.ShopReady = true },
		func(q *World) { q.LevelFinished = true },
		func(q *World) { q.ScreenClearFrames = 1 },
		func(q *World) { q.stepContinuation.active = true },
		func(q *World) { q.Dive.Phase = 1 },
		func(q *World) { q.Actors = nil },
	} {
		q := *w
		change(&q)
		if _, supported := p.fifthRisk.losses(&q, MotionInput{}, 6, 3); supported {
			t.Fatal("unsupported lifecycle or another stage admitted fifth missile prediction")
		}
	}
	for _, horizon := range []int{0, 9} {
		if _, supported := p.fifthRisk.losses(w, MotionInput{}, horizon, 3); supported {
			t.Fatal("invalid prediction horizon admitted")
		}
	}
}

func BenchmarkFifthGuidedRiskNineDirectionsOriginal(b *testing.B) {
	w := fifthGuidedMissileScene(b, 7)
	var risk demoFifthNativeRisk
	for _, motion := range demoDirections {
		risk.losses(w, motion, 6, 3)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for _, motion := range demoDirections {
			if _, ok := risk.losses(w, motion, 6, 3); !ok {
				b.Fatal("source callback forecast failed")
			}
		}
	}
}
