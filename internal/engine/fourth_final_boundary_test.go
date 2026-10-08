package engine

import "testing"

func fourthFinalBoundaryPoint(w *World, index int) (int, int, bool) {
	target := w.fourthFinalActors[index]
	if target == nil || !target.Active {
		return 0, 0, false
	}
	r := target.Collision
	if index == 0 && w.FourthFinal.EyesRemaining != 0 {
		// This is where the native core weakness will be published after both
		// eyes close. While locked, the body has no collider at this point.
		top := int(w.FourthFinal.Parts[0].Arc.Y>>16) + 32
		r = CollisionRect{Left: 148, Right: 172, Top: top, Bottom: top + 8}
	}
	x, y := (r.Left+r.Right)/2, (r.Top+r.Bottom)/2
	return x, y, x >= 0 && x < 320 && y >= 0 && y < 192
}

// This arranges the original final arena at its lower camera bound. It is not
// an earned fourth-stage victory. Original factory bullets, actor callbacks,
// fifty-HP eyes, hundred-HP core and reward lifetimes remain unmodified.
func TestOriginalFourthFinalProjectilesDrainExitAndAdmitFifthOptional(t *testing.T) {
	e := NewEquipment()
	for _, item := range []Item{ItemPowerup, ItemPowerup, ItemCannon, ItemRearShot, ItemSpeedup, ItemSpeedup, ItemAutofire, ItemAutofire} {
		e.ApplyItem(item)
	}
	data := playableOriginalWorldData(t, 4)
	data.InitialEquipment = &e
	s, err := NewSession(data, 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := s.ActiveWorld()
	w.Ready = false
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 0, 0, 16, 16
	w.cursor = EncounterCursor{MovingHighWater: 1, FixedHighWater: 0}
	w.Player.Y = 176
	clearPose := false
	for x := 14; x <= 304; x++ {
		if !w.Coverage.Touches(x, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
			w.Player.X, clearPose = x, true
			break
		}
	}
	if !clearPose {
		t.Fatal("original final arena has no clear bottom ship pose")
	}
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	for _, record := range w.Level.Encounters.Fixed {
		if record.EnemyKind == 4 {
			w.spawnFixed(record)
			break
		}
	}
	if w.FourthFinal == nil || w.FourthFinal.EyesRemaining != 2 || w.FourthFinal.Parts[0].Health != 100 || w.FourthFinal.Parts[1].Health != 50 || w.FourthFinal.Parts[2].Health != 50 {
		t.Fatal("original selector did not retain both fifty-HP eyes and the locked hundred-HP core")
	}
	credits, lives, score := w.ContinueCredits, w.Equipment.Lives, w.Score
	locked := [2]bool{}
	var next WorldForecast
	for pass := 0; pass < 700 && !w.FourthFinal.Defeated; pass++ {
		if err := next.Load(w); err != nil {
			t.Fatal(err)
		}
		for range 3 {
			next.AdvancePALTick()
		}
		if _, err := next.Advance(Input{}); err != nil {
			t.Fatal(err)
		}
		future := next.State()
		eyes, core := w.FourthFinal.EyesRemaining, w.FourthFinal.Parts[0].Health
		var lockedShot *WorldSmallShot
		lockedIndex := -1
		if eyes > 0 && !locked[2-eyes] {
			if x, y, found := fourthFinalBoundaryPoint(future, 0); found {
				fifthBoundaryShot(t, w, x, y)
				lockedShot = w.SmallShots[len(w.SmallShots)-1]
				lockedIndex = 2 - eyes
			}
		} else {
			index := 0
			if eyes > 0 {
				index = 1
				if w.FourthFinal.Parts[index].Disabled {
					index = 2
				}
			}
			if x, y, found := fourthFinalBoundaryPoint(future, index); found {
				fifthBoundaryShot(t, w, x, y)
			}
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if _, err := s.Advance(Input{}); err != nil {
			t.Fatal(err)
		}
		if lockedIndex >= 0 {
			if !lockedShot.Active || w.FourthFinal.Parts[0].Health != 100 || w.FourthFinal.EyesRemaining != eyes || w.PendingExitDrops != 0 || w.Score != score {
				t.Fatal("actual projectile reached a core collider before both original eyes were disabled")
			}
			locked[lockedIndex] = true
		}
		if eyes > 0 && w.FourthFinal.Parts[0].Health != core {
			t.Fatal("core health changed before its two-eye gate opened")
		}
		if w.FourthFinal.EyesRemaining == 1 {
			eye := w.fourthFinalActors[1]
			if !eye.Active || !w.FourthFinal.Parts[1].Disabled || eye.Patch != w.fourthFinalArt.Components[1].DestroyedTiles {
				t.Fatal("real lethal eye projectile lost its retained closed actor/artwork")
			}
		}
		if !w.PlayerAlive || w.Equipment.Lives != lives {
			t.Fatalf("arranged final boundary lost its ship at pass%d shield%d", pass, w.Equipment.Shield)
		}
	}
	if !locked[0] || !locked[1] || !w.FourthFinal.Defeated || w.FourthFinal.EyesRemaining != 0 || !w.LevelFinished || w.PendingExitDrops != 20 || len(w.Collectibles) != 20 || w.ShopReady || w.ExitReady || w.Score != score || w.ContinueCredits != credits {
		t.Fatal("native final projectile did not create the genuine twenty-coin stage exit")
	}
	if w.Pool.First(ActorPoolMoving) != NoActorSlot {
		t.Fatal("native final kill failed to immediately release the physical moving list")
	}
	for _, coin := range w.Collectibles {
		if !coin.Active || coin.Cash != 50 && coin.Cash != 100 {
			t.Fatal("final factory did not retain twenty normal cash rewards")
		}
	}
	for pass := 0; pass < 240 && !w.ShopReady; pass++ {
		input := Input{Motion: MotionInput{Down: true, Right: w.Player.X < 155, Left: w.Player.X > 165}}
		for range 3 {
			w.AdvancePALTick()
		}
		if _, err := s.Advance(input); err != nil {
			t.Fatal(err)
		}
	}
	if !w.ShopReady || !w.ExitReady || !w.LevelFinished || w.PendingExitDrops != 0 || !w.PlayerAlive || w.Equipment.Lives != lives || w.Money < 0 || w.Money > 1500 || w.ContinueCredits != credits {
		t.Fatal("natural final coin callbacks failed to open the actual exit merchant")
	}
	gear, shield, random := fifthBoundaryInventory(w.Equipment), w.Equipment.Shield, w.RandomState()
	transition, err := s.CompleteStage(playableOriginalWorldData(t, 5))
	if err != nil || transition != LoadedNextStage {
		t.Fatalf("fourth final merchant did not admit level five: %v %v", transition, err)
	}
	fifth := s.ActiveWorld()
	if fifth == w || fifth.Level.Number != 5 || !fifth.Ready || !fifth.PlayerAlive || fifth.LevelFinished || fifth.ShopReady || fifth.ExitReady || fifth.PendingExitDrops != 0 || fifth.Money != 0 || fifth.Score != score || fifth.ContinueCredits != credits || fifth.Equipment.Lives != lives || fifth.Equipment.Shield != shield || s.Difficulty != 1 {
		t.Fatal("fourth-to-fifth admission changed lives, credits, shield, wallet reset or clean stage gates")
	}
	if fifthBoundaryInventory(fifth.Equipment) != gear || fifth.Equipment.FireAdvance != 3 || fifth.Equipment.FirePeriod != 8 || fifth.Equipment.SpeedTier != 2 || fifth.RandomState() != random || fifth.Pool == w.Pool || fifth.Weapons == w.Weapons || len(fifth.Projectiles) != 0 || len(fifth.SmallShots) != 0 || len(fifth.Collectibles) != 0 || fifth.FifthMiddle != nil || fifth.FifthFinal != nil {
		t.Fatal("new fifth stage failed to retain regular equipment/RNG or shared old live entities")
	}
	if _, err := s.CompleteStage(playableOriginalWorldData(t, 1)); err == nil || fifth.ContinueCredits != credits || s.Difficulty != 1 {
		t.Fatal("fourth completion incorrectly awarded the fifth-victory credit or difficulty loop")
	}
}
