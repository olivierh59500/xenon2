package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestSecondDefenseWaveNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "00FA00FE.decoded"))
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
	groups, atlas, err := visualassets.DecodeCompoundGuardianArt(2, data, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := visualassets.DecodePaths(data, common)
	if err != nil {
		t.Fatal(err)
	}
	group := groups[0]
	var segment SecondDefenseSegment
	var gates [8]int
	var launch visualassets.GuardianLaunch
	var part visualassets.GuardianComponent
	cases, frames := 0, 0
	nativeCombatRows(t, "guardian-second-wave-trace.csv", func(v []int64) {
		if v[2] == 0 {
			launch = group.Launches[v[0]]
			part = group.Components[v[1]]
			segment, err = NewSecondDefenseSegment(launch, part, 0, 2700, 1)
			if err != nil {
				t.Fatal(err)
			}
			gates = [8]int{}
			cases++
		}
		event, err := segment.Advance(&launch.Path, &paths.SineTable, 1, &gates, nil)
		if err != nil {
			t.Fatal(err)
		}
		angle := uint32(segment.Motion.AngleFixed)
		nativeAngle := angle<<16 | angle>>16
		if uint32(segment.Motion.X) != uint32(v[3]) || uint32(segment.Motion.Y) != uint32(v[4]) || nativeAngle != uint32(v[5]) || uint32(segment.Motion.AngularVelocity) != uint32(v[6]) || int(segment.Motion.AngularAcceleration) != int(v[7]) || segment.Motion.Remaining != int(v[8]) || segment.Removed != (v[12] != 0) || gates[segment.GateID-1] != int(v[13]) {
			t.Fatalf("launch%d part%d frame%d: %+v native%v", v[0], v[1], v[2], segment, v)
		}
		if !segment.Removed {
			code := int64(0xe48)
			if segment.Presentation == "materializing" {
				code = 0xeba
			}
			if segment.Presentation == "normal" {
				code = 0xe12
			}
			if code != v[11] || atlas.SourceSpriteNames[int(v[10])] != part.HeadingFrames[event.HeadingFrame] {
				t.Fatalf("wave presentation differs: %+v native%v", segment, v)
			}
		}
		frames++
	})
	if cases != 192 || frames != 16890 {
		t.Fatalf("incomplete wave comparison: %d cases, %d passes", cases, frames)
	}
	t.Logf("Compared all %d launch/member combinations across %d original passes", cases, frames)
}

func TestSecondDefenseSchedulerIdleStreams(t *testing.T) {
	scheduler := NewSecondDefenseScheduler()
	launches := scheduler.LaunchIdleStreams([2]bool{}, 3)
	if len(launches) != 2 || scheduler.InitialWaves != 0 || scheduler.DefenseFlags != 3 {
		t.Fatalf("both source streams need their primary entry launches: %+v %+v", scheduler, launches)
	}
	before := scheduler
	if len(scheduler.LaunchIdleStreams([2]bool{true, true}, 3)) != 0 || scheduler != before {
		t.Fatal("active streams must not consume launch seeds")
	}
	launches = scheduler.LaunchIdleStreams([2]bool{false, true}, 3)
	if len(launches) != 1 || launches[0].Stream != 0 || launches[0].LaunchIndex < 8 || scheduler.Seeds[1] != before.Seeds[1] {
		t.Fatalf("only the idle stream advances its secondary launch bank: %+v %+v", scheduler, launches)
	}
	scheduler.UseFinalLaunches()
	launches = scheduler.LaunchIdleStreams([2]bool{}, 1)
	for _, launch := range launches {
		if launch.LaunchIndex >= 8 {
			t.Fatal("last defense must use the original primary exit bank")
		}
	}
}

func TestSecondDefenseSchedulerNativeTraceOptional(t *testing.T) {
	var scheduler SecondDefenseScheduler
	cases, passes := 0, 0
	nativeCombatRows(t, "guardian-second-scheduler-trace.csv", func(v []int64) {
		if v[1] == 0 {
			scheduler = NewSecondDefenseScheduler()
			if v[0] == 1 {
				scheduler.UseFinalLaunches()
			}
			cases++
		}
		updated := [2]bool{true, true}
		updated[v[2]] = false
		launches := scheduler.LaunchIdleStreams(updated, int(v[3]))
		if len(launches) != 1 || scheduler.InitialWaves != int(v[4]) || scheduler.Seeds[0] != uint32(v[5]) || scheduler.Seeds[1] != uint32(v[6]) || scheduler.DefenseFlags != uint8(v[7]) || launches[0].LaunchIndex != int(v[8]) {
			t.Fatalf("launch scheduler differs: %+v %+v native%v", scheduler, launches, v)
		}
		passes++
	})
	if cases != 2 || passes != 128 {
		t.Fatalf("incomplete launch comparison: %d cases, %d passes", cases, passes)
	}
}

func TestSecondDefenseFragmentsNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "00FA00FE.decoded"))
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
	groups, atlas, err := visualassets.DecodeCompoundGuardianArt(2, data, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := visualassets.DecodePaths(data, common)
	if err != nil {
		t.Fatal(err)
	}
	parts := []visualassets.GuardianComponent{groups[0].Components[0], groups[0].Components[1], groups[0].Components[11]}
	var state SecondDefenseFragment
	var animation visualassets.ActorAnimation
	cases, frames := 0, 0
	nativeCombatRows(t, "guardian-second-fragment-trace.csv", func(v []int64) {
		if v[2] == 0 {
			animation = parts[v[0]].DeathAnimation
			state = NewSecondDefenseFragment(160, 96, uint8(v[1]), animation)
			cases++
		}
		state.Advance(animation, &paths.SineTable)
		if state.X != int(v[3]) || state.Y != int(v[4]) || state.Animation.Remaining != int(v[5]) || state.Animation.Sprite(animation) != atlas.SourceSpriteNames[int(v[6])] || state.Removed != (v[7] != 0) {
			t.Fatalf("fragment differs: %+v native%v", state, v)
		}
		frames++
	})
	if cases != 48 || frames < 300 {
		t.Fatalf("incomplete fragments: %d cases %d passes", cases, frames)
	}
	t.Logf("Compared %d original fragment directions across %d passes", cases, frames)
}

func TestSecondDefenseBreakupChangesOnlyHeadStreamFlag(t *testing.T) {
	random := NewRandomState()
	before := random
	result := DamageSecondDefenseSegment(50, 10, true, 0, 3, &random)
	if result.Destroyed || result.Health != 40 || result.DefenseFlags != 3 || random != before {
		t.Fatalf("a partial hit must retain its defense stream: %+v", result)
	}
	result = DamageSecondDefenseSegment(50, 50, false, 0, 3, &random)
	if !result.Destroyed || result.DefenseFlags != 3 || result.Health != 50 || random == before {
		t.Fatalf("a broken link must retain the stream and its source health word: %+v", result)
	}
	result = DamageSecondDefenseSegment(50, 50, true, 1, 3, &random)
	if !result.Destroyed || result.DefenseFlags != 1 {
		t.Fatalf("a broken head must clear only its own stream flag: %+v", result)
	}
}
