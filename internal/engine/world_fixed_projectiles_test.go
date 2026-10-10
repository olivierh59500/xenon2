package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func turningShotConstructionFixture(t *testing.T) (*World, *WorldProjectile, ActorResidue) {
	t.Helper()
	w := testWorld(t)
	w.Level.Rules = &visualassets.LevelRules{DefaultEnemyShot: "point"}
	w.Level.FixedSprites = &visualassets.FixedSprites{Projectile: &visualassets.FixedProjectileArtwork{Kind: "turning-projectile", ResourceTag: 20}}
	retained := waveConstructorResidueFixture(w.Pool.FreeFirst())
	w.Pool.Slot(w.Pool.FreeFirst()).Residue = retained
	event := FixedSpriteEvents{ShotMode: "turning-projectile", ShotCount: 1, ShotX: 96, ShotY: 58, ShotSprite: "turning", ShotMotionBudget: 6}
	event.ShotDirections[0] = 3
	if !w.spawnSpecializedFixedShot(event) || w.poolError != nil || len(w.Projectiles) != 1 {
		t.Fatalf("turning projectile construction failed: %v", w.poolError)
	}
	return w, w.Projectiles[0], retained
}

func TestTurningFixedShotPublishesMotionAndRetainsUnassignedWords(t *testing.T) {
	w, shot, retained := turningShotConstructionFixture(t)
	want := retained
	want.X, want.Y, want.Counter, want.Direction, want.MotionBudget = 96, 58, 0, 3, 6
	if got := w.Pool.Slot(shot.Binding.Slot).Residue; got != want {
		t.Fatalf("new turning-shot state %v, want %v", got, want)
	}
}

func TestTurningFixedShotReclaimedBeforeFirstUpdateKeepsInitializedWords(t *testing.T) {
	w, shot, retained := turningShotConstructionFixture(t)
	slot, identity := shot.Binding.Slot, shot.ID
	fillWaveDamageMovingTail(t, w)
	w.spawnEnemyShot(160, 120, EnemyShot{Direction: 5, Speed: 7})
	current := w.Pool.Slot(slot)
	if w.poolError != nil || shot.Active || current.EntityID == identity || w.Projectiles[0].Binding.Slot != slot {
		t.Fatalf("ordinary point shot did not reclaim the unadvanced turning entry: %v", w.poolError)
	}
	want := retained
	want.X, want.Y, want.Counter, want.Direction, want.MotionBudget = 160, 120, 0, 5, 7
	if current.Residue != want {
		t.Fatalf("point-shot reuse inherited stale turning constructor fields: %v, want %v", current.Residue, want)
	}
}

func aimingShotConstructionFixture(t *testing.T, heading int) (*World, *WorldActor, ActorResidue) {
	t.Helper()
	w := testWorld(t)
	w.Level.Number = 5
	w.ScrollDelta = 0
	w.Level.Rules = &visualassets.LevelRules{DefaultEnemyShot: "point"}
	art := fixedProjectileTestArt()
	art.ResourceTag, art.Health, art.Score = 228, 6, 100
	w.Level.FixedSprites = &visualassets.FixedSprites{Projectile: art}
	w.playerCollision = CollisionRect{Left: 1000, Top: 1000, Right: 1020, Bottom: 1020}
	retained := waveConstructorResidueFixture(w.Pool.FreeFirst())
	w.Pool.Slot(w.Pool.FreeFirst()).Residue = retained
	event := FixedSpriteEvents{ShotMode: "animated-aiming-projectile", ShotCount: 1, ShotX: 96, ShotY: 58, ShotDelay: 30}
	event.ShotDirections[0] = heading
	if !w.spawnSpecializedFixedShot(event) || w.poolError != nil || len(w.Actors) != 1 {
		t.Fatalf("aiming projectile construction failed: %v", w.poolError)
	}
	return w, w.Actors[0], retained
}

func TestAimingFixedShotPublishesHealthHeadingAndTimerOnCreation(t *testing.T) {
	for heading := range 8 {
		w, actor, retained := aimingShotConstructionFixture(t, heading)
		want := retained
		want.X, want.Y, want.Counter, want.Direction = 96, 58, 30, int16(heading)
		want.Health, want.PowerOrScore, want.WaveBonusToken, want.StrongHealth = 6, 100, 0, false
		if got := w.Pool.Slot(actor.Binding.Slot).Residue; got != want {
			t.Fatalf("heading %d initialized fields %v, want %v", heading, got, want)
		}
	}
}

func TestAimingFixedShotUpdateRetainsFractionsAndPublishesLifetime(t *testing.T) {
	w, actor, retained := aimingShotConstructionFixture(t, 2)
	w.damageActor(actor, 1)
	w.advanceFixedAimingActor(actor)
	w.finishActorUpdate(actor)
	want := retained
	want.X, want.Y, want.Counter, want.Direction = 101, 58, 31, 2
	want.Health, want.PowerOrScore, want.WaveBonusToken, want.StrongHealth = 5, 100, 0, false
	if got := w.Pool.Slot(actor.Binding.Slot).Residue; got != want || !actor.Active {
		t.Fatalf("aiming update lost retained words or restored damaged health: %v, want %v", got, want)
	}
}

func TestAimingFixedShotReclaimedBeforeFirstUpdateKeepsInitializedWords(t *testing.T) {
	w, actor, retained := aimingShotConstructionFixture(t, 2)
	slot, identity := actor.Binding.Slot, actor.ID
	fillWaveDamageMovingTail(t, w)
	w.spawnEnemyShot(160, 120, EnemyShot{Direction: 5, Speed: 7})
	current := w.Pool.Slot(slot)
	if w.poolError != nil || actor.Active || current.EntityID == identity || w.Projectiles[0].Binding.Slot != slot {
		t.Fatalf("ordinary point shot did not reclaim the unadvanced aiming entry: %v", w.poolError)
	}
	want := retained
	want.X, want.Y, want.Counter, want.Direction, want.MotionBudget = 160, 120, 30, 5, 7
	want.Health, want.PowerOrScore, want.WaveBonusToken, want.StrongHealth = 6, 100, 0, false
	if current.Residue != want {
		t.Fatalf("point-shot reuse inherited stale aiming constructor fields: %v, want %v", current.Residue, want)
	}
}

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
