package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func fourthGuardianTestArt(t *testing.T) ([]visualassets.GuardianGroup, visualassets.SpriteAtlas, [256]int8) {
	t.Helper()
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local fourth guardian traces")
	}
	level, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "031F0159.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(level)
	if err != nil {
		t.Fatal(err)
	}
	groups, atlas, err := visualassets.DecodeCompoundGuardianArt(4, level, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	common, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	var sine [256]int8
	for i := range sine {
		sine[i] = int8(common[0x98f6+i])
	}
	return groups, atlas, sine
}

func TestFourthFinalGuardianNativeTraceOptional(t *testing.T) {
	groups, atlas, sine := fourthGuardianTestArt(t)
	art := &groups[1]
	state, err := NewFourthFinalGuardian(art, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	boxes := map[string]visualassets.CollisionBox{}
	for _, region := range atlas.Sprites {
		if region.Collision != nil {
			boxes[region.Name] = *region.Collision
		}
	}
	random := NewRandomState()
	maximum, cases := 0, 0
	nativeCombatRows(t, "guardian-fourth-final-trace.csv", func(v []int64) {
		index := int(v[1])
		event, err := state.AdvancePart(index, art, FourthGuardianInput{Frame: uint64(v[0]), ScrollY: int(v[2]), ScrollDelta: int(v[3]), PlayerX: int(v[4]), PlayerY: int(v[5]), MaximumScrollY: maximum}, &sine, func(name string) visualassets.CollisionBox { return boxes[name] }, random.Next)
		if err != nil {
			t.Fatal(err)
		}
		maximum = event.MaximumScrollY
		part := state.Parts[index]
		angle := uint32(part.Arc.AngleFixed)
		nativeAngle := angle<<16 | angle>>16
		if maximum != int(v[6]) || part.Arc.X != int32(v[7]) || part.Arc.Y != int32(v[8]) || nativeAngle != uint32(v[9]) || part.Arc.AngularVelocity != int32(v[10]) || part.Arc.AngularAcceleration != int32(v[11]) || part.Arc.Budget != int(v[12]) || part.Counter != int16(v[13]) || part.FireAccumulator != uint8(v[14]) || (index == 1 || index == 2) && part.Direction != int16(v[15]) || part.Collision.Left != int(v[16]) || part.Collision.Top != int(v[17]) || part.Collision.Right != int(v[18]) || part.Collision.Bottom != int(v[19]) || event.ShotCount != int(v[20]) || state.EyesRemaining != int(v[25]) || state.BodyScrollDelta != int(v[26]) || event.WarningSound != (v[27] != 0) || random.A != uint32(v[28]) || random.B != uint32(v[29]) {
			t.Fatalf("fourth final %v: part=%+v maximum=%d event=%+v random=%+v", v, part, maximum, event, random)
		}
		if event.ShotCount != 0 {
			shot := event.Shots[0]
			if shot.X != int(v[21]) || shot.Y != int(v[22]) || shot.Direction != uint8(v[23]) || shot.Speed != int(v[24]) {
				t.Fatalf("fourth eye shot %v: %+v", v, shot)
			}
		}
		cases++
	})
	if cases != 600*19 {
		t.Fatalf("fourth final coverage: %d states", cases)
	}
}

func TestFourthFinalGuardianDamageNativeTraceOptional(t *testing.T) {
	groups, _, _ := fourthGuardianTestArt(t)
	state, err := NewFourthFinalGuardian(&groups[1], 16, nil)
	if err != nil {
		t.Fatal(err)
	}
	nativeCombatRows(t, "guardian-fourth-final-damage-trace.csv", func(v []int64) {
		event := state.Strike(int(v[1]), uint16(v[2]), 16)
		if state.Parts[0].Health != uint16(v[3]) || state.Parts[1].Health != uint16(v[4]) || state.Parts[2].Health != uint16(v[5]) || state.EyesRemaining != int(v[6]) || state.Parts[1].Direction != int16(v[7]) || state.Parts[2].Direction != int16(v[8]) || event.CashPairs != int(v[9]) || event.ExplosionCount+boolCount(event.Explosion) != int(v[10]) || event.AdvanceLevel != (v[13] != 0) {
			t.Fatalf("fourth final damage %v: state=%+v event=%+v", v, state, event)
		}
		if event.Explosion && (event.ExplosionX != int(v[11]) || event.ExplosionY != int(v[12])) {
			t.Fatalf("eye explosion %v: %+v", v, event)
		}
		if event.ExplosionCount != 0 && (event.ExplosionRectangle.Left != int(v[11]) || event.ExplosionRectangle.Top != int(v[12])) {
			t.Fatalf("body explosion %v: %+v", v, event)
		}
	})
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}
