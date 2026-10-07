package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func fifthBirthFixture(t *testing.T, mouth bool) (*World, *WorldActor) {
	t.Helper()
	w := testWorld(t)
	w.Level.Encounters = &visualassets.Encounters{}
	w.ScrollDelta, w.MaterializationFrames, w.BackgroundStars = 0, 0, nil
	clip := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "seeker"}}}
	component := visualassets.GuardianComponent{HeadingAnimations: make([]visualassets.ActorAnimation, 8)}
	for direction := range 8 {
		component.HeadingAnimations[direction] = clip
	}
	art := &visualassets.GuardianGroup{Components: []visualassets.GuardianComponent{component}, MotionParameters: map[string]int{"side_health": 1, "side_lifetime": 100, "mouth_creature_health": 1}, MotionTables: map[string][]int{"seeking_x": make([]int, 8), "seeking_y": make([]int, 8)}}
	w.fifthMiddleArt, w.fifthFinalArt = art, art
	w.FifthFinal = &FifthFinalGuardianState{OuterRemaining: 18}
	w.movingSpriteBoxes["seeker"] = visualassets.CollisionBox{X: -2, Y: -2, Width: 5, Height: 5}
	w.Level.GuardianParts = &visualassets.SpriteAtlas{Sprites: []visualassets.SpriteRegion{{Name: "seeker", Width: 5, Height: 5, AnchorX: 2, AnchorY: 2}}}
	w.Player.X, w.Player.Y = 16, 16
	w.Equipment.ShadesFrames = 10
	w.Level.Ships = &visualassets.ShipArt{Atlas: visualassets.SpriteAtlas{Sprites: []visualassets.SpriteRegion{{Name: "player-ship-2", Collision: &visualassets.CollisionBox{Width: 20, Height: 20}}}}}
	if mouth {
		w.spawnFifthMouth(FifthMouthCreature{X: 160, Y: 100})
	} else {
		w.spawnFifthSide(FifthGuardianShot{X: 160, Y: 100, Heading: 2})
	}
	return w, w.Actors[0]
}

func TestFifthHiddenBirthHasNoOriginCollisionBeforeItsFirstCallback(t *testing.T) {
	for _, mouth := range []bool{false, true} {
		w, actor := fifthBirthFixture(t, mouth)
		random, slot := w.RandomState(), actor.Binding.Slot
		if actor.Visible || actor.Collision.Left != 1000 || actor.Collision.Right != 1000 || w.Pool.Slot(slot).ResourceTag != int16(actor.part.ResourceTag) {
			t.Fatal("newborn seeker lost the original hidden collider sentinel or pool tag")
		}
		if w.weaponHitPoint(0, 0, 127) || w.weaponHitRect(CollisionRect{Left: -48, Top: -48, Right: 80, Bottom: 80}, 127, false) {
			t.Fatal("newborn seeker intercepted an attack before its first callback")
		}
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		if !actor.Active || !actor.Visible || actor.Health != 1 || actor.Collision != (CollisionRect{Left: 158, Top: 98, Right: 162, Bottom: 102}) || w.Score != 0 || len(w.Collectibles) != 0 || w.RandomState() != random || w.Pool.Slot(slot).ResourceTag != int16(actor.part.ResourceTag) {
			t.Fatalf("mouth%v first player/Shades pass changed newborn rewards or admission: actor%+v score%d random%+v", mouth, actor, w.Score, w.RandomState())
		}
	}
}

func TestOriginalFifthSeekingFactoriesKeepHiddenBirthSentinelsOptional(t *testing.T) {
	w := fifthResourceWorld(t)
	for _, mouth := range []bool{false, true} {
		if mouth {
			w.spawnFifthMouth(FifthMouthCreature{X: 150, Y: 40})
		} else {
			w.spawnFifthSide(FifthGuardianShot{X: 150, Y: 40, Heading: 2, InitialClock: 7})
		}
		var born *WorldActor
		for _, actor := range w.Actors {
			if actor.fifthSeeking != nil && actor.fifthMouth == mouth {
				born = actor
				break
			}
		}
		if born == nil || born.Visible || born.Collision.Left != 1000 || born.Collision.Right != 1000 {
			t.Fatal("original factory omitted its hidden collider sentinel")
		}
		before := w.RandomState()
		if w.weaponHitPoint(0, 0, 127) || w.weaponHitRect(CollisionRect{Left: -64, Top: -64, Right: 64, Bottom: 64}, 127, false) || w.Score != 0 || w.RandomState() != before {
			t.Fatal("original hidden newborn generated an origin hit/reward")
		}
		if mouth && w.FifthFinal == nil {
			w.FifthFinal = &FifthFinalGuardianState{OuterRemaining: 18}
		}
		w.advanceFifthSeeking(born)
		if born.Active && (born.Collision.Empty() || born.Collision.Left == 1000 || !born.Visible) {
			t.Fatal("original first callback did not replace the hidden collider with its recovered sprite prefix")
		}
	}
}
