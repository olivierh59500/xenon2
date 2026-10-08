package engine

import "testing"

// The original diagonal-shot fixture makes an ordinary changing fallback
// necessary. Its source factory, art, motion fractions and callbacks remain live.
func fourthBranchSourcePlan(t testing.TB) (*World, *PresentationPilot, Input) {
	t.Helper()
	w := fourthOpeningSourceFixture(t, 4475, 129, 111, 2)
	w.Frame = 133
	dx := int32(directionX[5]) * 6 * 4
	dy := int32(directionY[5])*6*4 + 1<<16
	x, y := int32(7901968)-4*dx, int32(7105776)-4*dy
	w.spawnEnemyShot(int(x>>16), int(y>>16), EnemyShot{Direction: 5, Speed: 6})
	shot := w.Projectiles[0]
	shot.Motion.X, shot.Motion.Y = x, y
	shot.X, shot.Y = float64(x>>16), float64(y>>16)
	p := &PresentationPilot{PALRefreshes: 3, world: w, frame: w.Frame}
	before := forecastIsolationDigest(w)
	input := p.forecastOpeningGuard(w, Input{})
	if input.Motion == (MotionInput{}) || p.fourthBranch == nil || p.fourthBranch.count != 6 {
		t.Fatal("source safe changing fallback was not retained")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("branch capture changed live source")
	}
	return w, p, input
}

func TestFourthBranchRetainsOriginalInputsAndSameFrameIsIdempotentOptional(t *testing.T) {
	w, p, input := fourthBranchSourcePlan(t)
	for pass := 0; pass < 6; pass++ {
		before := forecastIsolationDigest(w)
		got := p.NormalInput(w)
		at := p.fourthBranch.at
		if got != input || p.NormalInput(w) != got || p.fourthBranch.at != at {
			t.Fatalf("retained source input repeated or changed at%d", pass)
		}
		if forecastIsolationDigest(w) != before {
			t.Fatal("remaining-callback verification changed live source")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(got); err != nil {
			t.Fatal(err)
		}
		if w.Equipment.Shield != 39 || w.Rewind.Timer != 0 || retainedGuardStateKey(w) != p.fourthBranch.before[pass+1] {
			t.Fatal("retained command lost its exact safe source successor")
		}
	}
	if _, ok := p.continueFourthOpeningBranch(w); ok {
		t.Fatal("completed six-command branch extended its horizon")
	}
}

func TestFourthBranchReforecastRejectsChangedExistingProjectileOptional(t *testing.T) {
	w, p, input := fourthBranchSourcePlan(t)
	key := retainedGuardStateKey(w)
	shot := w.Projectiles[0]
	shot.Motion.X, shot.Motion.Y = int32(w.Player.X)<<16, int32(w.Player.Y-7)<<16
	shot.Motion.Direction, shot.Motion.Speed = 4, 6
	shot.X, shot.Y = float64(w.Player.X), float64(w.Player.Y-7)
	if retainedGuardStateKey(w) != key {
		t.Fatal("entity-only mutation unexpectedly changed the compact key")
	}
	var actual WorldForecast
	if err := actual.Load(w); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		actual.AdvancePALTick()
	}
	result, err := actual.Advance(input)
	if err != nil || result.Shield >= w.Equipment.Shield {
		t.Fatal("changed native projectile did not threaten the retained input")
	}
	before := forecastIsolationDigest(w)
	if _, ok := p.continueFourthOpeningBranch(w); ok || p.fourthBranch.count != 0 {
		t.Fatal("remaining source callback replay accepted the changed projectile")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("rejected branch changed live projectile state")
	}
}

func TestFourthBranchRejectsNewNativeShotAndTerrainPatchOptional(t *testing.T) {
	t.Run("new native projectile", func(t *testing.T) {
		w, p, _ := fourthBranchSourcePlan(t)
		frame, player, gear, random := w.Frame, w.Player, w.Equipment, w.RandomState()
		w.spawnEnemyShot(w.Player.X, w.Player.Y-7, EnemyShot{Direction: 4, Speed: 6})
		if w.Frame != frame || w.Player != player || w.Equipment != gear || w.RandomState() != random {
			t.Fatal("native projectile factory changed scalar player state")
		}
		if _, ok := p.continueFourthOpeningBranch(w); ok {
			t.Fatal("branch survived a newly introduced native projectile")
		}
	})
	t.Run("original opaque terrain patch", func(t *testing.T) {
		w, p, _ := fourthBranchSourcePlan(t)
		solid, found := uint16(0), false
		for id, rows := range w.Coverage.coverage {
			opaque := true
			for _, row := range rows {
				opaque = opaque && row == 0xffff
			}
			if opaque {
				solid, found = uint16(id), true
				break
			}
		}
		if !found {
			t.Fatal("original terrain has no opaque tile for obstruction fixture")
		}
		column, row := w.Player.X/16, (w.Player.Y+w.ScrollY)/16
		for y := row - 1; y <= row+1; y++ {
			for x := column - 1; x <= column+1; x++ {
				w.setSecondMapCell(x, y, solid)
			}
		}
		if !w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
			t.Fatal("native terrain patch did not obstruct the actual stencil")
		}
		if _, ok := p.continueFourthOpeningBranch(w); ok {
			t.Fatal("retained branch ignored changed native terrain")
		}
	})
}

func TestFourthBranchDropsOtherOwnersAndLifecycleOptional(t *testing.T) {
	for _, mode := range []string{"level", "world", "middle", "ready", "dead", "shop", "exit", "finished", "drops", "strobe", "partial", "PAL", "gear"} {
		t.Run(mode, func(t *testing.T) {
			w, p, _ := fourthBranchSourcePlan(t)
			var other WorldForecast
			switch mode {
			case "level":
				w.Level.Number = 3
			case "world":
				if err := other.Load(w); err != nil {
					t.Fatal(err)
				}
				w = other.State()
			case "middle":
				w.FourthMiddle = &FourthMiddleGuardian{}
			case "ready":
				w.Ready = true
			case "dead":
				w.PlayerAlive = false
			case "shop":
				w.ShopReady = true
			case "exit":
				w.ExitReady = true
			case "finished":
				w.LevelFinished = true
			case "drops":
				w.PendingExitDrops = 1
			case "strobe":
				w.ScreenClearFrames = 1
			case "partial":
				w.stepContinuation.active = true
			case "PAL":
				p.PALRefreshes = 1
			case "gear":
				w.Equipment.ApplyItem(ItemProtection)
			}
			if _, ok := p.continueFourthOpeningBranch(w); ok || p.fourthBranch.count != 0 {
				t.Fatal("branch retained an invalid owner or lifecycle")
			}
		})
	}
}

func TestFourthBranchDoesNotCaptureUnsafeOrDiveInputOptional(t *testing.T) {
	w, _, safe := fourthBranchSourcePlan(t)
	for _, input := range []Input{{}, {Motion: safe.Motion, Dive: true}} {
		p := PresentationPilot{PALRefreshes: 3}
		p.captureFourthOpeningBranch(w, input)
		if p.fourthBranch != nil && p.fourthBranch.count != 0 {
			t.Fatal("unsafe or one-shot Dive input installed a retained branch")
		}
	}
}
