package engine

import "testing"

func TestThirdFixedFamiliesUseOriginalResourcesOptional(t *testing.T) {
	data := originalWorldData(t, 3)
	for _, kind := range []int{1, 5, 6} {
		t.Run(string(rune('0'+kind)), func(t *testing.T) {
			w, err := NewWorld(data)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, record := range data.Encounters.Fixed {
				if record.EnemyKind == kind {
					w.ScrollY = record.TriggerY
					w.MaximumScrollY = 4607
					w.cursor = RestartEncounterCursor(w.ScrollY)
					w.spawnFixed(record)
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("missing source fixed kind%d", kind)
			}
			var actor *WorldActor
			count := 0
			for _, a := range w.Actors {
				if kind == 1 && a.thirdChainPart > 0 {
					count++
					if a.thirdChainPart == 1 {
						actor = a
					}
				}
				if kind == 5 && a.thirdCannon != nil {
					actor = a
				}
				if kind == 6 && a.thirdCrawler != nil {
					actor = a
				}
			}
			if actor == nil {
				t.Fatalf("fixed kind%d did not create its Go controller", kind)
			}
			if kind == 1 && count != 8 {
				t.Fatalf("chain has %d members", count)
			}
			w.InvulnerableFrames = 10000
			for i := 0; i < 50; i++ {
				if err := w.Step(Input{}); err != nil {
					t.Fatalf("pass%d: %v", i, err)
				}
			}
			if kind == 5 {
				score := w.Score
				w.damageActor(actor, uint16(actor.Health))
				if actor.thirdCannon.Stage != 1 || !actor.Active || w.Score != score+500 {
					t.Fatal("first cannon defeat must retain the second body/core stage")
				}
				w.damageActor(actor, uint16(actor.Health))
				if actor.Active || w.Score != score+1000 {
					t.Fatal("second cannon defeat must remove its map and grant the second source score")
				}
			}
		})
	}
}

func TestThirdInitialSceneryUsesOnePhysicalSlotOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	var scenery *WorldActor
	for _, actor := range w.Actors {
		if actor.thirdScenery {
			count++
			scenery = actor
		}
	}
	if count != 1 || scenery.Binding.EntityID == 0 || worldActorPoolList(scenery) != ActorPoolScenery {
		t.Fatal("source initializer must reserve exactly one scenery actor")
	}
	w.thirdFinalUpdated = false
	if err := w.advanceSceneryActor(scenery); err != nil {
		t.Fatal(err)
	}
	if scenery.Visible {
		t.Fatal("conditional scenery must wait for the final tail update")
	}
	w.thirdFinalUpdated = true
	if err := w.advanceSceneryActor(scenery); err != nil {
		t.Fatal(err)
	}
	if !scenery.Visible || scenery.Patch == nil || scenery.Patch.Columns != 20 || scenery.Patch.Rows != 20 {
		t.Fatal("active final worm must display the original sparse tile scenery")
	}
}
