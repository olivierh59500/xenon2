package engine

import (
	"fmt"
	"strings"
	"testing"

	"xenon2/internal/visualassets"
)

func TestGuardianPartBulletFactoryUsesItsOwnCollisionPrefix(t *testing.T) {
	w := testWorld(t)
	w.Level.Rules = &visualassets.LevelRules{DefaultEnemyShot: "ordinary-bullet"}
	w.shotSpriteBoxes = map[string]visualassets.CollisionBox{"ordinary-bullet": {X: 50, Y: 50, Width: 1, Height: 1}}
	w.movingSpriteBoxes["guardian-bullet"] = visualassets.CollisionBox{X: -2, Y: -2, Width: 5, Height: 5}
	group := &visualassets.GuardianGroup{Animations: []visualassets.NamedActorAnimation{{ID: "guardian-shot", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "guardian-bullet"}}}}}}
	w.spawnThirdShot(ThirdGuardianShot{X: 160, Y: 100, Animation: "guardian-shot"}, group)
	w.ScrollDelta = 0
	w.playerCollision = CollisionRect{Left: 160, Top: 100, Right: 160, Bottom: 100}
	shot := w.Projectiles[0]
	if shot.Atlas != "guardian-parts" || shot.Sprite != "guardian-bullet" {
		t.Fatal("actual guardian factory did not select its separate image bank")
	}
	if err := w.advanceEnemyShot(shot); err != nil {
		t.Fatal(err)
	}
	w.finishProjectileUpdate(shot)
	if shot.Active || w.Equipment.Shield != 35 || w.Pool.Slot(shot.Binding.Slot).ResourceTag != 4 {
		t.Fatalf("guardian-part prefix did not damage and retire the colliding bullet: active%v shield%d", shot.Active, w.Equipment.Shield)
	}
}

func TestOriginalGuardianBulletFramesCollideThroughWorldStepOptional(t *testing.T) {
	unique, cases := make(map[string]bool), 0
	for _, level := range []int{3, 4} {
		data := originalWorldData(t, level)
		for groupIndex := range data.GuardianGroups {
			group := &data.GuardianGroups[groupIndex]
			for _, clip := range group.Animations {
				if !strings.Contains(clip.ID, "shot") {
					continue
				}
				for _, frame := range clip.Animation.Frames {
					unique[fmt.Sprintf("%d/%s", level, frame.Sprite)] = true
					for _, mode := range []string{"normal", "invulnerable", "diving"} {
						w, err := NewWorld(data)
						if err != nil {
							t.Fatal(err)
						}
						w.Level.Encounters = &visualassets.Encounters{}
						w.Player.X, w.Player.Y, w.ScrollDelta = 160, 100, 0
						w.MaterializationFrames = 0
						if mode == "invulnerable" {
							w.InvulnerableFrames = 80
						}
						if mode == "diving" {
							w.Dive.Phase = 1
						}
						if level == 3 {
							w.spawnThirdShot(ThirdGuardianShot{X: 160, Y: 100, Animation: clip.ID}, group)
						} else {
							w.spawnFourthShot(FourthGuardianShot{X: 160, Y: 100, Animation: clip.ID}, group)
						}
						if len(w.Projectiles) != 1 || w.Projectiles[0].Atlas != "guardian-parts" {
							t.Fatal("original shot animation bypassed its guardian factory")
						}
						shot := w.Projectiles[0]
						// Hold one recovered frame to exercise every actual collision
						// prefix through the normal world projectile phase.
						shot.animation = visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: frame.Sprite}}}
						shot.animationState = NewAnimation(shot.animation)
						shot.Sprite = frame.Sprite
						if _, exists := w.movingSpriteBoxes[frame.Sprite]; !exists {
							t.Fatalf("original guardian frame %s lacks its exported collision prefix", frame.Sprite)
						}
						if err := w.Step(Input{}); err != nil {
							t.Fatal(err)
						}
						wantShield, wantActive := 39, false
						if mode == "normal" {
							wantShield = 35
						}
						if mode == "diving" {
							wantActive = true
						}
						if w.Equipment.Shield != wantShield || shot.Active != wantActive || !w.PlayerAlive {
							t.Fatalf("level %d %s frame %s %s: shield%d active%v want%d/%v", level, clip.ID, frame.Sprite, mode, w.Equipment.Shield, shot.Active, wantShield, wantActive)
						}
						if !wantActive && w.Pool.Slot(shot.Binding.Slot).ResourceTag != 4 {
							t.Fatal("original collided shot was not retired in the shared pool")
						}
						cases++
					}
				}
			}
		}
	}
	if len(unique) != 21 || cases != 69 {
		t.Fatalf("incomplete guardian bullet coverage: %d unique frames, %d phase cases", len(unique), cases)
	}
	t.Logf("All %d original guardian bullet frames passed %d normal/invulnerable/diving world cases.", len(unique), cases)
}
