package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestFourthPodNativeTraceOptional(t *testing.T) {
	art, atlas := fourthStageTestArt(t)
	images := map[string]visualassets.SpriteRegion{}
	for _, region := range atlas.Sprites {
		images[region.Name] = region
	}
	var state FourthPodState
	nativeCombatRows(t, "fourth-pod-trace.csv", func(v []int64) {
		if v[1] == 0 {
			x := 32
			if v[0] == 1 {
				x = 288
			}
			state = NewFourthPod(int(v[0]), x, 40, art)
		}
		event := state.Advance(art, int(v[2]), func(name string) visualassets.SpriteRegion { return images[name] })
		x, y := state.X, state.Y
		if event.ConvertToExplosion {
			x, y = event.ExplosionX, event.ExplosionY
		}
		if x != int(v[3]) || y != int(v[4]) || state.Converted != (v[7] != 0) || event.SpawnChild && (event.ChildX != int(v[8]) || event.ChildY != int(v[9]) || event.ChildHeading != int(v[10]) || event.ChildPath != int(v[11])) {
			t.Fatalf("fourth pod %v: state=%+v event=%+v", v, state, event)
		}
		if !event.ConvertToExplosion {
			expected := atlas.SourceSpriteNames[int(v[5])]
			actual := state.Animation.Sprite(art.Variants[state.Side*2].PodAnimation)
			if expected != actual || state.Animation.Remaining != int(v[6]) {
				t.Fatalf("pod art %v: %s expected%s", v, actual, expected)
			}
		}
		if (v[8] != 0) != event.SpawnChild {
			t.Fatalf("pod transform %v: %+v", v, event)
		}
	})
}

func TestFourthPodChildNativeTraceOptional(t *testing.T) {
	art, atlas := fourthStageTestArt(t)
	boxes := map[string]visualassets.CollisionBox{}
	for _, region := range atlas.Sprites {
		boxes[region.Name] = *region.Collision
	}
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	common, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	var sine [256]int8
	for i := range sine {
		sine[i] = int8(common[0x98f6+i])
	}
	var state FourthPodChild
	var random RandomState
	nativeCombatRows(t, "fourth-pod-child-trace.csv", func(v []int64) {
		if v[1] == 0 {
			x := 42
			if v[0] == 1 {
				x = 278
			}
			var err error
			state, err = NewFourthPodChild(int(v[0]), x, 50, art, ActorResidue{XFraction: 0x1234, YFraction: 0x5678})
			if err != nil {
				t.Fatal(err)
			}
			random = NewRandomState()
		}
		if err := state.Advance(art, &sine, random.Next, func(name string) visualassets.CollisionBox { return boxes[name] }); err != nil {
			t.Fatal(err)
		}
		angle := uint32(state.Motion.AngleFixed)
		raw := angle<<16 | angle>>16
		if state.Motion.X != int32(v[2]) || state.Motion.Y != int32(v[3]) || raw != uint32(v[4]) || state.Motion.AngularVelocity != int32(v[5]) || state.Motion.AngularAcceleration != int32(v[6]) || state.Motion.Budget != int(v[7]) || state.Motion.Remaining != int(v[8]) || state.Sprite(art) != atlas.SourceSpriteNames[int(v[9])] || v[10] >= 0 && state.Animation.Remaining != int(v[10]) || random.A != uint32(v[11]) || random.B != uint32(v[12]) || state.PathIndex != int(v[13]) {
			t.Fatalf("pod child %v: state=%+v random=%+v", v, state, random)
		}
	})
}
