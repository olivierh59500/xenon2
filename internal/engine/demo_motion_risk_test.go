package engine

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"testing"

	"xenon2/internal/visualassets"
)

func originalDemoRouteRiskWorld(t testing.TB) *World {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	var wave visualassets.Wave
	for _, candidate := range w.Level.Encounters.Moving {
		if candidate.PathID == 33 {
			wave = candidate
			break
		}
	}
	if wave.Count == 0 {
		t.Fatal("original third-stage path33 wave is missing")
	}
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = wave.TriggerY, wave.TriggerY+16, wave.TriggerY+16
	w.Player.X, w.Player.Y = 160, 136
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.Ready, w.MaterializationFrames = false, 0
	w.Level.Encounters = &visualassets.Encounters{}
	if err := w.spawnWave(wave); err != nil {
		t.Fatal(err)
	}
	if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
	w.spawnEnemyShot(152, 112, EnemyShot{Direction: 3, Speed: 6})
	w.spawnEnemyShot(176, 156, EnemyShot{Direction: 7, Speed: 6})
	supported := false
	for _, actor := range w.Actors {
		view, ok := demoActorPrediction(w, actor, 6, w.ScrollY)
		supported = supported || ok && view.Active && !view.Bounds.Empty()
	}
	if !supported || len(w.Projectiles) != 2 || w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("benchmark needs a supported active source collider, two shots and clear terrain start")
	}
	return w
}

func TestOriginalDemoRouteRiskCacheDecisionsAndReadOnlyOptional(t *testing.T) {
	w := originalDemoRouteRiskWorld(t)
	hash := sha256.New()
	count := 0
	for _, cameraChange := range []int{-16, -8, 0, 8} {
		for _, x := range []int{48, 80, 112, 160, 208, 240, 272} {
			for _, y := range []int{16, 64, 136, 176} {
				for _, avoid := range []bool{false, true} {
					w.Player.X, w.Player.Y = x, y
					w.ScrollY += cameraChange
					w.Rewind = NewTerrainRewind(w.ScrollY, x, y)
					actors := make([]WorldActor, len(w.Actors))
					for index, actor := range w.Actors {
						actors[index] = *actor
					}
					shots := make([]WorldProjectile, len(w.Projectiles))
					for index, shot := range w.Projectiles {
						shots[index] = *shot
					}
					pool, random, player, camera, rewind := *w.Pool, w.RandomState(), w.Player, w.ScrollY, w.Rewind
					input := demoRouteMotionWithOptions(w, 224, w.ScrollY+96, 136, avoid)
					code := byte(0)
					for bit, held := range []bool{input.Up, input.Down, input.Left, input.Right} {
						if held {
							code |= 1 << bit
						}
					}
					hash.Write([]byte{code})
					if *w.Pool != pool || w.RandomState() != random || w.Player != player || w.ScrollY != camera || w.Rewind != rewind {
						t.Fatal("route forecast changed live world state")
					}
					for index, actor := range w.Actors {
						if !reflect.DeepEqual(*actor, actors[index]) {
							t.Fatal("route forecast changed live actor state")
						}
					}
					for index, shot := range w.Projectiles {
						if !reflect.DeepEqual(*shot, shots[index]) {
							t.Fatal("route forecast changed live projectile state")
						}
					}
					w.ScrollY -= cameraChange
					count++
				}
			}
		}
	}
	const original = "654a995298d701ecbfcba21afd5b414821b933ca786e5672342f117a6e941ccc"
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != original || count != 224 {
		t.Fatalf("original source route decisions: count%d hash%s want%s", count, got, original)
	}
	if allocations := testing.AllocsPerRun(20, func() { demoRouteMotionWithOptions(w, 224, w.ScrollY+96, 136, true) }); allocations != 0 {
		t.Fatalf("route risk cache allocates %.0f objects per call", allocations)
	}
}

func BenchmarkDemoRouteRiskOriginal(b *testing.B) {
	w := originalDemoRouteRiskWorld(b)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		demoRouteMotionWithOptions(w, 224, w.ScrollY+96, 136, true)
	}
}
