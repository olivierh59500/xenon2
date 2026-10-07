package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestSecondWorldHatchSpawnsEightSourceCreaturesOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY = 800
	if !w.spawnFixedHatch(visualassets.FixedEncounter{EnemyKind: 2, X: 104, Y: 1000, Variant: 0}) {
		t.Fatal("source hatch was not created")
	}
	hatch := w.Actors[0]
	for range 21 {
		w.advanceFixedHatch(hatch)
	}
	count := 0
	for _, actor := range w.Actors {
		if actor.hatchCreature != nil {
			count++
			if actor.Health != 1 || actor.Score != 20 || actor.Sprite == "" || actor.Binding.EntityID == 0 {
				t.Fatal("trap child lacks original state or artwork")
			}
		}
	}
	if hatch.Active || count != 8 || hatch.Visible {
		t.Fatal("completed hatch must leave its final tiles and exactly eight moving creatures")
	}
	if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
}
