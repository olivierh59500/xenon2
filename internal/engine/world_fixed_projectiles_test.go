package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestWorldAimingProjectileExpiryCreatesCenteredExplosion(t *testing.T) {
	w := testWorld(t)
	clip := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "aiming", Duration: 0}}}
	art := &visualassets.FixedProjectileArtwork{Kind: "animated-aiming-projectile", Lifetime: 80,
		HeadingAnimations: make([]visualassets.ActorAnimation, 8)}
	for i := range art.HeadingAnimations {
		art.HeadingAnimations[i] = clip
	}
	w.Level.FixedSprites = &visualassets.FixedSprites{Projectile: art, Atlas: visualassets.SpriteAtlas{
		Sprites: []visualassets.SpriteRegion{{Name: "aiming", Width: 16, Height: 14, AnchorX: 7, AnchorY: 5}},
	}}
	w.commonAnimations["explosion-small"] = visualassets.NamedActorAnimation{Animation: visualassets.ActorAnimation{
		Frames: []visualassets.AnimationFrame{{Sprite: "explosion", Duration: 2}}, Ending: "remove",
	}}
	state := AnimatedAimingFixedProjectile{X: 70, Y: 90, Timer: 79, Animation: NewAnimation(clip)}
	actor := &WorldActor{ID: 10, Active: true, fixedAiming: &state}
	w.advanceFixedAimingActor(actor)
	if actor.Active || len(w.Actors) != 1 || w.Actors[0].Sprite != "explosion" || w.Actors[0].X != 71 || w.Actors[0].Y != 91 {
		t.Fatal("expired projectile must become a source-centered explosion without advancing its motion")
	}
	if w.SoundRequests[2] != "sampled-effect-05" {
		t.Fatal("expiry must request the small explosion's sampled sound")
	}
}
