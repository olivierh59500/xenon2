package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestPlayerMotionHeldDriftAndReverse(t *testing.T) {
	state := PlayerMotionState{X: 160, Y: 100}
	context := MotionContext{ScrollY: 3000, VisitedScrollY: 3000, BaseScrollStep: 1}
	for i := 0; i < 8; i++ {
		state.Advance(MotionInput{Right: true}, context)
	}
	if state.X != 184 || state.Inertia != 6 || state.BankFrame() != 4 {
		t.Fatalf("held motion: %+v", state)
	}
	for i := 0; i < 8; i++ {
		state.Advance(MotionInput{}, context)
	}
	if state.X != 190 || state.Inertia != 0 || state.BankFrame() != 2 {
		t.Fatalf("release drift: %+v", state)
	}
	state.Advance(MotionInput{Left: true}, context)
	if state.X != 187 || state.Inertia != -1 || state.BankFrame() != 1 {
		t.Fatalf("reverse motion: %+v", state)
	}
	state.Advance(MotionInput{}, context)
	if state.X != 187 || state.Inertia != 0 {
		t.Fatalf("short release: %+v", state)
	}
}

func TestPlayerMotionPreservesSignedRoundingAndVerticalEdges(t *testing.T) {
	state := PlayerMotionState{X: 160, Y: 17, SpeedTier: 2, Inertia: -6}
	context := MotionContext{ScrollY: 2900, VisitedScrollY: 3000, BaseScrollStep: 1}
	state.Advance(MotionInput{Up: true}, context)
	if state.X != 152 || state.Y != 12 || state.Inertia != -5 {
		t.Fatalf("signed drift and first edge crossing: %+v", state)
	}
	state.Advance(MotionInput{Up: true, Down: true}, context)
	if state.Y != 16 || state.ScrollStep != 1 || state.ScrollReverseRequested {
		t.Fatalf("upper edge takes precedence: %+v", state)
	}
	state.Y = 176
	state.Advance(MotionInput{Down: true}, context)
	if state.Y != 176 || state.ScrollStep != -1 || !state.ScrollReverseRequested {
		t.Fatalf("reverse scroll: %+v", state)
	}
	context.VisitedScrollY = context.ScrollY
	state.Advance(MotionInput{Down: true}, context)
	if state.ScrollStep != 1 || state.ScrollReverseRequested {
		t.Fatalf("visited extent prevents reverse scroll: %+v", state)
	}
}

// TestPlayerMotionNativeTraceOptional compares directional motion with an
// isolated original-routine trace kept outside the repository.
func TestPlayerMotionNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	f, err := os.Open(filepath.Join(root, "player-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	state := PlayerMotionState{}
	context := MotionContext{ScrollY: 3000, VisitedScrollY: 3000, BaseScrollStep: 1}
	for _, row := range rows[1:] {
		values := make([]int, len(row))
		for i, field := range row {
			values[i], err = strconv.Atoi(field)
			if err != nil {
				t.Fatal(err)
			}
		}
		if values[1] == 0 {
			state = PlayerMotionState{X: 160, Y: 100, SpeedTier: values[0]}
		}
		input := values[2]
		state.Advance(MotionInput{Up: input&1 != 0, Down: input&2 != 0, Left: input&4 != 0, Right: input&8 != 0}, context)
		if state.X != values[3] || state.Y != values[4] || state.Inertia != values[5] || state.ScrollStep != values[6] {
			t.Fatalf("tier %d frame %d: got %+v, expected %v", values[0], values[1], state, row)
		}
	}
}
