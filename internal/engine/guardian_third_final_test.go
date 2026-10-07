package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestThirdFinalWormNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "02020113.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	common, err := os.ReadFile(filepath.Join(filepath.Dir(root), "XenonII-unpacked.bin"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(data)
	if err != nil {
		t.Fatal(err)
	}
	groups, atlas, err := visualassets.DecodeCompoundGuardianArt(3, data, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := visualassets.DecodePaths(data, common)
	if err != nil {
		t.Fatal(err)
	}
	group := groups[1]
	var states [11]ThirdFinalMember
	var path *visualassets.Path
	cases, frames := 0, 0
	nativeCombatRows(t, "guardian-third-final-trace.csv", func(v []int64) {
		launch, part, frame := int(v[0]), int(v[1]), int(v[2])
		if frame == 0 {
			if part == 0 {
				path = &paths.Paths[group.Launches[launch].PathID-1]
				cases++
			}
			states[part], err = NewThirdFinalMember(path, group.Components[part], 160, 1)
			if err != nil {
				t.Fatal(err)
			}
		}
		state := &states[part]
		event, err := state.Advance(path, group.Components[part], &paths.SineTable, ThirdFinalInput{ScrollDelta: 1, FireRate: uint8(group.MotionParameters["fire_rate"]), ShotSpeed: group.MotionParameters["shot_speed"]}, nil)
		if err != nil {
			t.Fatal(err)
		}
		angle := uint32(state.Motion.AngleFixed)
		angle = angle<<16 | angle>>16
		if uint32(state.Motion.X) != uint32(v[3]) || uint32(state.Motion.Y) != uint32(v[4]) || angle != uint32(v[5]) || uint32(state.Motion.AngularVelocity) != uint32(v[6]) || int16(state.Motion.AngularAcceleration) != int16(v[7]) || state.Motion.Remaining != int(v[8]) || state.Sprite != atlas.SourceSpriteNames[int(v[9])] || state.Animation.Remaining != int(v[10]) || state.FireAccumulator != uint8(v[11]) || state.Removed != (v[12] != 0) || event.ShotCount != int(v[13]) {
			t.Fatalf("final launch%d part%d frame%d: state%+v event%+v native%v (%s)", launch, part, frame, state, event, v, atlas.SourceSpriteNames[int(v[9])])
		}
		if event.ShotCount > 0 {
			shot := event.Shots[event.ShotCount-1]
			if shot.X != int(v[14]) || shot.Y != int(v[15]) || shot.Speed != int(v[16]) || shot.Direction != uint8(v[17]) {
				t.Fatalf("final radial emission differs: %+v native%v", shot, v)
			}
		}
		frames++
	})
	if cases != 8 || frames < 10000 {
		t.Fatalf("incomplete final reference: %d cases, %d passes", cases, frames)
	}
	t.Logf("Compared all eight final launch paths across %d original member passes", frames)
}

func TestThirdStageBoundaryNativeTraceOptional(t *testing.T) {
	var state ThirdStageState
	minimum, maximum, passes := 2800, 4607, 0
	nativeCombatRows(t, "guardian-third-stage-trace.csv", func(v []int64) {
		if v[1] == 0 {
			state = ThirdStageState{}
			minimum, maximum = 2800, 4607
		}
		requested := int(v[0]) - 1
		event := state.Advance(ThirdStageInput{ScrollY: int(v[2]), MinimumScrollY: minimum, MaximumScrollY: maximum, RequestedStep: requested, MiddleUpdated: v[3] != 0, FinalUpdated: v[4] != 0, FinalDefeated: v[5] != 0})
		minimum, maximum = event.MinimumScrollY, event.MaximumScrollY
		if event.HoldRequestedStep {
			requested = 0
		}
		if minimum != int(v[6]) || maximum != int(v[7]) || requested != int(v[8]) || event.LaunchFinal != (v[9] != 0) || event.FinalCash != (v[10] != 0) || state.MiddleHeartbeat != (v[11] != 0) {
			t.Fatalf("stage case%d frame%d: state%+v event%+v native%v", v[0], v[1], state, event, v)
		}
		passes++
	})
	if passes != 60 {
		t.Fatalf("incomplete stage comparison: %d passes", passes)
	}
}

func TestThirdFinalLaunchesExcludeEndpointSentinels(t *testing.T) {
	state := NewThirdFinalState(20)
	first, err := state.SelectLaunch()
	if err != nil || first != 0 {
		t.Fatal("first source launch must be zero")
	}
	for range 128 {
		choice, err := state.SelectLaunch()
		if err != nil || choice < 1 || choice > 6 {
			t.Fatalf("subsequent launch must be1..6: %d %v", choice, err)
		}
	}
	if state.Strike(19) || state.Health != 1 {
		t.Fatal("partial final damage must preserve its shared health")
	}
	if !state.Strike(1) || !state.Defeated {
		t.Fatal("lethal final hit must close its source stream")
	}
	if _, err := state.SelectLaunch(); err == nil {
		t.Fatal("defeated stream must not launch again")
	}
}
