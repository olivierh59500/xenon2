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

// TestAllPathMotionNativeTracesOptional checks every recovered path against
// bounded native execution, using recorded random outputs for branch choices.
func TestAllPathMotionNativeTracesOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	trace, err := os.Open(filepath.Join(root, "all-paths-trace.csv"))
	if os.IsNotExist(err) {
		t.Skip("all-paths-trace.csv is not available")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer trace.Close()
	common, err := os.ReadFile(filepath.Join(filepath.Dir(root), "XenonII-unpacked.bin"))
	if err != nil {
		t.Fatal(err)
	}
	var levels [5]*visualassets.Paths
	for index, name := range []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"} {
		data, err := os.ReadFile(filepath.Join(root, name+".unpacked"))
		if err != nil {
			t.Fatal(err)
		}
		levels[index], err = visualassets.DecodePaths(data, common)
		if err != nil {
			t.Fatal(err)
		}
	}
	reader := csv.NewReader(trace)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	state := PathMotionState{}
	var path *visualassets.Path
	var sine *[256]int8
	var offsets []int
	cases, frames := 0, 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		var values [11]uint64
		for i := range values {
			values[i], err = strconv.ParseUint(row[i], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		if values[2] == 0 {
			paths := levels[values[0]-1]
			path, sine = &paths.Paths[values[1]-1], &paths.SineTable
			state, err = NewPathMotion(path, PathMotionConfig{Budget: 7})
			if err != nil {
				t.Fatal(err)
			}
			offsets = make([]int, len(path.Commands))
			for i := 1; i < len(offsets); i++ {
				length := map[string]int{"end": 2, "curve": 10, "pause": 4, "random-branch": 18, "jump": 4, "origin": 6}[path.Commands[i-1].Kind]
				offsets[i] = offsets[i-1] + length
			}
			cases++
		}
		var randomValues []uint16
		if row[11] != "" {
			for _, field := range strings.Split(row[11], ";") {
				value, err := strconv.ParseUint(field, 16, 32)
				if err != nil {
					t.Fatal(err)
				}
				randomValues = append(randomValues, uint16(value))
			}
		}
		calls := 0
		random := func() uint16 {
			if calls >= len(randomValues) {
				t.Fatalf("level %d path %d frame %d requested an unexpected random value", values[0], values[1], values[2])
			}
			value := randomValues[calls]
			calls++
			return value
		}
		if err := state.Advance(path, sine, random); err != nil {
			t.Fatalf("level %d path %d frame %d: %v", values[0], values[1], values[2], err)
		}
		angle := uint32(state.AngleFixed)
		swapped := angle<<16 | angle>>16
		if calls != len(randomValues) || uint32(state.X) != uint32(values[4]) || uint32(state.Y) != uint32(values[5]) || swapped != uint32(values[6]) || uint32(state.AngularVelocity) != uint32(values[7]) || uint16(state.AngularAcceleration) != uint16(values[8]) || uint16(state.Remaining) != uint16(values[9]) || offsets[state.ProgramCounter] != int(values[10]) || state.Active != (values[3] != 4) {
			t.Fatalf("level %d path %d frame %d: got %+v, expected %v", values[0], values[1], values[2], state, row)
		}
		frames++
	}
	if cases != 418 {
		t.Fatalf("incomplete native comparison: %d of 418 paths", cases)
	}
	t.Logf("Compared %d paths across %d original-routine frames", cases, frames)
}
