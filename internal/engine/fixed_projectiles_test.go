package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func fixedProjectileTestArt() *visualassets.FixedProjectileArtwork {
	art := &visualassets.FixedProjectileArtwork{Kind: "animated-aiming-projectile", Lifetime: 80, ContactDamage: 4}
	for heading := range 8 {
		art.HeadingAnimations = append(art.HeadingAnimations, visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: fmt.Sprint(heading)}}})
	}
	return art
}

func TestAnimatedFixedProjectilePreservesSourceContactProbe(t *testing.T) {
	art := fixedProjectileTestArt()
	box := func(string) visualassets.CollisionBox {
		return visualassets.CollisionBox{X: -5, Y: -5, Width: 11, Height: 11}
	}
	image := func(string) visualassets.SpriteRegion { return visualassets.SpriteRegion{Width: 11, Height: 11} }
	state := AnimatedAimingFixedProjectile{X: 160, Y: 100, Direction: 2}
	events := state.Advance(art, FixedProjectileInputs{CanHitPlayer: true, PlayerBounds: CollisionRect{Left: 155, Top: 95, Right: 175, Bottom: 105}}, box, image)
	if state.Removed || events.PlayerDamage != 0 {
		t.Fatal("actor-rectangle overlap replaced the original displacement probe")
	}
	state = AnimatedAimingFixedProjectile{X: 160, Y: 100, Direction: 2}
	events = state.Advance(art, FixedProjectileInputs{CanHitPlayer: true, PlayerBounds: CollisionRect{Left: 4, Top: -1, Right: 6, Bottom: 1}}, box, image)
	if !state.Removed || events.PlayerDamage != 4 {
		t.Fatal("original point contact was lost")
	}
}

func TestAnimatedFixedProjectileExpiryPrecedesMotion(t *testing.T) {
	art := fixedProjectileTestArt()
	state := AnimatedAimingFixedProjectile{X: -80, Y: 250, Direction: 2, Timer: 78}
	box := func(string) visualassets.CollisionBox { return visualassets.CollisionBox{} }
	image := func(string) visualassets.SpriteRegion {
		return visualassets.SpriteRegion{AnchorX: 2, AnchorY: 3, Width: 12, Height: 9}
	}
	state.Advance(art, FixedProjectileInputs{}, box, image)
	if state.Removed || state.X != -75 || state.Y != 250 {
		t.Fatal("offscreen projectile was removed before its source lifetime")
	}
	events := state.Advance(art, FixedProjectileInputs{}, box, image)
	if !state.Removed || state.X != -75 || !events.Explosion || events.ExplosionX != -71 || events.ExplosionY != 251 {
		t.Fatalf("expiry moved or lost its image-center explosion: %+v %+v", state, events)
	}
}

func TestFixedProjectilesNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare original projectile callbacks")
	}
	banks := map[int]*visualassets.FixedSprites{}
	regions := map[int]map[string]visualassets.SpriteRegion{}
	for level, name := range map[int]string{3: "02020113", 5: "04820138"} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", name+".decoded"))
		if err != nil {
			t.Fatal(err)
		}
		terrain, err := visualassets.DecodeTerrain(data)
		if err != nil {
			t.Fatal(err)
		}
		banks[level], err = visualassets.DecodeFixedSprites(level, data, terrain.Palette)
		if err != nil {
			t.Fatal(err)
		}
		regions[level] = make(map[string]visualassets.SpriteRegion)
		for _, region := range banks[level].Atlas.Sprites {
			regions[level][region.Name] = region
		}
	}
	var turning TurningFixedProjectile
	var aiming AnimatedAimingFixedProjectile
	cases, frames := 0, 0
	nativeCombatRows(t, "fixed-projectile-trace.csv", func(v []int64) {
		level := int(v[0])
		bank, art := banks[level], banks[level].Projectile
		if v[2] == 0 {
			cases++
			event := FixedSpriteEvents{ShotCount: 1, ShotX: int(v[3]), ShotY: int(v[4]), ShotDelay: int(v[6]), ShotMotionBudget: art.MotionBudget, ShotDirections: [3]int{int(v[5])}}
			var err error
			if level == 3 {
				event.ShotSprite = art.TurningSprites[0]
				if event.ShotDirections[0] == 5 {
					event.ShotSprite = art.TurningSprites[1]
				}
				turning, err = NewTurningFixedProjectile(event, art, uint16(v[7]), uint16(v[8]))
			} else {
				aiming, err = NewAnimatedAimingFixedProjectile(event, art)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		input := FixedProjectileInputs{PlayerX: int(v[9]), PlayerY: int(v[10]), ScrollDelta: int(v[11]), CanHitPlayer: v[12] == 0, Invulnerable: v[13] != 0, PlayerBounds: CollisionRect{Left: int(v[14]), Top: int(v[15]), Right: int(v[16]), Bottom: int(v[17])}}
		var events FixedProjectileEvents
		var x, y, direction, timer int
		var sprite string
		var removed bool
		if level == 3 {
			var err error
			events, err = turning.Advance(art, input, *regions[level][turning.Sprite].Collision)
			if err != nil {
				t.Fatal(err)
			}
			x, y, direction, sprite, removed = int(turning.Motion.X), int(turning.Motion.Y), int(turning.Motion.Direction), turning.Sprite, turning.Removed
		} else {
			events = aiming.Advance(art, input, func(name string) visualassets.CollisionBox { return *regions[level][name].Collision }, func(name string) visualassets.SpriteRegion { return regions[level][name] })
			x, y, direction, timer, sprite, removed = aiming.X, aiming.Y, int(aiming.Direction), aiming.Timer, aiming.Sprite(art), aiming.Removed
		}
		expectedSprite := bank.Atlas.SourceSpriteNames[int(v[22])]
		if x != int(v[18]) || y != int(v[19]) || direction != int(v[20]) || timer != int(v[21]) || sprite != expectedSprite || removed != (v[23] == 0) || events.PlayerDamage != int(v[24]) || events.Explosion != (v[25] != 0) || events.Explosion && (events.ExplosionX != int(v[26]) || events.ExplosionY != int(v[27])) {
			t.Fatalf("fixed projectile %v: got x=%d y=%d dir=%d timer=%d sprite=%s expected=%s removed=%v events=%+v", v, x, y, direction, timer, sprite, expectedSprite, removed, events)
		}
		frames++
	})
	if cases != 56 || frames < 2000 {
		t.Fatalf("native projectile coverage unexpectedly short: %d cases %d frames", cases, frames)
	}
}
