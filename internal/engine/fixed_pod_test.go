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

func TestSecondPodsNativeTraceOptional(t *testing.T) {
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
	file, err := os.Open(filepath.Join(root, "fixed-pod-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	var state FixedPodState
	var tile0, tile1 uint32
	children, frames := 0, 0
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
			state = FixedPodState{X: 100, WorldY: 1000, Repeats: 1, Large: v[0] == 0}
			tile0, tile1 = 0, 0
			children = 0
		}
		event := state.Advance(uint64(v[1]), v[2], 1000)
		kind := art.Kinds[2]
		if state.Large {
			kind = art.Kinds[3]
		}
		if event.WriteTiles {
			patch := kind.Variants[0].Frames[event.Frame]
			tile0 = uint32(patch.Tiles[0]) << 16
			if patch.Columns == 2 {
				tile0 |= uint32(patch.Tiles[1])
				tile1 = uint32(patch.Tiles[2])<<16 | uint32(patch.Tiles[3])
			}
		}
		if event.Spawn {
			children++
		}
		parts := strings.Split(row[6], ":")
		want0, _ := strconv.ParseUint(parts[0], 10, 32)
		want1, _ := strconv.ParseUint(parts[1], 10, 32)
		repeats, _ := strconv.Atoi(row[7])
		if state.Phase != v[3] || state.Removed != (v[4] == 4) || children != v[5] || state.Repeats != repeats || tile0 != uint32(want0) || tile1 != uint32(want1) {
			t.Fatalf("pod differs%v: %+v children=%d tiles=%d:%d", v, state, children, tile0, tile1)
		}
		frames++
	}
	if frames < 60 {
		t.Fatalf("incomplete pod comparisons:%d", frames)
	}
}

func TestPodCreatureNativeMotionOptional(t *testing.T) {
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
	art := bank.PodCreatures
	file, err := os.Open(filepath.Join(root, "pod-creature-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	var state PodCreatureState
	frames := 0
	pool := NewActorPool()
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
		if v[2] == 0 {
			clip := art.Idle[v[1]]
			state = PodCreatureState{X: 100, Y: 100, Variant: int(v[1]), AllocationPhase: int(pool.Slot(int(v[0])).AllocationPhase), Clip: clip, Animation: NewAnimation(clip)}
		}
		playerX := 160
		if v[2] >= 90 {
			playerX = 100
		}
		state.Advance(uint64(v[2]), playerX, art)
		if state.X != int(v[3]) || state.Y != int(v[4]) || state.Phase != int(v[5]) || state.Animation.Sprite(state.Clip) != bank.Atlas.SourceSpriteNames[int(v[6])] {
			t.Fatalf("pod creature differs%v: %+v", v, state)
		}
		frames++
	}
	if frames != 5120 {
		t.Fatalf("incomplete creature comparisons:%d", frames)
	}
}
