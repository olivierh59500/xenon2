package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"xenon2/internal/visualassets"
)

func TestPathMotionFormationDelayAndPause(t *testing.T) {
	path := visualassets.Path{ID: 1, Commands: []visualassets.PathCommand{
		{Kind: "origin", X: 38, Y: -53},
		{Kind: "curve", Heading: 64, Duration: 4},
		{Kind: "pause", Duration: 3},
		{Kind: "curve", Heading: 64, Duration: 2},
		{Kind: "end"},
	}}
	var sine [256]int8
	sine[64] = 64
	state, err := NewPathMotion(&path, PathMotionConfig{StartXOffset: 8, Delay: 3, Budget: 2})
	if err != nil {
		t.Fatal(err)
	}
	wantY := []int{-53, -52, -50, -49, -49, -47, -47}
	for frame, want := range wantY {
		if err := state.Advance(&path, &sine, nil); err != nil {
			t.Fatal(err)
		}
		if state.X != 46<<16 || state.Y != int32(want)<<16 {
			t.Fatalf("frame %d: position %+v, expected Y %d", frame, state, want)
		}
	}
	if state.Active {
		t.Fatalf("path did not end: %+v", state)
	}
}

func TestPathMotionBranchAndInvalidLoop(t *testing.T) {
	path := visualassets.Path{ID: 2, Commands: []visualassets.PathCommand{
		{Kind: "random-branch", Targets: []int{-1, 1, -1, -1, -1, -1, -1, -1}},
		{Kind: "curve", Heading: 0, Duration: 1},
		{Kind: "end"},
	}}
	var sine [256]int8
	sine[64] = 64
	state, err := NewPathMotion(&path, PathMotionConfig{Budget: 2})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	random := func() uint16 {
		calls++
		if calls == 1 {
			return 0
		}
		return 2
	}
	if err := state.Advance(&path, &sine, random); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || state.X != 1<<16 || state.Active {
		t.Fatalf("branch outcome: calls=%d state=%+v", calls, state)
	}
	path.Commands = []visualassets.PathCommand{{Kind: "jump", Target: 0}}
	state, err = NewPathMotion(&path, PathMotionConfig{Budget: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Advance(&path, &sine, nil); err == nil {
		t.Fatal("non-progressing path must fail without hanging")
	}
}

// TestPathMotionNativeTraceOptional compares fractional movement with an
// isolated original-routine trace and artwork data kept outside Git.
func TestPathMotionNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	level, err := os.ReadFile(filepath.Join(root, "000B00E5.unpacked"))
	if err != nil {
		t.Fatal(err)
	}
	common, err := os.ReadFile(filepath.Join(filepath.Dir(root), "XenonII-unpacked.bin"))
	if err != nil {
		t.Fatal(err)
	}
	paths, err := visualassets.DecodePaths(level, common)
	if err != nil {
		t.Fatal(err)
	}
	path := &paths.Paths[0]
	state, err := NewPathMotion(path, PathMotionConfig{Budget: 7})
	if err != nil {
		t.Fatal(err)
	}
	offsets := make([]int, len(path.Commands))
	for i := 1; i < len(offsets); i++ {
		length := map[string]int{"end": 2, "curve": 10, "pause": 4, "random-branch": 18, "jump": 4, "origin": 6}[path.Commands[i-1].Kind]
		offsets[i] = offsets[i-1] + length
	}
	file, err := os.Open(filepath.Join(root, "path1-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows[1:] {
		values := make([]uint64, len(row))
		for i, field := range row {
			values[i], err = strconv.ParseUint(field, 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := state.Advance(path, &paths.SineTable, nil); err != nil {
			t.Fatal(err)
		}
		angle := uint32(state.AngleFixed)
		swapped := angle<<16 | angle>>16
		if uint32(state.X) != uint32(values[2]) || uint32(state.Y) != uint32(values[3]) || swapped != uint32(values[4]) || uint32(state.AngularVelocity) != uint32(values[5]) || uint16(state.AngularAcceleration) != uint16(values[6]) || uint16(state.Remaining) != uint16(values[7]) || offsets[state.ProgramCounter] != int(values[8]) || state.Active != (values[1] != 4) {
			t.Fatalf("frame %d: got %+v, expected %v", values[0], state, row)
		}
	}
}

func TestRecoveredPathsAcceptMotionSchemaOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to validate local original path data")
	}
	common, err := os.ReadFile(filepath.Join(filepath.Dir(root), "XenonII-unpacked.bin"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"} {
		level, err := os.ReadFile(filepath.Join(root, name+".unpacked"))
		if err != nil {
			t.Fatal(err)
		}
		paths, err := visualassets.DecodePaths(level, common)
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths.Paths {
			if err := ValidatePath(&path); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		}
	}
}
