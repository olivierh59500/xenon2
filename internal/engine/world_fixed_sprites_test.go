package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestWorldFixedBeamContactUsesSourceDamageAndDiveSuppression(t *testing.T) {
	w := testWorld(t)
	w.playerCollision = CollisionRect{Left: 120, Top: 108, Right: 130, Bottom: 124}
	kind := &visualassets.FixedSpriteKind{Behavior: "extending-beam", ActorList: "scenery", ContactDamage: 6,
		MotionParameters: map[string]int{"clip_bottom": 392, "maximum_phase": 6, "phase_spacing": 16, "fire_rate": 8},
		Variants:         []visualassets.FixedSpriteVariant{{ID: 0, Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "beam"}}}}}}
	actor := &WorldActor{Active: true, ActorList: "scenery", fixedKind: kind,
		fixedState: FixedSpriteState{X: 100, Y: 100, Phase: 1, PhaseDirection: 1},
		part:       &visualassets.ActorPart{MotionMode: "world-anchored"}}
	w.ScrollDelta = 0
	w.advanceFixedSprite(actor)
	if w.Equipment.Shield != 33 || actor.X != 100 || actor.Y != 100 {
		t.Fatal("source extending beam must inflict six shield points")
	}
	w.Dive.Phase = 4
	w.advanceFixedSprite(actor)
	if w.Equipment.Shield != 33 {
		t.Fatal("diving must suppress the scenery beam's contact")
	}
}

func fixedBurstFixture(t *testing.T, sprite string) (*World, *WorldActor) {
	t.Helper()
	w := testWorld(t)
	w.Level.Rules = &visualassets.LevelRules{DefaultEnemyShot: "default"}
	w.Player.Y = 100
	w.ScrollDelta = 0
	idle := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "idle"}}}
	attack := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "attack", Duration: 2}}, Ending: "remove"}
	kind := &visualassets.FixedSpriteKind{Behavior: "scroll-bounce-attack", ActorList: "moving",
		MotionParameters: map[string]int{"clip_margin": 208, "attack_y_minimum": -30, "attack_y_maximum": 10, "attack_random_threshold": 256},
		Variants: []visualassets.FixedSpriteVariant{{ID: 0, Animation: idle, AttackAnimation: attack,
			ShotMode: "point-burst", ShotSprite: sprite, ShotDirections: []int{3, 2, 1}, ShotSpeed: 4}}}
	actor := &WorldActor{ID: 1, Active: true, ActorList: "moving", fixedKind: kind, part: &visualassets.ActorPart{},
		fixedState: FixedSpriteState{X: 40, Y: 100, VelocityY: 1, Animation: NewAnimation(idle)}}
	w.nextActorID = 1
	w.advanceFixedSprite(actor)
	return w, actor
}

func TestWorldFixedAttackTransfersBeforeItsShots(t *testing.T) {
	w, actor := fixedBurstFixture(t, "shot")
	if actor.ActorList != "transient" || actor.Order != 2 || len(w.Projectiles) != 3 || w.Projectiles[0].ID != 5 || w.Projectiles[2].ID != 3 {
		t.Fatal("attack transfer must retain identity and precede new burst shots")
	}
	for _, shot := range w.Projectiles {
		if shot.Sprite != "shot" || shot.Atlas != "fixed" {
			t.Fatal("explicit shot artwork must retain its fixed atlas")
		}
	}
}

func TestWorldFixedBurstKeepsDefaultArtworkAndPlayerContact(t *testing.T) {
	w, _ := fixedBurstFixture(t, "")
	if len(w.Projectiles) != 3 {
		t.Fatal("ordinary burst lost one of its three shots")
	}
	for _, shot := range w.Projectiles {
		if shot.Sprite != "default" || shot.Atlas != "enemy-shots" {
			t.Fatal("omitted override erased the ordinary point-shot image or atlas")
		}
	}
	w.InvulnerableFrames = 0
	w.Equipment.Shield = 39
	w.shotSpriteBoxes = map[string]visualassets.CollisionBox{"default": {Width: 1, Height: 1}}
	w.playerCollision = CollisionRect{Left: 0, Top: 0, Right: 320, Bottom: 200}
	shot := w.Projectiles[0]
	if err := w.advanceEnemyShot(shot); err != nil {
		t.Fatal(err)
	}
	if shot.Active || w.Equipment.Shield != 35 {
		t.Fatal("the visible point shot lost its original four-point player contact")
	}
}
