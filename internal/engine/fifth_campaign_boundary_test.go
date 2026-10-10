package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

// fifthBoundaryShot arranges one already-emitted ordinary tier-two bullet. Its
// damage and movement come from the production small-weapon factory; all actor
// collisions, damage callbacks and retirement still run through World.Step.
func fifthBoundaryShot(t *testing.T, w *World, x, y int) {
	t.Helper()
	var storage [2]SmallShot
	shots, err := AppendSmallWeaponShots(storage[:0], WeaponSlot{Item: ItemForwardShot, Tier: 2}, x, y+15)
	if err != nil || len(shots) != 1 || shots[0].Damage != 3 {
		t.Fatalf("ordinary source bullet factory: shots%v error%v", shots, err)
	}
	binding, err := w.reserveWorldActor(16, ActorPoolProjectile, false)
	if err != nil {
		t.Fatal(err)
	}
	shot := &WorldSmallShot{ID: binding.EntityID, Binding: binding, Active: true,
		PreviousX: shots[0].X, PreviousY: shots[0].Y, Shot: shots[0]}
	w.poolSmallShots[binding.Slot] = shot
	w.SmallShots = append(w.SmallShots, shot)
}

func fifthBoundaryInventory(e Equipment) [7][3]int {
	var result [7][3]int
	for index, slot := range e.slots() {
		result[index] = [3]int{int(slot.Item), slot.Tier, slot.MaxTier}
	}
	return result
}

