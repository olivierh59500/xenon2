package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

// The native factory creates heads in sequence, then starts voice zero directly
// while retaining the two requests consumed by the next audio interrupt.
func TestWorldRandomExplosionsMatchNativeFactoryOptional(t *testing.T) {
	cases := [][5]int{{1, 0, 0, 320, 192}, {20, 0, 0, 320, 192}, {30, 80, 32, 160, 112}, {40, 112, -80, 96, 160}}
	worlds := make([]*World, len(cases))
	for i, parameters := range cases {
		w := testWorld(t)
		w.commonAnimations["explosion-large"] = visualassets.NamedActorAnimation{Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "large"}}}}
		w.spawnSecondRandomExplosions(parameters[0], parameters[1], parameters[2], parameters[3], parameters[4])
		worlds[i] = w
	}
	rows := 0
	nativeCombatRows(t, "random-explosion-trace.csv", func(v []int64) {
		w := worlds[v[0]]
		actor := w.Actors[v[1]]
		if w.Pool.Slot(actor.Binding.Slot).ResourceTag != int16(v[2]) || actor.X != float64(v[3]) || actor.Y != float64(v[4]) || w.random.A != uint32(v[5]) || w.random.B != uint32(v[6]) {
			t.Fatalf("explosion factory state differs: %v, actor=%+v random=%+v", v, actor, w.random)
		}
		if v[7] != 131 || v[8] != 131 || v[9] != 131 || v[10] != 0 || w.SoundRequests[1] != "sampled-effect-03" || w.SoundRequests[2] != "sampled-effect-03" || w.ImmediateSoundRequests[0] != "sampled-effect-03" {
			t.Fatalf("explosion factory audio differs: %v queued=%v immediate=%v", v, w.SoundRequests, w.ImmediateSoundRequests)
		}
		rows++
	})
	if rows != 91 {
		t.Fatalf("incomplete native explosion comparisons: %d", rows)
	}
}
