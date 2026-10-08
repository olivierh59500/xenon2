package engine

import "testing"

func TestWorldForecastPointObserverIsolationOptional(t *testing.T) {
	for level := 1; level <= 5; level++ {
		live := forecastOriginalScene(t, level, true)
		before := forecastIsolationDigest(live)
		var observed, plain WorldForecast
		if err := observed.Load(live); err != nil {
			t.Fatal(err)
		}
		if err := plain.Load(live); err != nil {
			t.Fatal(err)
		}
		queries := 0
		for pass := 0; pass < 3; pass++ {
			input := Input{Fire: true}
			for tick := 0; tick < 3; tick++ {
				observed.AdvancePALTick()
				plain.AdvancePALTick()
			}
			if _, err := observed.AdvanceObserved(input, func(event WeaponPointImpact) {
				queries++
				if event.ProjectileID <= live.nextActorID || event.Kind != "small-shot" || event.OwnerSlot != 0 || event.Damage != 1 {
					t.Fatalf("level%d: unexpected primary point query %+v", level, event)
				}
				// Loading during observation must not transfer the external callback.
				var nested WorldForecast
				if err := nested.Load(observed.State()); err != nil || nested.State().Weapons.context.ObservePointImpact != nil {
					t.Fatal("loading another forecast retained its source observer")
				}
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := plain.Advance(input); err != nil {
				t.Fatal(err)
			}
			if observed.State().Weapons.context.ObservePointImpact != nil || forecastDigest(observed.State()) != forecastDigest(plain.State()) {
				t.Fatalf("level%d: observing changed native simulation or retained the callback", level)
			}
		}
		if queries == 0 || forecastIsolationDigest(live) != before {
			t.Fatalf("level%d: queries%d or live world changed", level, queries)
		}
	}
}

func TestWorldForecastPointObserverKeepsPrimaryDoubleRearAndSideIdentityOptional(t *testing.T) {
	for _, auxiliary := range []Item{ItemRearShot, ItemSideShot} {
		t.Run(map[Item]string{ItemRearShot: "rear", ItemSideShot: "side"}[auxiliary], func(t *testing.T) {
			w := forecastOriginalScene(t, 5, true)
			w.Player.X, w.Player.Y = 160, 100
			for _, item := range []Item{ItemDoubleShot, auxiliary} {
				if !w.Equipment.ApplyItem(item) {
					t.Fatal("ordinary equipment installation failed")
				}
			}
			var forecast WorldForecast
			if err := forecast.Load(w); err != nil {
				t.Fatal(err)
			}
			var events []WeaponPointImpact
			if _, err := forecast.AdvanceObserved(Input{Fire: true}, func(event WeaponPointImpact) {
				events = append(events, event)
			}); err != nil {
				t.Fatal(err)
			}
			var expected []WeaponPointImpact
			for _, owner := range []int{0, 5, 6} {
				if w.Equipment.slots()[owner].Item == ItemNone {
					continue
				}
				shots, err := AppendSmallWeaponShots(nil, *w.Equipment.slots()[owner], w.Player.X, w.Player.Y)
				if err != nil {
					t.Fatal(err)
				}
				for index := len(shots) - 1; index >= 0; index-- {
					shot := shots[index]
					if !shot.Advance(false) {
						t.Fatal("source small-shot fixture exited before its query")
					}
					expected = append(expected, WeaponPointImpact{OwnerSlot: owner, Kind: "small-shot", X: shot.X, Y: shot.Y, Damage: shot.Damage})
				}
			}
			if len(events) != len(expected) {
				t.Fatalf("queries%+v expected%+v", events, expected)
			}
			for index, event := range events {
				if event.ProjectileID <= w.nextActorID {
					t.Fatal("new equipment emission lost its actual projectile identity")
				}
				event.ProjectileID = 0
				if event != expected[index] {
					t.Fatalf("query%d=%+v expected%+v", index, event, expected[index])
				}
			}
		})
	}
}