// This is an arranged original-resource boundary fixture, not an earned fifth
// stage or campaign victory. It keeps every native defense/core health value
// and source gate, and injects in-flight bullets rather than calling damage or
// assigning completion/drop counters. Ordinary source movement drains rewards.
func TestOriginalFifthProjectileDeathDrainsExitAndStartsNextLoopOptional(t *testing.T) {
	data := playableOriginalWorldData(t, 5)
	equipment := NewEquipment()
	for _, item := range []Item{ItemPowerup, ItemCannon, ItemRearShot, ItemSpeedup, ItemSpeedup, ItemAutofire, ItemAutofire} {
		equipment.ApplyItem(item)
	}
	data.InitialEquipment = &equipment // Explicit regular-loadout fixture, not earned inventory.
	s, err := NewSession(data, 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := s.ActiveWorld()
	w.Ready = false
	w.Player.X, w.Player.Y = 24, 176
	w.cursor = RestartEncounterCursor(0) // Earlier encounters precede this isolated arena.
	var final visualassets.FixedEncounter
	for _, record := range w.Level.Encounters.Fixed {
		if record.EnemyKind == 6 {
			final = record
			break
		}
	}
	if final.EnemyKind != 6 {
		t.Fatal("original final guardian encounter is missing")
	}
	w.spawnFixed(final)
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	if w.FifthFinal == nil || w.FifthFinal.OuterRemaining != 18 || w.FifthFinal.CoreHealth != 20 || w.LevelFinished {
		t.Fatal("source factory did not retain the original gated final guardian")
	}
	credits, lives := w.ContinueCredits, w.Equipment.Lives
	for pass := 0; pass < 700 && !w.FifthFinal.Defeated; pass++ {
		input := Input{}
		if w.FifthFinal.OuterRemaining == 0 {
			// The core lies below the upper defenses. Native reverse scrolling
			// exposes it again after the ship has reached the upper component row.
			input.Motion.Down = true
		}
		for index := 3; index < len(w.fifthFinalActors); index++ {
			actor := w.fifthFinalActors[index]
			if !actor.Active || w.FifthFinal.Parts[index].Destroyed || actor.Collision.Empty() {
				continue
			}
			x, y := actor.Collision.Left+1, actor.Collision.Top+w.ScrollDelta+1
			if y >= 0 && y < 192 {
				fifthBoundaryShot(t, w, x, y)
			}
		}
		if _, err := s.Advance(input); err != nil {
			t.Fatal(err)
		}
		if !w.PlayerAlive || w.Equipment.Lives != lives || w.GameOver {
			t.Fatalf("isolated boundary fixture lost its ship before final defeat: pass%d camera%d shield%d", pass, w.ScrollY, w.Equipment.Shield)
		}
	}
	if !w.FifthFinal.Defeated || !w.LevelFinished || w.FifthFinal.OuterRemaining != 0 || w.PendingExitDrops != 20 || w.ShopReady || w.ExitReady {
		t.Fatalf("ordinary final projectile did not reach the native exit gate: defeated%v remaining%d core%d drops%d shop%v exit%v", w.FifthFinal.Defeated, w.FifthFinal.OuterRemaining, w.FifthFinal.CoreHealth, w.PendingExitDrops, w.ShopReady, w.ExitReady)
	}
	if w.Score != 18*200 || w.ContinueCredits != credits || len(w.Collectibles) != 20 {
		t.Fatal("final callback changed defense rewards, awarded an early credit or lost exit coins")
	}
	// Immediate moving-list release changes the inherited coin headings. This
	// fixture retains all twenty paired coins after the lethal projectile pass.
	created, live, small, large, remainingValue := 0, 0, 0, 0, 0
	for _, coin := range w.poolCollectibles {
		if coin == nil {
			continue
		}
		created++
		switch coin.Cash {
		case 50:
			small++
		case 100:
			large++
		default:
			t.Fatal("source exit factory created another reward type")
		}
		if coin.Active {
			live++
			remainingValue += coin.Cash
		}
	}
	if created != 20 || small != 10 || large != 10 || live != 20 || live != w.PendingExitDrops || remainingValue+w.Money != 1500 {
		t.Fatalf("exit callback lost original paired cash: created%d live%d small%d large%d remaining%d wallet%d", created, live, small, large, remainingValue, w.Money)
	}
	for pass := 0; pass < 220 && !w.ShopReady; pass++ {
		input := Input{Motion: MotionInput{Right: w.Player.X < 155, Left: w.Player.X > 165, Up: w.Player.Y > 105, Down: w.Player.Y < 95}}
		if _, err := s.Advance(input); err != nil {
			t.Fatal(err)
		}
		if !w.PlayerAlive || w.Equipment.Lives != lives {
			t.Fatal("native pending-exit sequence lost its protected surviving ship")
		}
	}
	if !w.ShopReady || !w.ExitReady || !w.LevelFinished || w.PendingExitDrops != 0 || w.Money < 300 || w.Money > 1500 {
		t.Fatalf("real coin callbacks failed to reach the merchant boundary: drops%d shop%v exit%v money%d", w.PendingExitDrops, w.ShopReady, w.ExitReady, w.Money)
	}
	for _, coin := range w.Collectibles {
		if coin.Active {
			t.Fatal("merchant boundary retained an active exit coin")
		}
	}
	for _, actor := range w.fifthFinalActors {
		slot := w.Pool.Slot(actor.Binding.Slot)
		if actor.Active || slot.allocated && slot.EntityID == actor.ID {
			t.Fatal("final guardian retained a live physical slot after exit callbacks")
		}
	}

	// Spend actual emitted cash at the real final-level price, then initialize
	// the temporary suite exactly as leaving the merchant does.
	regular := fifthBoundaryInventory(w.Equipment)
	rules := w.PrepareShop(ShopRules{Level: 5, StockLimit: w.Level.Terrain.EndShopStockLimit})
	if price, err := rules.Buy(&w.Equipment, &w.Money, ItemSuperNashwan); err != nil || price != 300 {
		t.Fatalf("earned cash could not purchase the original discounted Nashwan: price%d error%v", price, err)
	}
	rules.Leave(&w.Equipment)
	if !w.Equipment.SuperLoadoutActive || w.Equipment.SuperFrames != 170 {
		t.Fatal("merchant departure did not install the pending native temporary suite")
	}
	nextData := playableOriginalWorldData(t, 1)
	transition, err := s.CompleteStage(nextData)
	if err != nil || transition != LoadedNextStage || s.Difficulty != 2 {
		t.Fatalf("genuine final boundary did not start the next loop: %v %v difficulty%d", transition, err, s.Difficulty)
	}
	next := s.ActiveWorld()
	if next == w || next.Level.Number != 1 || !next.Ready || next.LevelFinished || next.ShopReady || next.ExitReady || next.PendingExitDrops != 0 || next.Money != 0 || next.ContinueCredits != credits+1 || next.Equipment.Lives != lives {
		t.Fatal("next-loop admission lost source credits, wallet reset, living ship or clean gates")
	}
	if !next.Equipment.SuperLoadoutActive || next.Equipment.SuperFrames != 170 {
		t.Fatal("next-loop reconstruction lost or advanced the live Nashwan timer")
	}
	regularCopy := next.Equipment
	regularCopy.RestoreSuperLoadout()
	checkpointCopy := Equipment{WeaponLoadout: next.Checkpoint.Loadout}
	if fifthBoundaryInventory(regularCopy) != regular || fifthBoundaryInventory(checkpointCopy) != regular {
		t.Fatal("fifth victory cleanup failed to restore and record the real regular loadout")
	}
	if len(next.Projectiles) != 0 || len(next.SmallShots) != 0 || len(next.Collectibles) != 0 || next.Pool == w.Pool || next.Weapons == w.Weapons {
		t.Fatal("new stage shared the old live projectile/coin/weapon pool")
	}
	if _, err := s.CompleteStage(playableOriginalWorldData(t, 2)); err == nil || next.ContinueCredits != credits+1 || s.Difficulty != 2 {
		t.Fatal("noncompleted fresh stage awarded another fifth credit")
	}
	if _, err := s.Advance(Input{Fire: true}); err != nil {
		t.Fatal(err)
	}
	found := false
	for pass := 0; pass < 64 && !found; pass++ {
		if _, err := s.Advance(Input{}); err != nil {
			t.Fatal(err)
		}
		for _, actor := range next.Actors {
			if !actor.Active || actor.fixed || actor.fifthIndex != 0 || actor.part == nil || actor.part.MotionMode != "path" {
				continue
			}
			want := 2 * nextData.Rules.OrdinaryHealthMultiplier
			if actor.part.StrongHealth {
				want = 2 * nextData.Rules.StrongHealthMultiplier
			}
			if actor.Health != want {
				t.Fatalf("first genuine next-loop wave has health%d, want native doubled health%d", actor.Health, want)
			}
			found = true
		}
	}
	if !found || next.ContinueCredits != credits+1 {
		t.Fatal("ordinary next-loop scrolling failed to admit the first native moving wave")
	}
}
