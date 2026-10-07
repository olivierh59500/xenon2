package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func fourthStageTestArt(t *testing.T) (*visualassets.FourthStageArt, visualassets.SpriteAtlas) {
	t.Helper()
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR for local fourth stage proofs")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "031F0159.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(data)
	if err != nil {
		t.Fatal(err)
	}
	bank, err := visualassets.DecodeFixedSprites(4, data, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	if bank.FourthStage == nil {
		t.Fatal("fourth stage metadata missing")
	}
	return bank.FourthStage, bank.Atlas
}

func TestFourthFallingActorsNativeTraceOptional(t *testing.T) {
	art, atlas := fourthStageTestArt(t)
	boxes := map[string]visualassets.CollisionBox{}
	for _, region := range atlas.Sprites {
		boxes[region.Name] = *region.Collision
	}
	var state FourthFallingActor
	var random RandomState
	nativeCombatRows(t, "fourth-falling-trace.csv", func(v []int64) {
		if v[1] == 0 {
			state = NewFourthFallingActor(int(v[0]), art, ActorResidue{EmitterClock: 19}, uint32(v[0]*43))
			random = NewRandomState()
		}
		event := state.Advance(art, int(v[2]), func(name string) visualassets.CollisionBox { return boxes[name] }, random.Next)
		if state.X != int(v[3]) || state.Y != int(v[4]) || state.Phase != int(v[5]) || state.PrimaryClock != uint8(v[6]) || state.SecondaryClock != uint8(v[7]) || state.Sprite(art) != atlas.SourceSpriteNames[int(v[8])] || state.Collision.Left != int(v[9]) || state.Collision.Top != int(v[10]) || state.Collision.Right != int(v[11]) || state.Collision.Bottom != int(v[12]) || state.Removed != (v[13] != 0) || event.SpawnPod != (v[14] != 0) || event.SpawnPod && (event.PodX != int(v[15]) || event.PodY != int(v[16])) || event.Shot != (v[17] != 0) || event.Shot && (event.ShotX != int(v[18]) || event.ShotY != int(v[19]) || event.ShotDirection != uint8(v[20]) || event.ShotSpeed != int(v[21])) || random.A != uint32(v[22]) || random.B != uint32(v[23]) {
			t.Fatalf("falling actor %v: state=%+v event=%+v random=%+v", v, state, event, random)
		}
	})
}
