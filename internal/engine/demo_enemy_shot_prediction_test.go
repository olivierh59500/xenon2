package engine

import (
	"fmt"
	"testing"

	"xenon2/internal/visualassets"
)

func publishedBulletScene(t testing.TB, level int) *World {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, level))
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
	if level == 2 {
		// This level's bouncing attacker replaces the harmless default image
		// with its own recovered point-shot prefix after ordinary allocation.
		w.Projectiles[0].Sprite = w.Level.FixedSprites.Kinds[0].Variants[0].ShotSprite
		w.Projectiles[0].Atlas = "fixed"
	}
	return w
}

func TestPublishedBulletHitsBeforeAllNineMovementEndpointsAcrossFiveLevelsOptional(t *testing.T) {
	for level := 1; level <= 5; level++ {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			w := publishedBulletScene(t, level)
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
			if level == 5 {
				risk, _ := presentationMotionScore(w, MotionInput{Right: true}, 169, 171)
				if risk < 100000 {
					t.Fatalf("tactical movement still treats the right endpoint as a bullet escape: %g", risk)
				}
			}
			for _, motion := range demoDirections {
				if !demoPublishedEnemyShotContact(w, shot, 1, w.Player) {
					t.Fatal("published native ship prefix lost the original bullet overlap")
				}
				independent := publishedBulletScene(t, level)
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
		})
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
	w := publishedBulletScene(t, 5)
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
	q.Level.Number = 0
	if demoPublishedEnemyShotContact(&q, shot, 1, q.Player) {
		t.Fatal("unsupported world admitted ordinary projectile contact prediction")
	}
}

func TestOrdinaryShotPredictionFollowsActualCameraChangesAcrossFiveLevelsOptional(t *testing.T) {
	for level := 1; level <= 5; level++ {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			w := publishedBulletScene(t, level)
			// A narrow arranged scroll window reaches both source boundaries
			// within the short horizon, without a full-stage traversal.
			w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY, w.ScrollDeviationPasses = 4592, 4592, 4596, 4596, 34
			w.Player.X, w.Player.Y = 160, 176
			w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
			shot := w.Projectiles[0]
			shot.X, shot.Y, shot.PreviousX, shot.PreviousY = 20, 20, 20, 20
			shot.Motion = DirectionalProjectile{X: 20 << 16, Y: 20 << 16, Direction: 4, Speed: 2}
			initial := *shot
			var steps [18]int
			displacements := make(map[int]bool)
			for index := range steps {
				steps[index] = w.ScrollDelta
				displacements[w.ScrollDelta] = true
				before := forecastIsolationDigest(w)
				view, ok := demoEnemyShotPredictionSteps(w, &initial, steps[:index+1])
				if !ok || forecastIsolationDigest(w) != before {
					t.Fatal("variable-camera prediction changed live state or rejected an ordinary shot")
				}
				motion := MotionInput{Down: true}
				if index >= 9 && index < 13 {
					motion = MotionInput{Up: true}
				} else if index >= 13 {
					motion = MotionInput{Right: index&1 == 0, Left: index&1 != 0}
				}
				if err := w.Step(Input{Motion: motion}); err != nil {
					t.Fatal(err)
				}
				if !w.PlayerAlive || w.Rewind.Timer != 0 || view.Active != shot.Active || view.X != int(shot.X) || view.Y != int(shot.Y) || view.Sprite != shot.Sprite {
					t.Fatalf("camera pass %d differs: steps %v view %+v actual %v/%v active %v player alive %v rewind %d", index, steps[:index+1], view, shot.X, shot.Y, shot.Active, w.PlayerAlive, w.Rewind.Timer)
				}
			}
			if !displacements[-2] || !displacements[0] || !displacements[1] {
				t.Fatalf("camera fixture omitted doubled reverse, a clamp or ordinary scrolling: %v", displacements)
			}
			constant, ok := demoEnemyShotPrediction(w, &initial, len(steps), steps[0])
			if !ok || constant.Y == int(shot.Y) {
				t.Fatal("fixture no longer distinguishes the old repeated camera displacement")
			}
			if allocations := testing.AllocsPerRun(20, func() { demoEnemyShotPredictionSteps(w, &initial, steps[:]) }); allocations != 0 {
				t.Fatalf("variable-camera shot prediction allocates %.0f objects", allocations)
			}
		})
	}
}

func TestTurningShotPredictionKeepsPreTurnCollisionAndVariableScroll(t *testing.T) {
	w, shot, _ := turningShotConstructionFixture(t)
	w.PlayerAlive = false
	w.Level.FixedSprites.Projectile.TurningSprites = [2]string{"left", "right"}
	w.movingSpriteBoxes["turning"] = visualassets.CollisionBox{X: -7, Y: -5, Width: 15, Height: 11}
	w.movingSpriteBoxes["left"] = visualassets.CollisionBox{X: -1, Y: -1, Width: 3, Height: 3}
	w.movingSpriteBoxes["right"] = visualassets.CollisionBox{X: -2, Y: -2, Width: 5, Height: 5}
	initial, motion := *shot, *shot.turning
	frozen := motion
	initial.turning = &motion
	steps := []int{1, 0, -1, -2, 0, 1, 2, 0, -2, 1, 1, 0}
	for index, delta := range steps {
		before := *shot.turning
		view, ok := demoProjectilePredictionSteps(w, &initial, steps[:index+1])
		if !ok || *shot.turning != before || motion != frozen {
			t.Fatal("turning prediction changed the source controller or rejected its camera history")
		}
		oldBox := w.movingSpriteBoxes[shot.Sprite]
		w.ScrollDelta = delta
		if err := w.advanceEnemyShot(shot); err != nil {
			t.Fatal(err)
		}
		w.finishProjectileUpdate(shot)
		if view.Active != shot.Active || view.Sprite != shot.Sprite || view.X != int(shot.X) || view.Y != int(shot.Y) || view.Bounds != ActorCollisionRect(oldBox, int(shot.X), int(shot.Y)) {
			t.Fatalf("turning pass %d differs: view %+v actual %+v image %s", index, view, shot.turning, shot.Sprite)
		}
		if index == 0 && (view.Sprite != "left" || view.Bounds == ActorCollisionRect(w.movingSpriteBoxes[view.Sprite], view.X, view.Y)) {
			t.Fatal("fixture lost the source's collision-before-turn distinction")
		}
	}
	if allocations := testing.AllocsPerRun(20, func() { demoProjectilePredictionSteps(w, &initial, steps) }); allocations != 0 {
		t.Fatalf("turning prediction allocates %.0f objects", allocations)
	}
}
