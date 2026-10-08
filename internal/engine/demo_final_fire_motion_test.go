package engine

import "testing"

// Find the real source-route mismatch rather than substituting a synthetic
// firing target. A pure final-motion veto must retain the once-evaluated burst.
func TestExpertGuardFinalFireKeepsOneBurstDecisionOnOriginalRouteOptional(t *testing.T) {
	s, err := NewSession(playableOriginalWorldData(t, 1), 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	p := PresentationPilot{PALRefreshes: 3}
	for pass := 0; pass < 2000; pass++ {
		w := s.ActiveWorld()
		if w.GameOver || w.ShopReady || w.LevelFinished {
			break
		}
		for range 3 {
			w.AdvancePALTick()
		}
		input := p.NormalInput(w)
		opportunity := presentationShotOpportunityForMotion(w, input.Motion)
		if !w.Ready && input.Fire && !opportunity {
			t.Fatalf("guard retained pre-motion Fire without a final intercept at pass%d frame%d camera%d ship%d,%d motion%+v", pass, w.Frame, w.ScrollY, w.Player.X, w.Player.Y, input.Motion)
		}
		if !w.Ready && !w.blockedFireUntilRelease && w.Dive.Phase == 0 && !input.Fire && !opportunity && p.burstUntil > w.Frame && p.restUntil <= w.Frame {
			burst, rest := p.burstUntil, p.restUntil
			before := forecastIsolationDigest(w)
			for range 3 {
				if again := p.NormalInput(w); again != input || p.burstUntil != burst || p.restUntil != rest {
					t.Fatalf("final-motion veto re-evaluated the held burst: input%+v/%+v burst%d/%d rest%d/%d", input, again, burst, p.burstUntil, rest, p.restUntil)
				}
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("final-motion firing check mutated the live source world")
			}
			t.Logf("Real firing veto: pass%d frame%d camera%d ship%d,%d motion%+v burstUntil%d restUntil%d", pass, w.Frame, w.ScrollY, w.Player.X, w.Player.Y, input.Motion, burst, rest)
			return
		}
		if _, err := s.Advance(input); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("original route did not exercise a guarded-motion firing veto with an active burst")
}

func TestExpertGuardFinalFirePreservesBlockedAndDiveOptional(t *testing.T) {
	for _, mode := range []string{"ready-release", "dive"} {
		t.Run(mode, func(t *testing.T) {
			w := firstLevelGuardOriginalScene(t, 1)
			w.spawnEnemyShot(160, 80, EnemyShot{Direction: 4, Speed: 6})
			if mode == "ready-release" {
				w.blockedFireUntilRelease = true
			} else {
				w.Dive.Phase = 1
			}
			before := forecastIsolationDigest(w)
			p := PresentationPilot{PALRefreshes: 3}
			if input := p.NormalInput(w); input.Fire {
				t.Fatalf("guard re-enabled firing during %s: %+v", mode, input)
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("guarded lifecycle check changed source state")
			}
		})
	}
}
