package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestSpecialActorsKeepPreviousCompletedCoordinates(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 5))
	if err != nil {
		t.Fatal(err)
	}
	if !w.spawnSpecializedFixedShot(FixedSpriteEvents{ShotMode: "animated-aiming-projectile", ShotCount: 1, ShotX: 80, ShotY: 60, ShotSpeed: 4, ShotDelay: 2}) || len(w.Actors) == 0 {
		t.Fatal("original aiming projectile did not construct")
	}
	actor := w.Actors[0]
	for range 2 {
		x, y := actor.X, actor.Y
		w.advanceFixedAimingActor(actor)
		if actor.PreviousX != x || actor.PreviousY != y {
			t.Fatal("aiming projectile interpolated from its construction position")
		}
	}
	scenery := &WorldActor{X: 0, Y: -4608, thirdScenery: true, Active: true, ActorList: "scenery", part: &visualassets.ActorPart{}}
	w.thirdFinalUpdated = true
	for _, scroll := range []int{4607, 4606} {
		x, y := scenery.X, scenery.Y
		w.ScrollY = scroll
		if err := w.advanceSceneryActor(scenery); err != nil {
			t.Fatal(err)
		}
		if scenery.PreviousX != x || scenery.PreviousY != y || scenery.Y != float64(-scroll) {
			t.Fatal("conditional scenery lost its previous completed camera position")
		}
	}
}
