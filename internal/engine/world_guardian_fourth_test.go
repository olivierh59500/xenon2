package engine

import "testing"

func TestFourthWorldMiddleArenaProgressionOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	if w.FourthMiddle != nil || w.FourthFinal != nil {
		t.Fatal("fourth guardians must wait for their source fixed selectors")
	}
	w.Money = 1234
	if err := w.activateFourthGuardian(false); err != nil {
		t.Fatal(err)
	}
	if w.ScrollY != 2480 || w.Player.X != 160 || w.Player.Y != 176 || len(w.fourthMiddleActors) != 20 || w.Pool.Last(ActorPoolMoving) != w.fourthMiddleActors[19].Binding.Slot {
		t.Fatal("middle constructor did not preserve its source arena and tail allocation")
	}
	w.InvulnerableFrames = 10000
	for pass := 0; pass < 80; pass++ {
		if err := w.Step(Input{Fire: true}); err != nil {
			t.Fatalf("middle pass%d: %v", pass, err)
		}
	}
	w.damageActor(w.fourthMiddleActors[4], 1000)
	if w.FourthMiddle.Parts[4].Health != 175 {
		t.Fatal("locked middle core accepted damage")
	}
	w.damageActor(w.fourthMiddleActors[15], 20)
	for index := 16; index < 20; index++ {
		w.damageActor(w.fourthMiddleActors[index], 20)
	}
	if w.FourthMiddle.OuterTargets != 0 || w.Score != 2200 {
		t.Fatalf("outer-target collapse: targets=%d score=%d", w.FourthMiddle.OuterTargets, w.Score)
	}
	w.damageActor(w.fourthMiddleActors[5], 174)
	if w.FourthMiddle.Parts[4].Health != 1 || w.FourthMiddle.Defeated {
		t.Fatal("companion did not forward the nonlethal core hit")
	}
	w.damageActor(w.fourthMiddleActors[4], 1)
	if !w.FourthMiddle.Defeated || w.PendingExitDrops != 10 || w.MinimumScrollY != 0 || w.ScrollY != 2208 || w.MaximumScrollY != 2208 || w.LevelFinished || w.Score != 4200 {
		t.Fatal("middle defeat did not open the same-stage reward/terrain transition")
	}
	for row := 143; row < 159; row++ {
		for _, tile := range w.Level.Terrain.Map[row*20 : (row+1)*20] {
			if tile != 0 {
				t.Fatal("middle terrain corridor was not cleared")
			}
		}
	}
	for _, coin := range w.Collectibles {
		coin.Motion.Mode, coin.Motion.Y = 0, 199
		w.advanceCollectible(coin)
	}
	if !w.ShopReady || w.ExitReady {
		t.Fatal("middle coin exhaustion must open the same-level shop")
	}
}

func TestFourthWorldFinalEyesAndCompletionOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 0, 0, 16, 16
	w.cursor = RestartEncounterCursor(0)
	if err := w.activateFourthGuardian(true); err != nil {
		t.Fatal(err)
	}
	w.InvulnerableFrames = 10000
	for pass := 0; pass < 80; pass++ {
		if err := w.Step(Input{}); err != nil {
			t.Fatalf("final pass%d: %v", pass, err)
		}
	}
	w.damageActor(w.fourthFinalActors[1], 50)
	if w.FourthFinal.EyesRemaining != 1 || !w.fourthFinalActors[1].Active {
		t.Fatal("destroyed eye must retain its closed actor")
	}
	w.damageActor(w.fourthFinalActors[0], 10)
	if w.FourthFinal.Parts[0].Health != 100 {
		t.Fatal("one surviving eye must keep the final core locked")
	}
	w.damageActor(w.fourthFinalActors[2], 50)
	w.damageActor(w.fourthFinalActors[0], 99)
	if w.FourthFinal.Parts[0].Health != 1 {
		t.Fatal("unlocked final core lost its original health")
	}
	w.damageActor(w.fourthFinalActors[0], 1)
	if !w.FourthFinal.Defeated || !w.LevelFinished || w.PendingExitDrops != 20 || w.Pool.First(ActorPoolMoving) != NoActorSlot {
		t.Fatal("final defeat must immediately release moving actors and wait for twenty coins")
	}
	for _, coin := range w.Collectibles {
		coin.Motion.Mode, coin.Motion.Y = 0, 199
		w.advanceCollectible(coin)
	}
	if !w.ExitReady || !w.ShopReady {
		t.Fatal("final coins did not admit the stage exit shop")
	}
}

func TestFourthWorldCheckpointKeepsGuardianSlotsAndHealthOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	if err := w.activateFourthGuardian(false); err != nil {
		t.Fatal(err)
	}
	w.FourthMiddle.Parts[16].Health = 7
	before := w.fourthMiddleActors[16].Binding
	w.ScrollY = 2300
	w.RestartCheckpoint()
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
	if w.FourthMiddle.Parts[16].Health != 7 || w.fourthMiddleActors[16].Binding.Slot != before.Slot || w.fourthMiddleActors[16].ID != before.EntityID {
		t.Fatal("checkpoint replaced a surviving middle guardian")
	}
}
