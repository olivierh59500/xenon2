package engine

import "testing"

func TestThirdWorldGuardiansActivateAtSourceSelectorsOptional(t *testing.T) {
	data := originalWorldData(t, 3)
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	if w.ThirdMiddle != nil || w.ThirdFinal == nil {
		t.Fatal("level init must prepare shared health without allocating the middle boss")
	}
	slot := w.Pool.freeFirst
	w.Pool.Slot(slot).Residue.Health = 13
	w.Pool.Slot(slot).Residue.SetFireState(4, 23)
	var found bool
	for _, record := range data.Encounters.Fixed {
		if record.EnemyKind == 3 {
			w.spawnFixed(record)
			found = true
			break
		}
	}
	if !found || w.ThirdMiddle == nil || w.ThirdMiddle.ResidualFireRate != 23 {
		t.Fatal("fixed kind3 must activate the source middle constructor and its inherited firing rate")
	}
	if w.thirdMiddleActors[0].Health != 13 || w.thirdMiddleActors[0].Binding.Residue.Health != 13 || w.thirdMiddleActors[0].Binding.Residue.FireAccumulator() != 0 {
		t.Fatal("middle constructor must preserve slot health and clear only its firing accumulator")
	}
	if len(w.thirdMiddleActors) != 17 || w.Pool.last[ActorPoolMoving] != w.thirdMiddleActors[16].Binding.Slot {
		t.Fatal("middle body must allocate seventeen parts at the list tail")
	}
	w.ScrollY, w.PreviousScrollY, w.RenderScrollY, w.MaximumScrollY, w.VisitedScrollY = 2800, 2800, 2800, 2800, 2800
	w.cursor = RestartEncounterCursor(w.ScrollY)
	w.InvulnerableFrames = 10000
	for pass := range 120 {
		if err := w.Step(Input{}); err != nil {
			t.Fatalf("middle pass%d: %v", pass, err)
		}
		if w.ScrollY != 2800 || w.MinimumScrollY != 2800 {
			t.Fatal("living middle boss must retain the source boundary")
		}
	}
	w.damageThirdMiddle(w.thirdMiddleActors[3], w.ThirdMiddle.EyeHealth[0])
	if w.ThirdMiddle.Defeated || w.MinimumScrollY != 2800 {
		t.Fatal("one eye is insufficient to release the middle arena")
	}
	w.damageThirdMiddle(w.thirdMiddleActors[4], w.ThirdMiddle.EyeHealth[1])
	if !w.ThirdMiddle.Defeated || w.PendingExitDrops != 10 || w.LevelFinished {
		t.Fatal("two destroyed eyes must emit ten middle coins without final completion")
	}
	for _, coin := range w.Collectibles {
		coin.Motion.Mode, coin.Motion.Y = 0, 199
		w.advanceCollectible(coin)
	}
	if !w.ShopReady || w.ExitReady {
		t.Fatal("middle reward exhaustion must request the same-level shop")
	}
	w.ResumeShop()
	for range 2 {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if w.MinimumScrollY != 0 {
		t.Fatal("consuming the stale middle heartbeat must release the lower camera bound")
	}
}

func TestThirdWorldFinalWormPreservesFractionsAndCompletesOptional(t *testing.T) {
	data := originalWorldData(t, 3)
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 160, 0, 192, 192
	w.cursor = RestartEncounterCursor(w.ScrollY)
	w.InvulnerableFrames = 10000
	slot := w.Pool.freeFirst
	w.Pool.Slot(slot).Residue.XFraction = 0x1234
	w.Pool.Slot(slot).Residue.YFraction = 0x5678
	w.Pool.Slot(slot).Residue.SetFireState(9, 17)
	if err := w.spawnThirdFinal(); err != nil {
		t.Fatal(err)
	}
	var head *WorldActor
	count := 0
	for _, actor := range w.Actors {
		if actor.thirdFinalMember != nil {
			count++
			if actor.thirdPart.Index == 0 {
				head = actor
			}
		}
	}
	if count != 11 || head == nil || uint16(head.thirdFinalMember.Motion.X) != 0x1234 || uint16(head.thirdFinalMember.Motion.Y) != 0x5678 || head.thirdFinalMember.ResidualFireRate != 17 {
		t.Fatal("final constructor must create eleven independent delayed members and retain reused fractions/rate")
	}
	w.damageActor(head, w.ThirdFinal.Health-1)
	if w.ThirdFinal.Defeated || w.ThirdFinal.Health != 1 || w.LevelFinished {
		t.Fatal("final shared health must survive a nonlethal hit")
	}
	w.damageActor(head, 1)
	if !w.ThirdFinal.Defeated || w.LevelFinished {
		t.Fatal("final controller must schedule completion at its late stage callback")
	}
	if err := w.advanceThirdStage(); err != nil {
		t.Fatal(err)
	}
	if !w.LevelFinished || w.PendingExitDrops != 20 || w.ExitReady {
		t.Fatal("final stage callback must issue exactly twenty coins before exit")
	}
	for _, coin := range w.Collectibles {
		coin.Motion.Mode, coin.Motion.Y = 0, 199
		w.advanceCollectible(coin)
	}
	if !w.ExitReady || !w.ShopReady {
		t.Fatal("final cash completion must admit the source final shop")
	}
}

func TestThirdWorldCheckpointRetainsLivingMiddleBindingsOptional(t *testing.T) {
	data := originalWorldData(t, 3)
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.activateThirdMiddle(); err != nil {
		t.Fatal(err)
	}
	w.ThirdMiddle.EyeHealth[0] = 7
	ids := [17]int{}
	slots := [17]int{}
	for i, actor := range w.thirdMiddleActors {
		ids[i], slots[i] = actor.ID, actor.Binding.Slot
	}
	w.Checkpoint.ScrollY = 2800
	w.RestartCheckpoint()
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
	if w.ThirdMiddle.EyeHealth[0] != 7 {
		t.Fatal("ship restart must not heal the persistent middle eyes")
	}
	count := 0
	for _, actor := range w.Actors {
		if actor.thirdMiddlePart > 0 {
			count++
			i := actor.thirdMiddlePart - 1
			if actor.ID != ids[i] || actor.Binding.Slot != slots[i] {
				t.Fatal("checkpoint must preserve existing physical middle bindings")
			}
		}
	}
	if count != 17 {
		t.Fatalf("restored middle group has %d of17 parts", count)
	}
}
