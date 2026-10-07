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

func TestFirstTileCannonNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to local source comparisons")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "000B00E5.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	art, _, err := visualassets.DecodeFirstLevelFixedTiles(data)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(filepath.Join(root, "fixed-tiles-first-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err = reader.Read(); err != nil {
		t.Fatal(err)
	}
	var state FixedTileState
	var random RandomState
	frames := 0
	var tile0, tile1 uint32
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		v := make([]int64, 19)
		for i := range v {
			v[i], err = strconv.ParseInt(row[i], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		if v[2] == 0 {
			state = FixedTileState{Kind: int(v[0]), Variant: int(v[1]), X: 100, WorldY: 1000, Accumulator: 250}
			random = NewRandomState()
			tile0, tile1 = 0, 0
		}
		kind := art.Kinds[state.Kind-1]
		event := StepFirstTileCannon(&state, kind, int(v[3]), int(v[4]), &random)
		if event.WriteTiles {
			patch := kind.Variants[state.Variant].Frames[event.Frame]
			tile0 = uint32(patch.Tiles[0])<<16 | uint32(patch.Tiles[1])
			if patch.Rows == 2 {
				tile1 = uint32(patch.Tiles[2])<<16 | uint32(patch.Tiles[3])
			}
		}
		parts := strings.Split(row[19], ":")
		want0, _ := strconv.ParseUint(parts[0], 10, 32)
		want1, _ := strconv.ParseUint(parts[1], 10, 32)
		if tile0 != uint32(want0) || tile1 != uint32(want1) {
			t.Fatalf("tile frame differs at %v: %d:%d want %s", v[:7], tile0, tile1, row[19])
		}
		if state.Phase != int(v[5]) || int(state.Accumulator) != int(v[6]) || state.Removed != (v[7] == 4) || uint64(random.A) != uint64(v[17]) || uint64(random.B) != uint64(v[18]) {
			t.Fatalf("state differs at %v: %+v random=%+v", v[:7], state, random)
		}
		if !state.Removed && (event.Collision.Left != int(v[8]) || event.Collision.Top != int(v[9]) || event.Collision.Right != int(v[10]) || event.Collision.Bottom != int(v[11])) {
			t.Fatalf("collision differs at %v: %+v", v[:7], event)
		}
		if event.Shot != (v[12] != 0) || event.Shot && (event.ShotX != int(v[13]) || event.ShotY != int(v[14]) || int(event.ShotDirection) != int(v[15]) || event.ShotSpeed != int(v[16])) {
			t.Fatalf("shot differs at %v: %+v", v[:7], event)
		}
		frames++
	}
	if frames != 960 {
		t.Fatalf("incomplete fixed-tile comparisons: %d", frames)
	}
}

func TestOtherTileCannonNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to local source comparisons")
	}
	sources := []string{"00FA00FE.decoded", "02020113.decoded", "031F0159.decoded", "04820138.decoded"}
	kinds := make(map[int]visualassets.FixedTileKind)
	for index, name := range sources {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", name))
		if err != nil {
			t.Fatal(err)
		}
		art, _, err := visualassets.DecodeFixedTiles(index+2, data)
		if err != nil {
			t.Fatal(err)
		}
		kinds[index+2] = art.Kinds[0]
	}
	file, err := os.Open(filepath.Join(root, "fixed-tiles-other-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err = reader.Read(); err != nil {
		t.Fatal(err)
	}
	var state FixedTileState
	var random RandomState
	frames := 0
	var tile0, tile1 uint32
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		v := make([]int64, 19)
		for i := range v {
			v[i], err = strconv.ParseInt(row[i], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		if v[2] == 0 {
			state = FixedTileState{Kind: int(v[0]), Variant: int(v[1]), X: 100, WorldY: 1000, Accumulator: 250}
			random = NewRandomState()
			tile0, tile1 = 0, 0
		}
		kind := kinds[state.Kind]
		event := StepTileFireCycle(&state, kind, int(v[3]), int(v[4]), &random)
		if event.WriteTiles {
			patch := kind.Variants[state.Variant].Frames[event.Frame]
			if patch.Columns == 1 {
				tile0 = uint32(patch.Tiles[0]) << 16
			} else {
				tile0 = uint32(patch.Tiles[0])<<16 | uint32(patch.Tiles[1])
				tile1 = uint32(patch.Tiles[2])<<16 | uint32(patch.Tiles[3])
			}
		}
		parts := strings.Split(row[19], ":")
		want0, _ := strconv.ParseUint(parts[0], 10, 32)
		want1, _ := strconv.ParseUint(parts[1], 10, 32)
		if tile0 != uint32(want0) || tile1 != uint32(want1) {
			t.Fatalf("tile frame differs at %v: %d:%d want %s", v[:7], tile0, tile1, row[19])
		}
		if state.Phase != int(v[5]) || int(state.Accumulator) != int(v[6]) || state.Removed != (v[7] == 4) || uint64(random.A) != uint64(v[17]) || uint64(random.B) != uint64(v[18]) {
			t.Fatalf("state differs at %v: %+v random=%+v", v[:7], state, random)
		}
		if !state.Removed && (event.Collision.Left != int(v[8]) || event.Collision.Top != int(v[9]) || event.Collision.Right != int(v[10]) || event.Collision.Bottom != int(v[11])) {
			t.Fatalf("collision differs at %v: %+v", v[:7], event)
		}
		if event.Shot != (v[12] != 0) || event.Shot && (event.ShotX != int(v[13]) || event.ShotY != int(v[14]) || int(event.ShotDirection) != int(v[15]) || event.ShotSpeed != int(v[16])) {
			t.Fatalf("shot differs at %v: %+v", v[:7], event)
		}
		frames++
	}
	if frames != 1280 {
		t.Fatalf("incomplete fixed-tile comparisons: %d", frames)
	}
}
