package engine

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"xenon2/internal/visualassets"
)

func TestSecondFixedHatchNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to local source comparisons")
	}
	bytes, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "00FA00FE.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	art, _, err := visualassets.DecodeFixedTiles(2, bytes)
	if err != nil {
		t.Fatal(err)
	}
	kind := art.Kinds[1]
	file, err := os.Open(filepath.Join(root, "fixed-hatch-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	var state FixedHatchState
	var tile0, tile1 uint32
	frames := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		v := make([]int, 6)
		for i := range v {
			v[i], err = strconv.Atoi(row[i])
			if err != nil {
				t.Fatal(err)
			}
		}
		if v[1] == 0 {
			state = FixedHatchState{X: 100, WorldY: 1000}
			tile0, tile1 = 0, 0
		}
		event := state.Advance(v[2], kind.TriggerScreenY)
		if event.WriteTiles {
			patch := kind.Variants[v[0]].Frames[event.Frame]
			tile0 = uint32(patch.Tiles[0])<<16 | uint32(patch.Tiles[1])
			tile1 = uint32(patch.Tiles[2])<<16 | uint32(patch.Tiles[3])
		}
		parts := strings.Split(row[6], ":")
		want0, _ := strconv.ParseUint(parts[0], 10, 32)
		want1, _ := strconv.ParseUint(parts[1], 10, 32)
		if state.Phase != v[3] || state.Removed != (v[4] == 4) || event.Spawn != (v[5] == 8) || tile0 != uint32(want0) || tile1 != uint32(want1) {
			t.Fatalf("hatch differs%v state=%+v tiles=%d:%d", v, state, tile0, tile1)
		}
		frames++
	}
	if frames != 48 {
		t.Fatalf("incomplete hatch comparisons:%d", frames)
	}
}

func TestHatchCreatureNativeMotionOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to local source comparisons")
	}
	bytes, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "00FA00FE.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(bytes)
	if err != nil {
		t.Fatal(err)
	}
	bank, err := visualassets.DecodeFixedSprites(2, bytes, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	art := bank.HatchCreatures
	file, err := os.Open(filepath.Join(root, "hatch-creature-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	var state HatchCreatureState
	frames := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		v := make([]int64, len(row))
		for i := range v {
			v[i], err = strconv.ParseInt(row[i], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		if v[1] == 0 {
			clip := art.HeadingAnimations[v[0]]
			state = HatchCreatureState{X: 160, Y: 100, Direction: uint8(v[0]), Clip: clip, Animation: NewAnimation(clip)}
		}
		state.Advance(art, FixedProjectileInputs{ScrollDelta: 1, PlayerX: 160, PlayerY: 176}, func(string) visualassets.CollisionBox { return visualassets.CollisionBox{} })
		if state.X != int(v[2]) || state.Y != int(v[3]) || state.Timer != int(v[4]) || state.Direction != uint8(v[5]) || state.Removed != (v[7] == 4) || state.Animation.Sprite(state.Clip) != bank.Atlas.SourceSpriteNames[int(v[6])] {
			t.Fatalf("creature differs %v: %+v", v, state)
		}
		frames++
	}
	if frames < 100 {
		t.Fatalf("incomplete creature motion comparisons:%d", frames)
	}
}
