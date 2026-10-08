package engine

import "testing"

func fourthPostMiddleBeamScene(t testing.TB) *World {
	t.Helper()
	w := fourthOpeningSourceFixture(t, 1700, 204, 106, 2)
	// Explicit captured callback scene, not an earned campaign victory. The
	// earlier merchant is a scope condition; only the original beam is isolated.
	w.FourthMiddle = &FourthMiddleGuardian{Defeated: true}
	// The native core-defeat event releases the initial2176 lower camera gate.
	w.MinimumScrollY = 0
	w.Frame = 5003
	w.cursor = RestartEncounterCursor(0)
	found := false
	for _, record := range w.Level.Encounters.Fixed {
		if record.EnemyKind != 5 || record.X != 248 || record.Y != 1800 || record.Variant != 1 {
			continue
		}
		w.spawnFixed(record)
		found = true
		break
	}
	if !found || len(w.Actors) != 1 {
		t.Fatal("original right beam encounter missing")
	}
	a := w.Actors[0]
	if a.fixedKind.Behavior != "extending-beam" || a.fixedKind.ContactDamage != 6 || a.part.ResourceTag != 272 {
		t.Fatal("original right beam callback changed")
	}
	// Six callbacks before the captured phase2 hit: overflow at the fifth
	// callback starts phase1. Actor coordinates precede the camera's last step.
	a.fixedState.FireAccumulator = 246
	a.fixedState.Y = 91
	a.Y, a.PreviousY = 91, 91
	return w
}

func TestFourthPostMiddleGuardAvoidsOriginalExtendingBeamOptional(t *testing.T) {
	w := fourthPostMiddleBeamScene(t)
	before := forecastIsolationDigest(w)
	planned := Input{}
	p := PresentationPilot{PALRefreshes: 3}
	guarded := p.forecastOpeningGuard(w, planned)
	held, _ := sixThirdFinalGuardPasses(t, w, planned)
	safe, _ := sixThirdFinalGuardPasses(t, w, guarded)
	if held.Shield != 33 || !held.Alive || safe.Shield != 39 || !safe.Alive || guarded.Motion == planned.Motion || guarded.Fire != planned.Fire || guarded.Dive != planned.Dive {
		t.Fatalf("native right beam avoidance differs: held%+v safe%+v planned%+v guarded%+v", held, safe, planned, guarded)
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("post-middle guard changed its source scene")
	}
}

func TestFourthPostMiddleGuardDeclinesOtherStagesAndFinalOptional(t *testing.T) {
	w := fourthPostMiddleBeamScene(t)
	for _, name := range []string{"stage2", "final-camera", "ready", "dead"} {
		t.Run(name, func(t *testing.T) {
			var clone WorldForecast
			if err := clone.Load(w); err != nil {
				t.Fatal(err)
			}
			q := clone.State()
			switch name {
			case "stage2":
				q.Level.Number = 2
			case "final-camera":
				q.ScrollY = 176
			case "ready":
				q.Ready = true
			case "dead":
				q.PlayerAlive = false
			}
			before := forecastIsolationDigest(q)
			planned := Input{}
			if got := (&PresentationPilot{PALRefreshes: 3}).forecastOpeningGuard(q, planned); got != planned || forecastIsolationDigest(q) != before {
				t.Fatal("post-middle scope changed unrelated ordinary input")
			}
		})
	}
}
