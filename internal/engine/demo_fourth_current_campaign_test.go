package engine

import "testing"

// The equipment and READY seed are captured from the current default-intro
// fourth-stage admission. This fixture does not replay the preceding stages.
func capturedCurrentFourthReady(t testing.TB) *World {
	e := Equipment{WeaponLoadout: WeaponLoadout{Primary: WeaponSlot{Item: ItemForwardShot, Tier: 1, MaxTier: 2, Serial: 12}, Mounts: [4]WeaponSlot{{Item: ItemCannon, Serial: 13}}, Rear: WeaponSlot{Item: ItemRearShot, Tier: 1, MaxTier: 2, Serial: 14}}, Shield: 39, Lives: 3, SpeedTier: 2, FirePeriod: 8, FireAdvance: 3, NextWeaponSerial: 14}
	data := playableOriginalWorldData(t, 4)
	data.InitialEquipment = &e
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.Score, w.DisplayScore = 163850, 163850
	w.ContinueCredits = 2
	w.SetRandomState(RandomState{A: 3274314000, B: 1879443244})
	w.ResetBackgroundStars()
	w.PrimeBackgroundStars(2)
	w.Ready = true
	if err := w.Step(Input{Fire: true}); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestCurrentFourthReferenceDepthReachesNativeMiddleMerchantOptional(t *testing.T) {
	w := capturedCurrentFourthReady(t)
	p := PresentationPilot{PALRefreshes: 3, FourthLookahead: 12}
	minimum := 39
	admitted, tail, outer, shortEscape := false, false, false, false
	for pass := 0; pass < 6000; pass++ {
		if w.ShopReady {
			if !admitted || !tail || !outer || !shortEscape || w.Frame != 4692 || w.ScrollY != 2155 || w.Equipment.Shield != 27 || minimum != 27 || w.Money != 750 || w.Score != 181950 || w.RandomState() != (RandomState{A: 1129128063, B: 1283032548}) || !w.FourthMiddle.Defeated || int16(w.FourthMiddle.Parts[4].Health) > 0 || w.PendingExitDrops != 0 || w.LevelFinished {
				t.Fatalf("current middle boundary differs: F%d C%d HP%d min%d cash%d score%d RNG%+v short%v", w.Frame, w.ScrollY, w.Equipment.Shield, minimum, w.Money, w.Score, w.RandomState(), shortEscape)
			}
			t.Log("Current captured fourth entry reaches its real middle merchant with all three ships, two continues, 27 shield and 750 cash.")
			return
		}
		before := ""
		if w.FourthMiddle != nil && w.FourthMiddle.OuterTargets == 0 {
			before = forecastIsolationDigest(w)
		}
		input := p.NormalInput(w)
		if p.FourthLookahead != 12 {
			t.Fatal("extended depth was lost during world admission")
		}
		if core := p.fourthMiddleCore; core != nil && core.plan.count == 1 && !shortEscape {
			shortEscape = true
			plan, at := core.plan, core.at
			if repeated := p.NormalInput(w); repeated != input || core.plan != plan || core.at != at || forecastIsolationDigest(w) != before {
				t.Fatal("short escape resampling changed its input, cache or live source")
			}
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 3 || w.ContinueCredits != 2 || w.Cheats.Enabled() {
			t.Fatalf("current reference route lost an earned reserve: F%d HP%d", w.Frame, w.Equipment.Shield)
		}
		if w.Frame == 1 && (w.ScrollY != 4607 || w.Player.X != 160 || w.Player.Y != 171 || w.RandomState() != (RandomState{A: 4128617536, B: 3288965136})) {
			t.Fatal("current READY fixture differs from actual first combat pass")
		}
		if m := w.FourthMiddle; m != nil {
			if !admitted {
				admitted = true
				if w.Frame != 2777 || w.ScrollY != 2480 || w.Equipment.Shield != 35 || minimum != 35 || m.OuterTargets != 5 || m.Parts[4].Health != 175 {
					t.Fatal("original current guardian admission changed")
				}
			}
			if m.Parts[15].Disabled && !tail {
				tail = true
				if w.Frame != 3209 || w.Equipment.Shield != 35 || m.OuterTargets != 4 {
					t.Fatal("native tail gate changed")
				}
			}
			if m.OuterTargets == 0 && !outer {
				outer = true
				if w.Frame != 4153 || w.Equipment.Shield != 31 || m.Parts[4].Health != 175 {
					t.Fatal("native satellite/core gate changed")
				}
			}
		}
	}
	t.Fatal("current fourth reference route did not reach its middle merchant")
}
