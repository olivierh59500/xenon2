package engine

import "testing"

// fifthMiddleBoundaryPoint preserves physical first-hit order, around the
// narrow, independently vulnerable core and its noncollidable body artwork.
func fifthMiddleBoundaryPoint(w *World, index int) (int, int, bool) {
	target := w.fifthMiddleActors[index]
	if target == nil || !target.Active || target.Collision.Empty() {
		return 0, 0, false
	}
	var storage [ActorPoolCapacity]*WorldActor
	actors := w.orderedMovingActors(&storage)
	r := target.Collision
	for y := max(0, r.Top); y <= min(191, r.Bottom); y++ {
		for x := max(0, r.Left); x <= min(319, r.Right); x++ {
			point := CollisionRect{Left: x, Right: x, Top: y, Bottom: y}
			for _, actor := range actors {
				if !actor.Active || actor.ActorList != "moving" || !actor.Collision.Intersects(point) {
					continue
				}
				if actor == target {
					return x, y, true
				}
				break
			}
		}
	}
	return 0, 0, false
}

// This original-resource source-arena fixture arranges an ordinary loadout and
// in-flight native-factory bullets. It is not an earned stage/campaign victory.
// All ten parts retain their health, collision prefixes, slots and callbacks.
func TestOriginalFifthMiddleProjectilesDrainRewardsAndResumeOptional(t *testing.T) {
	e := NewEquipment()
	for _, item := range []Item{ItemPowerup, ItemPowerup, ItemSpeedup, ItemSpeedup, ItemAutofire, ItemAutofire} {
		e.ApplyItem(item)
	}
	data := playableOriginalWorldData(t, 5)
	data.InitialEquipment = &e
	s, err := NewSession(data, 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := s.ActiveWorld()
	w.Ready = false
	// Arrange the original body's on-screen world pose. The preceding forest
	// approach is outside this damage/reward boundary fixture.
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2176, 2336, 2336
	w.cursor = EncounterCursor{MovingHighWater: 2177, FixedHighWater: 2176}
	for _, record := range w.Level.Encounters.Fixed {
		if record.EnemyKind == 5 {
			w.spawnFixed(record)
			break
		}
	}
	if w.FifthMiddle == nil || len(w.fifthMiddleActors) != 10 || w.FifthMiddle.Parts[5].Health != 200 {
		t.Fatal("original selector did not construct the ten-part middle guardian")
	}
	for index := 1; index <= 4; index++ {
		if w.FifthMiddle.Parts[index].Health != 40 {
			t.Fatal("source mount did not retain its original forty HP")
		}
	}
	clear := false
	for y := 176; y >= 16 && !clear; y-- {
		for x := 14; x <= 304; x++ {
			if !w.Coverage.Touches(x, y, w.ScrollY, *w.Level.PlayerStencil) {
				w.Player.X, w.Player.Y, clear = x, y, true
				break
			}
		}
	}
	if !clear {
		t.Fatal("original middle arena has no clear native ship pose")
	}
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	credits, lives := w.ContinueCredits, w.Equipment.Lives
	bodyMissChecked, earlyCoreHit := false, false
	var next WorldForecast
	for pass := 0; pass < 700 && !w.FifthMiddle.Defeated; pass++ {
		outerAlive := false
		for index := 1; index <= 4; index++ {
			outerAlive = outerAlive || !w.FifthMiddle.Parts[index].Destroyed
		}
		core := w.fifthMiddleActors[5]
		input := Input{Motion: MotionInput{Down: !outerAlive && core.Collision.Bottom >= 180}}
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
		beforeCore, beforeScore := w.FifthMiddle.Parts[5].Health, w.Score
		probeBodyMiss, probeCore := false, false
		var bodyMissShot *WorldSmallShot
		bodyMiss := false
		if !bodyMissChecked && !outerAlive {
			bounds := future.fifthMiddleActors[5].Collision
			x, y := bounds.Left-1, (bounds.Top+bounds.Bottom)/2
			if x >= 0 && x < 320 && y >= 0 && y < 192 {
				if hit, _ := presentationFirstPointImpact(future, x, y); !hit {
					fifthBoundaryShot(t, w, x, y)
					bodyMissShot, probeBodyMiss, bodyMiss = w.SmallShots[len(w.SmallShots)-1], true, true
				}
			}
		}
		if !bodyMiss {
			for index := 1; index <= 4; index++ {
				if !w.FifthMiddle.Parts[index].Destroyed {
					if x, y, found := fifthMiddleBoundaryPoint(future, index); found {
						fifthBoundaryShot(t, w, x, y)
					}
				}
			}
			if !earlyCoreHit && outerAlive || bodyMissChecked && !outerAlive {
				if x, y, found := fifthMiddleBoundaryPoint(future, 5); found {
					shots := 1
					if bodyMissChecked && !outerAlive {
						// Arrange the remaining already-emitted legal bullets in one
						// source callback phase. This isolates the reward boundary
						// from campaign navigation and preserves every HP debit.
						shots = (beforeCore + 2) / 3
					}
					for range shots {
						fifthBoundaryShot(t, w, x, y)
					}
					probeCore = !earlyCoreHit
				}
			}
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if _, err := s.Advance(input); err != nil {
			t.Fatal(err)
		}
		if probeBodyMiss {
			if !bodyMissShot.Active || w.FifthMiddle.Parts[5].Health != beforeCore || w.Score != beforeScore || w.FifthMiddle.Defeated || w.fifthMiddleActors[0].Flash {
				t.Fatal("shot beside the narrow core did not pass harmlessly through the source body artwork")
			}
			bodyMissChecked = true
		}
		if probeCore {
			if !outerAlive || beforeCore != 200 || w.FifthMiddle.Parts[5].Health != 197 || !w.fifthMiddleActors[0].Flash {
				t.Fatal("original narrow core did not accept its independent hit beside living mounts")
			}
			earlyCoreHit = true
		}
		if !w.PlayerAlive || w.Equipment.Lives != lives {
			t.Fatalf("arranged source middle boundary lost its ship at pass%d camera%d shield%d pose%d,%d rewind%d bodyMiss%v early%v core%d mounts%v body%d coreBounds%+v", pass, w.ScrollY, w.Equipment.Shield, w.Player.X, w.Player.Y, w.Rewind.Timer, bodyMissChecked, earlyCoreHit, w.FifthMiddle.Parts[5].Health, [4]FifthGuardianPartState{w.FifthMiddle.Parts[1], w.FifthMiddle.Parts[2], w.FifthMiddle.Parts[3], w.FifthMiddle.Parts[4]}, w.FifthMiddle.Parts[0].Y, core.Collision)
		}
	}
	if !bodyMissChecked || !earlyCoreHit || !w.FifthMiddle.Defeated || w.PendingExitDrops != 10 || len(w.Collectibles) != 10 || w.ShopReady || w.ExitReady || w.LevelFinished || w.ContinueCredits != credits || w.Score != 2300 {
		t.Fatalf("source middle lethal projectile lost its reward boundary: bodyMiss%v earlyCore%v defeated%v drops%d score%d", bodyMissChecked, earlyCoreHit, w.FifthMiddle.Defeated, w.PendingExitDrops, w.Score)
	}
	for row := 141; row < 151; row++ {
		for column := 6; column < 14; column++ {
			if w.Coverage.Map[row*20+column] != 0 {
				t.Fatal("native middle death did not clear its exact terrain rectangle")
			}
		}
	}
	for pass := 0; pass < 240 && !w.ShopReady; pass++ {
		// Legal alternating forward/reverse commands keep the source camera
		// near the cleared arena while its real coin callbacks run to completion.
		input := Input{Motion: MotionInput{Down: w.Frame&1 == 0, Right: w.Player.X < 155, Left: w.Player.X > 165}}
		for range 3 {
			w.AdvancePALTick()
		}
		if _, err := s.Advance(input); err != nil {
			t.Fatal(err)
		}
	}
	if !w.ShopReady || w.ExitReady || w.LevelFinished || w.PendingExitDrops != 0 || !w.PlayerAlive || w.Money < 0 || w.Money > 750 || w.ContinueCredits != credits {
		t.Fatal("natural fifth-middle coins failed to admit only the same-stage merchant")
	}
	for _, actor := range w.fifthMiddleActors {
		slot := w.Pool.Slot(actor.Binding.Slot)
		if actor.Active || slot.allocated && slot.EntityID == actor.ID {
			t.Fatal("middle guardian retained a living physical slot at merchant admission")
		}
	}
	camera, frame, player, equipment, random, money, score := w.ScrollY, w.Frame, w.Player, w.Equipment, w.RandomState(), w.Money, w.Score
	w.ResumeShop()
	if s.ActiveWorld() != w || w.Ready || w.ShopReady || w.ExitReady || w.LevelFinished || w.ScrollY != camera || w.Frame != frame || w.Player != player || w.Equipment != equipment || w.RandomState() != random || w.Money != money || w.Score != score || w.Checkpoint.Loadout != w.Equipment.WeaponLoadout {
		t.Fatal("middle merchant return did not preserve the actual source world")
	}
	for range 3 {
		w.AdvancePALTick()
	}
	if _, err := s.Advance(Input{}); err != nil {
		t.Fatal(err)
	}
	if s.ActiveWorld() != w || w.Level.Number != 5 || w.Frame != frame+1 || w.ScrollY != camera-1 || !w.PlayerAlive || w.LevelFinished || w.ContinueCredits != credits || s.Difficulty != 1 {
		t.Fatal("ordinary post-merchant pass did not continue the same fifth stage")
	}
}
