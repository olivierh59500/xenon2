package engine

import (
	"reflect"
	"testing"
	"xenon2/internal/visualassets"
)

func TestDemoTurningShotPredictionKeepsLiveControllerAndSourceThresholds(t *testing.T) {
	w := testWorld(t)
	w.Level.FixedSprites = &visualassets.FixedSprites{Projectile: &visualassets.FixedProjectileArtwork{Kind: "turning-projectile", TurningSprites: [2]string{"right", "left"}}}
	for _, test := range []struct {
		x         int
		direction uint8
		want      [3][2]int
	}{
		{132, 5, [3][2]int{{129, 103}, {126, 107}, {129, 111}}},
		{191, 3, [3][2]int{{193, 103}, {191, 107}, {188, 111}}},
	} {
		state := TurningFixedProjectile{Motion: DirectionalProjectile{X: int32(test.x) << 16, Y: 100 << 16, Direction: test.direction, Speed: 4}, Sprite: "left"}
		shot := &WorldProjectile{Active: true, turning: &state}
		before := *shot
		beforeState := state
		for pass, want := range test.want {
			x, y, alive := demoProjectilePosition(w, shot, pass+1, 1)
			if !alive || x != want[0] || y != want[1] {
				t.Fatalf("origin%d pass%d predicted%d,%d alive%v; want%v", test.x, pass+1, x, y, alive, want)
			}
		}
		if !reflect.DeepEqual(*shot, before) || *shot.turning != beforeState || shot.Motion != (DirectionalProjectile{}) {
			t.Fatal("prediction advanced the live specialized or generic controller")
		}
	}
}

func TestDemoShotPredictionRetainsRemovalAndFractionalMotion(t *testing.T) {
	w := testWorld(t)
	shot := &WorldProjectile{Active: true, Motion: DirectionalProjectile{X: 100<<16 | 32768, Y: 100 << 16, Direction: 2, Speed: 4}}
	if x, y, alive := demoProjectilePosition(w, shot, 2, 1); !alive || x != 108 || y != 102 {
		t.Fatal("ordinary shot lost its fractional position or scenery scroll")
	}
	w.Level.FixedSprites = &visualassets.FixedSprites{Projectile: &visualassets.FixedProjectileArtwork{Kind: "turning-projectile"}}
	shot.turning = &TurningFixedProjectile{Motion: DirectionalProjectile{X: 100 << 16, Y: 1 << 16, Direction: 0, Speed: 4}}
	if _, _, alive := demoProjectilePosition(w, shot, 1, 0); alive {
		t.Fatal("a retired offscreen shot remained a future obstacle")
	}
	shot.Active = false
	if _, _, alive := demoProjectilePosition(w, shot, 0, 0); alive {
		t.Fatal("an inactive shot was admitted to prediction")
	}
}
