package engine

import (
	"fmt"
	"testing"
	"xenon2/internal/visualassets"
)

func TestSecondWorldPodsCreateBothOriginalCreatureVariantsOptional(t *testing.T) {
	for _, kind := range []int{4, 5} {
		w, err := NewWorld(originalWorldData(t, 2))
		if err != nil {
			t.Fatal(err)
		}
		w.ScrollY, w.MaximumScrollY = 880, 1000
		if !w.spawnFixedPod(visualassets.FixedEncounter{EnemyKind: kind, X: 104, Y: 1000}) {
			t.Fatal("source emitter encounter was not consumed")
		}
		pod := w.Actors[0]
		for pass := range 60 {
			w.Frame = uint64(pass)
			w.advanceFixedPod(pod)
		}
		count := 0
		for _, actor := range w.Actors {
			if actor.podCreature != nil {
				count++
				variant := 1
				if kind == 5 {
					variant = 0
				}
				if actor.podCreature.Variant != variant || actor.Health != w.Level.FixedSprites.PodCreatures.Health[variant] || actor.Score != w.Level.FixedSprites.PodCreatures.Score[variant] || actor.Sprite == "" || actor.Binding.EntityID == 0 {
					t.Fatal("emitter lost source creature data or allocator binding")
				}
			}
		}
		if count != 2 || pod.Active || pod.Visible {
			t.Fatal("source emitter must issue two creatures and retain its final terrain frame")
		}
		if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
			t.Fatal(err)
		}
	}
}

// Encounter coverage does not include objects created by level-stage callbacks.
func TestFirstFourWorldsConsumeEveryFixedEncounterSelectorOptional(t *testing.T) {
	for number := 1; number <= 4; number++ {
		t.Run(fmt.Sprint(number), func(t *testing.T) {
			w, err := NewWorld(originalWorldData(t, number))
			if err != nil {
				t.Fatal(err)
			}
			for _, record := range w.Level.Encounters.Fixed {
				w.spawnFixed(record)
			}
			if len(w.UnimplementedFixedEncounters) != 0 {
				t.Fatalf("unimplemented source encounters:%+v", w.UnimplementedFixedEncounters)
			}
			if w.poolError != nil {
				t.Fatal(w.poolError)
			}
		})
	}
}
