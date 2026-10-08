package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func fifthPublishedBulletScene(t testing.TB) *World {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, 5))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames, w.ScrollDelta = false, 0, 1
	w.Player.X, w.Player.Y = 160, 171
	w.Equipment.ApplyItem(ItemSpeedup)
	w.Equipment.ApplyItem(ItemSpeedup)
	w.Player.SpeedTier = 2
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	// Isolate the actual bullet and ship callbacks from future encounter births.
	w.Level.Encounters = &visualassets.Encounters{}
	w.spawnEnemyShot(153, 152, EnemyShot{Direction: 4, Speed: 6})
	return w
}

func TestFifthPublishedBulletHitsBeforeAllNineMovementEndpointsOptional(t *testing.T) {
	w := fifthPublishedBulletScene(t)
	before := forecastIsolationDigest(w)
	shot := w.Projectiles[0]
	view, ok := demoEnemyShotPrediction(w, shot, 1, w.ScrollDelta)
	if !ok || !view.Active || view.X != 153 || view.Y != 159 || !thirdMiddlePlayerBounds(w, w.Player).Intersects(view.Bounds) {
		t.Fatalf("original impending bullet differs: %+v supported%v", view, ok)
	}
	// Moving right takes the new ship bounds away from the bullet. The source
	// still tests the already published prefix, so this is not an escape.
	endpoint := w.Player
	endpoint.Advance(MotionInput{Right: true}, MotionContext{BaseScrollStep: 1})
	if thirdMiddlePlayerBounds(w, endpoint).Intersects(view.Bounds) {
		t.Fatal("fixture does not distinguish the movement endpoint from the native prefix")
	}
	risk, _ := presentationMotionScore(w, MotionInput{Right: true}, 169, 171)
	if risk < 100000 {
		t.Fatalf("tactical movement still treats the right endpoint as a bullet escape: %g", risk)
	}
	for _, motion := range demoDirections {
		if !demoFifthEnemyShotContact(w, shot, 1, w.Player) {
			t.Fatal("published native ship prefix lost the original bullet overlap")
		}
		independent := fifthPublishedBulletScene(t)
		independentShot := independent.Projectiles[0]
		if err := independent.Step(Input{Motion: motion}); err != nil {
			t.Fatal(err)
		}
		if independent.Equipment.Shield != 35 || independentShot.Active || !independent.PlayerAlive {
			t.Fatalf("native sampled input changed the impending four-point hit: %+v HP%d", motion, independent.Equipment.Shield)
		}
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("prediction changed live bullet, animation, pool, ship or random state")
	}
}

func TestOrdinaryShotPredictionMatchesOriginalAnimationAndFractionsOptional(t *testing.T) {
	for heading := range uint8(8) {
		for _, delta := range []int{-2, 0, 1, 2} {
			w, err := NewWorld(originalWorldData(t, 5))
			if err != nil {
				t.Fatal(err)
			}
			w.PlayerAlive, w.ScrollDelta = false, delta
			w.spawnEnemyShot(100, 90, EnemyShot{Direction: heading, Speed: 6})
			shot := w.Projectiles[0]
			shot.Motion.X |= 32768
			shot.Motion.Y |= 16384
			shot.animation = w.Level.Rules.RadialTileShotAnimation
			shot.animationState = NewAnimation(shot.animation)
			shot.Sprite = shot.animationState.Sprite(shot.animation)
			if len(shot.animation.Frames) < 2 {
				t.Fatal("original radial bullet animation is missing")
			}
			var views [18]demoActorView
			before := forecastIsolationDigest(w)
			for i := range views {
				var supported bool
				views[i], supported = demoEnemyShotPrediction(w, shot, i+1, delta)
				if !supported {
					t.Fatal("ordinary original bullet prediction became unsupported")
				}
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("ordinary animation forecast changed source fields")
			}
			for i, view := range views {
				if shot.Active {
					if err := w.advanceEnemyShot(shot); err != nil {
						t.Fatal(err)
					}
				}
				if view.Active != shot.Active || view.Sprite != shot.Sprite || view.X != int(shot.X) || view.Y != int(shot.Y) {
					t.Fatalf("heading%d delta%d pass%d: predicted%+v native%+v", heading, delta, i+1, view, shot)
				}
				if shot.Active {
					box, exists := w.enemyShotCollisionBox(shot.Sprite, shot.Atlas)
					if !exists || view.Bounds != ActorCollisionRect(box, int(shot.X), int(shot.Y)) {
						t.Fatal("predicted animation selected different native collision bounds")
					}
				}
			}
		}
	}
}

func TestOrdinaryShotPredictionRetiresAtNativeEdgeAndRejectsOtherScopesOptional(t *testing.T) {
	w := fifthPublishedBulletScene(t)
	shot := w.Projectiles[0]
	shot.Motion.Y, shot.Motion.Direction = 1<<16, 0
	view, ok := demoEnemyShotPrediction(w, shot, 1, 0)
	if !ok || view.Active || !shot.Active || !view.Bounds.Empty() {
		t.Fatal("expired forecast projectile stayed collidable or retired its live source")
	}
	for _, change := range []func(*WorldProjectile){
		func(q *WorldProjectile) { q.turning = &TurningFixedProjectile{} },
		func(q *WorldProjectile) { q.Sprite = "unknown" },
	} {
		q := *shot
		change(&q)
		if _, ok := demoEnemyShotPrediction(w, &q, 0, 0); ok {
			t.Fatal("unsupported projectile controller or collision artwork was admitted")
		}
	}
	for _, passes := range []int{-1, 19} {
		if _, ok := demoEnemyShotPrediction(w, shot, passes, 0); ok {
			t.Fatal("unsupported ordinary projectile horizon was admitted")
		}
	}
	q := *w
	q.Level.Number = 4
	if demoFifthEnemyShotContact(&q, shot, 1, q.Player) {
		t.Fatal("fifth prefix correction changed another stage policy")
	}
}
