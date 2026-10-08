package engine

import "testing"

// fourthMiddleBoundaryPoint finds a point whose first native moving-list
// callback belongs to this target. The companion may forward a core hit.
func fourthMiddleBoundaryPoint(w *World, index int) (int, int, bool) {
	target := w.fourthMiddleActors[index]
	if target == nil || !target.Active || target.Collision.Empty() {
		return 0, 0, false
	}
	var storage [ActorPoolCapacity]*WorldActor
	actors := w.orderedMovingActors(&storage)
	r := target.Collision
	for y := max(0, r.Top+1); y < min(192, r.Bottom); y++ {
		for x := max(0, r.Left); x <= min(319, r.Right); x++ {
			point := CollisionRect{Left: x, Right: x, Top: y, Bottom: y}
			for _, actor := range actors {
				if !actor.Active || actor.ActorList != "moving" || !actor.Collision.Intersects(point) {
					continue
				}
				if actor == target || index == 4 && actor == w.fourthMiddleActors[5] {
					return x, y, true
				}
				break
			}
		}
	}
	return 0, 0, false
}

// This explicitly arranged original-resource arena is not a campaign victory.
// Native-factory in-flight bullets retain legal damage and run every actual
// collision/retirement callback; no guardian HP, death or exit flag is assigned.
func TestOriginalFourthMiddleProjectilesOpenRealMerchantAndResumeOptional(t *testing.T) {
	e := NewEquipment()
	for _, item := range []Item{ItemPowerup, ItemPowerup, ItemSpeedup, ItemSpeedup} {
		e.ApplyItem(item)
	}
	data := playableOriginalWorldData(t, 4)
	data.InitialEquipment = &e
	s, err := NewSession(data, 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := s.ActiveWorld()
	// Cross the original fixed selector. Its own constructor performs the
	// native checkpoint restoration and creates all twenty physical parts.
	w.Ready = false
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2672, 2672, 2672
	w.cursor = EncounterCursor{MovingHighWater: 2673, FixedHighWater: 2673}
	if _, err := s.Advance(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.FourthMiddle == nil || w.FourthMiddle.OuterTargets != 5 || w.FourthMiddle.Parts[4].Health != 175 || w.LevelFinished {
		t.Fatal("original selector did not admit the native locked middle core")
	}
	w.Player.X, w.Player.Y, w.Player.Inertia = 24, 176, 0
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("isolated native arena's edge pose is covered")
	}
	credits, lives := w.ContinueCredits, w.Equipment.Lives
	var next WorldForecast
	lockedHit := false
	for pass := 0; pass < 700 && !w.FourthMiddle.Defeated; pass++ {
		// The native upper defenses enter before the lower core. Reverse the
		// released camera within its2480 bound to expose that core again.
		input := Input{Motion: MotionInput{Down: w.FourthMiddle.OuterTargets == 0}}
		if err := next.Load(w); err != nil {
			t.Fatal(err)
		}
		for range 3 {
			next.AdvancePALTick()
		}
		if _, err := next.Advance(input); err != nil {
			t.Fatal(err)
		}
		future := next.State()
		beforeCore, beforeOuter := w.FourthMiddle.Parts[4].Health, w.FourthMiddle.OuterTargets
		probeLocked := false
		var lockedProjectile *WorldSmallShot
		if !lockedHit && beforeOuter == 5 {
			if x, y, found := fourthMiddleBoundaryPoint(future, 4); found {
				fifthBoundaryShot(t, w, x, y)
				probeLocked = true
				lockedProjectile = w.SmallShots[len(w.SmallShots)-1]
			}
		} else if beforeOuter != 0 {
			for _, index := range []int{15, 16, 17, 18, 19} {
				if future.FourthMiddle.Parts[index].Disabled {
					continue
				}
				if x, y, found := fourthMiddleBoundaryPoint(future, index); found {
					fifthBoundaryShot(t, w, x, y)
				}
			}
		} else if x, y, found := fourthMiddleBoundaryPoint(future, 4); found {
			fifthBoundaryShot(t, w, x, y)
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if _, err := s.Advance(input); err != nil {
			t.Fatal(err)
		}
		if probeLocked {
			if lockedProjectile.Active || w.FourthMiddle.Parts[4].Health != 175 || w.FourthMiddle.OuterTargets != 5 || w.Score != 0 || w.PendingExitDrops != 0 {
				t.Fatal("real locked-core projectile bypassed its original five-target gate")
			}
			lockedHit = true
		}
		if beforeOuter != 0 && w.FourthMiddle.Parts[4].Health != beforeCore {
			t.Fatal("core health changed before all five native targets were destroyed")
		}
		if !w.PlayerAlive || w.Equipment.Lives != lives {
			t.Fatalf("arranged projectile boundary lost its living ship at pass%d camera%d shield%d", pass, w.ScrollY, w.Equipment.Shield)
		}
	}
	if !lockedHit || !w.FourthMiddle.Defeated || w.FourthMiddle.OuterTargets != 0 || w.Score != 4200 || w.PendingExitDrops != 10 || len(w.Collectibles) != 10 || w.ShopReady || w.ExitReady || w.LevelFinished {
		t.Fatalf("native lethal core projectile lost the same-stage reward boundary: locked%v defeated%v outer%d score%d drops%d shop%v exit%v finished%v", lockedHit, w.FourthMiddle.Defeated, w.FourthMiddle.OuterTargets, w.Score, w.PendingExitDrops, w.ShopReady, w.ExitReady, w.LevelFinished)
	}
	if w.MinimumScrollY != 0 || w.PreviousScrollY != 2208 || w.RenderScrollY != 2208 || w.ScrollY != 2208-w.ScrollDelta || w.MaximumScrollY != 2208 || w.BaseScrollStep != 1 {
		t.Fatal("real defeat did not publish the source forced-scroll boundary")
	}
	for row := 143; row < 159; row++ {
		for _, tile := range w.Coverage.Map[row*20 : (row+1)*20] {
			if tile != 0 {
				t.Fatal("native middle defeat did not clear its original terrain corridor")
			}
		}
	}
	for _, coin := range w.Collectibles {
		if !coin.Active || coin.Cash != 50 && coin.Cash != 100 {
			t.Fatal("original reward factory did not preserve ten ordinary cash objects")
		}
	}
	for pass := 0; pass < 240 && !w.ShopReady; pass++ {
		// Native Down holds the released arena's rear bound while falling
		// coins converge/expire. Collection and gate consumption remain real.
		input := Input{Motion: MotionInput{Down: true, Right: w.Player.X < 155, Left: w.Player.X > 165}}
		for range 3 {
			w.AdvancePALTick()
		}
		if _, err := s.Advance(input); err != nil {
			t.Fatal(err)
		}
	}
	if !w.ShopReady || w.ExitReady || w.LevelFinished || w.PendingExitDrops != 0 || !w.PlayerAlive || w.Money < 0 || w.Money > 750 || w.ContinueCredits != credits {
		t.Fatal("natural reward callbacks failed to open only the same-stage merchant")
	}
	for _, actor := range w.fourthMiddleActors {
		slot := w.Pool.Slot(actor.Binding.Slot)
		if actor.Active || slot.allocated && slot.EntityID == actor.ID {
			t.Fatal("middle defeat retained a living guardian physical slot at the merchant boundary")
		}
	}
	camera, frame, player, equipment, random, score, money := w.ScrollY, w.Frame, w.Player, w.Equipment, w.RandomState(), w.Score, w.Money
	w.ResumeShop()
	if s.ActiveWorld() != w || w.Ready || w.ShopReady || w.ExitReady || w.LevelFinished || w.ScrollY != camera || w.Frame != frame || w.Player != player || w.Equipment != equipment || w.RandomState() != random || w.Score != score || w.Money != money || w.Checkpoint.Loadout != w.Equipment.WeaponLoadout {
		t.Fatal("middle merchant return restarted or replaced the native living world")
	}
	for range 3 {
		w.AdvancePALTick()
	}
	if _, err := s.Advance(Input{}); err != nil {
		t.Fatal(err)
	}
	if s.ActiveWorld() != w || w.Level.Number != 4 || w.Frame != frame+1 || w.ScrollY != camera-1 || !w.PlayerAlive || w.LevelFinished || s.Difficulty != 1 || w.ContinueCredits != credits {
		t.Fatal("ordinary post-merchant movement did not continue the same fourth stage")
	}
}
