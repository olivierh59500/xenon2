package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestThirdChainNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "02020113.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(data)
	if err != nil {
		t.Fatal(err)
	}
	bank, err := visualassets.DecodeFixedSprites(3, data, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	art := bank.Third.Chain
	var state ThirdChainState
	var random RandomState
	passes := 0
	nativeCombatRows(t, "fixed-third-chain-trace.csv", func(v []int64) {
		variant, frame, index := int(v[0]), int(v[1]), int(v[2])
		if index == 0 {
			if frame == 0 {
				state = NewThirdChain(visualassets.FixedEncounter{X: 128, Y: 2850, Variant: variant}, 2700, art.Tails[variant])
				random = NewRandomState()
			}
			state.Advance(ThirdChainInput{PlayerY: 120}, art, &random)
		}
		part := state.Parts[index]
		sprite := art.Bodies[variant]
		if index == 7 {
			clip := art.Tails[variant]
			sprite = part.Animation.Sprite(clip)
		}
		if part.X != int(v[3]) || part.Y != int(v[4]) || state.Phase != int(v[5]) || state.Speed != int(v[6]) || state.AmplitudeOrCooldown != int(v[7]) || part.Visible != (v[8] != 0) || part.Animation.Remaining != int(v[9]) || sprite != bank.Atlas.SourceSpriteNames[int(v[10])] || random.A != uint32(v[11]) || random.B != uint32(v[12]) {
			t.Fatalf("chain variant%d frame%d part%d: %+v state%+v rng%+v native%v", variant, frame, index, part, state, random, v)
		}
		passes++
	})
	if passes != 6400 {
		t.Fatalf("incomplete chain proof: %d part passes", passes)
	}
}
