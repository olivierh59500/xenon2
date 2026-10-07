package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestSecondGuardianMinionsNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "00FA00FE.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(data)
	if err != nil {
		t.Fatal(err)
	}
	artwork, _, err := visualassets.DecodeGuardianArt(2, data, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	config, err := SecondMinionConfigFromVisual(artwork.Visuals[0])
	if err != nil {
		t.Fatal(err)
	}
	var state SecondMinionState
	var random RandomState
	cases, frames := 0, 0
	nativeCombatRows(t, "guardian-second-minion-trace.csv", func(v []int64) {
		if v[1] == 0 {
			state = NewSecondGuardianMinion(160, 160, uint8(v[0]), config)
			random = NewRandomState()
			cases++
		}
		event := state.Advance(SecondMinionInput{Frame: uint64(v[1]), ScrollY: 40, PlayerX: 160, PlayerY: 100}, config, &random)
		phase := state.MinionPhase
		if state.Turret {
			phase = state.TargetHeading
		}
		if state.Turret != (v[2] != 0) || state.X != int(v[3]) || state.Y != int(v[4]) || state.Heading != uint8(v[5]) || phase != int(v[6]) || state.Animation.Remaining != int(v[7]) || state.Animation.Sprite(state.Clip(config)) != artwork.Atlas.SourceSpriteNames[int(v[8])] || state.FireAccumulator != uint8(v[9]) || random.A != uint32(v[10]) || random.B != uint32(v[11]) || event.Transform != (v[12] != 0) || event.Shot != (v[13] != 0) || event.RestoreTerrain != (v[18] != 0) {
			t.Fatalf("minion case%d frame%d: state%+v event%+v rng%+v native%v", v[0], v[1], state, event, random, v)
		}
		if event.Shot && (event.ShotX != int(v[14]) || event.ShotY != int(v[15]) || event.ShotSpeed != int(v[16]) || event.ShotDirection != uint8(v[17])) {
			t.Fatalf("turret shot differs: %+v native%v", event, v)
		}
		if event.Transform {
			state = event.TurretState
		}
		frames++
	})
	if cases != 8 || frames != 14400 {
		t.Fatalf("incomplete minion reference: %d cases, %d passes", cases, frames)
	}
	t.Logf("Compared %d initial minion directions across %d original passes", cases, frames)
}
