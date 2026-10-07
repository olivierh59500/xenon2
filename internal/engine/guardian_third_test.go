package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestThirdMiddleGuardianNativeTraceOptional(t *testing.T) {
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
	group := &groups[0]
	state, err := NewThirdGuardianState(group)
	if err != nil {
		t.Fatal(err)
	}
	random := NewRandomState()
	var event ThirdGuardianEvents
	frames := 0
	nativeCombatRows(t, "guardian-third-middle-trace.csv", func(v []int64) {
		index := int(v[1])
		part := state.Parts[index]
		if index == 0 {
			state.EyeHealth = [2]uint16{uint16(v[4]), uint16(v[5])}
			event, err = state.Advance(group, &paths.SineTable, uint64(v[0]), int(v[2]), int(v[3]), &random)
			if err != nil {
				t.Fatal(err)
			}
			part = state.Parts[index]
		}
		part = state.Parts[index]
		x, y, angle, velocity, acceleration, budget := part.Arc.X, part.Arc.Y, part.Arc.AngleFixed, part.Arc.AngularVelocity, part.Arc.AngularAcceleration, part.Arc.Budget
		behavior := group.Components[index].Behavior
		if index == 0 {
			x, y, angle, velocity, acceleration, budget = state.Flight.X, state.Flight.Y, state.Flight.AngleFixed, state.Flight.AngularVelocity, int32(state.Flight.AngularAcceleration), state.Flight.Budget
		}
		if behavior == "follow-head" || behavior == "eye-turret" || behavior == "arm-base" {
			x, y = int32(part.X)<<16, int32(part.Y)<<16
		}
		rawAngle := uint32(angle)
		rawAngle = rawAngle<<16 | rawAngle>>16
		if uint32(x) != uint32(v[6]) || uint32(y) != uint32(v[7]) || rawAngle != uint32(v[8]) || uint32(velocity) != uint32(v[9]) || int16(acceleration) != int16(v[10]) || budget != int(v[11]) || part.Extension != int(v[12]) || part.ExtensionDirection != int(v[13]) || part.FireAccumulator != uint8(v[14]) || part.Animation.Remaining != int(v[15]) || part.Sprite != atlas.SourceSpriteNames[int(v[16])] {
			t.Fatalf("middle frame%d part%d: %+v flight%+v native%v (%s)", v[0], index, part, state.Flight, v, atlas.SourceSpriteNames[int(v[16])])
		}
		if index == 16 && (random.A != uint32(v[17]) || random.B != uint32(v[18])) {
			t.Fatalf("middle random stream differs: %+v native%v", random, v)
		}
		if index == 0 {
			if len(event.Shots) != int(v[19]) {
				t.Fatalf("middle shot count differs: %+v native%v", event, v)
			}
			if len(event.Shots) > 0 {
				shot := event.Shots[len(event.Shots)-1]
				if shot.X != int(v[20]) || shot.Y != int(v[21]) || shot.Speed != int(v[22]) || shot.Direction != uint8(v[23]) {
					t.Fatalf("middle shot differs: %+v native%v", shot, v)
				}
			}
		}
		frames++
	})
	if frames != 20400 {
		t.Fatalf("incomplete middle reference: %d part passes", frames)
	}
	t.Logf("Compared all 17 parts across 1,200 original guardian passes")
}
