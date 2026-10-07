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

func TestWorldFixedAttackTransfersBeforeItsShots(t *testing.T) {
	w := testWorld(t)
	w.Level.Rules = &visualassets.LevelRules{DefaultEnemyShot: "default"}
	w.Player.Y = 100
	w.ScrollDelta = 0
	idle := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "idle"}}}
	attack := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "attack", Duration: 2}}, Ending: "remove"}
	kind := &visualassets.FixedSpriteKind{Behavior: "scroll-bounce-attack", ActorList: "moving",
		MotionParameters: map[string]int{"clip_margin": 208, "attack_y_minimum": -30, "attack_y_maximum": 10, "attack_random_threshold": 256},
		Variants: []visualassets.FixedSpriteVariant{{ID: 0, Animation: idle, AttackAnimation: attack,
			ShotMode: "point-burst", ShotSprite: "shot", ShotDirections: []int{3, 2, 1}, ShotSpeed: 4}}}
	actor := &WorldActor{ID: 1, Active: true, ActorList: "moving", fixedKind: kind, part: &visualassets.ActorPart{},
		fixedState: FixedSpriteState{X: 40, Y: 100, VelocityY: 1, Animation: NewAnimation(idle)}}
	w.nextActorID = 1
	w.advanceFixedSprite(actor)
	if actor.ActorList != "transient" || actor.Order != 2 || len(w.Projectiles) != 3 || w.Projectiles[0].ID != 5 || w.Projectiles[2].ID != 3 {
		t.Fatal("attack transfer must retain identity and precede new burst shots")
	}
}
