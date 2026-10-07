package engine

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestFirstMiddleSchedulerNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to local source comparisons")
	}
	data := originalWorldData(t, 1)
	group := data.GuardianGroups[0]
	var gates [16]int
	for i, launch := range group.Launches {
		gates[i] = launch.GateID
	}
	file, err := os.Open(filepath.Join(root, "first-middle-scheduler-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	state := NewFirstMiddleState()
	maximum := 3200
	frames := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		v := make([]int64, 10)
		for i := range v {
			v[i], err = strconv.ParseInt(row[i], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		pass := int(v[0])
		scroll := 3500
		if pass >= 5 {
			scroll = 3000
		}
		if pass >= 130 {
			scroll = 2400
		}
		for i := range state.Updated {
			state.Updated[i] = int(v[4])&(1<<i) != 0
		}
		event := state.Advance(scroll, maximum, int(v[3]), gates)
		maximum = event.Maximum
		if event.Scroll != int(v[1]) || maximum != int(v[2]) {
			t.Fatalf("camera differs at pass%d: %+v", pass, event)
		}
		for i, seed := range state.Seeds {
			if uint64(seed) != uint64(v[5+i]) {
				t.Fatalf("seed%d differs at pass%d", i, pass)
			}
		}
		launches := 0
		if row[10] != "" {
			for _, entry := range strings.Split(row[10], "|") {
				p := strings.Split(entry, ":")
				stream, _ := strconv.Atoi(p[0])
				launch, _ := strconv.Atoi(p[1])
				if event.Launches[stream] != launch {
					t.Fatalf("launch differs pass%d stream%d", pass, stream)
				}
				launches++
			}
		}
		if event.LaunchCount != launches {
			t.Fatalf("launch count differs pass%d", pass)
		}
		for i, p := range strings.Split(row[11], "|") {
			value, _ := strconv.Atoi(p)
			if state.GateCounters[i] != value {
				t.Fatalf("gate%d differs pass%d", i, pass)
			}
		}
		frames++
	}
	if frames != 160 {
		t.Fatalf("incomplete scheduler comparisons:%d", frames)
	}
}

func TestFirstMiddleFollowerNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to local source comparisons")
	}
	data := originalWorldData(t, 1)
	group := data.GuardianGroups[0]
	file, err := os.Open(filepath.Join(root, "first-middle-parts-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	var anchor FirstMiddleAnchor
	var followers [11]FirstMiddleFollower
	random := NewRandomState()
	previousLaunch := -1
	previousPass := -1
	comparisons := 0
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
		launch, pass, part := int(v[0]), int(v[1]), int(v[2])
		if launch != previousLaunch {
			anchor, err = NewFirstMiddleAnchor(&group.Launches[launch].Path, group.Launches[launch], 0, 4, 3000, 1)
			if err != nil {
				t.Fatal(err)
			}
			for i := range followers {
				followers[i] = FirstMiddleFollower{X: anchor.Motion.X, Y: anchor.Motion.Y, Remaining: -(10 - i) * 7}
			}
			previousLaunch, previousPass = launch, -1
		}
		if pass != previousPass {
			for i := 0; i < len(followers); i++ {
				source := anchor
				if i < len(followers)-1 {
					next := followers[i+1]
					source = FirstMiddleAnchor{Motion: PathMotionState{X: next.X, Y: next.Y, AngleFixed: next.AngleFixed, Remaining: next.Remaining}, Removed: next.Removed}
				}
				followers[i].Advance(source, 1)
			}
			if err := anchor.Advance(&group.Launches[launch].Path, &data.Paths.SineTable, 1, &random); err != nil {
				t.Fatal(err)
			}
			previousPass = pass
		}
		if part < 1 || part > 12 || v[3] == 4 || v[3] == 0 {
			continue
		}
		var x, y, angle int32
		var remaining int
		var visible bool
		if part == 12 {
			x, y, angle, remaining = anchor.Motion.X, anchor.Motion.Y, anchor.Motion.AngleFixed, anchor.Motion.Remaining
		} else {
			f := followers[part-1]
			x, y, angle, remaining, visible = f.X, f.Y, f.AngleFixed, f.Remaining, f.Visible
		}
		originalAngle := uint32(v[6])<<16 | uint32(v[6])>>16
		if x != int32(v[4]) || y != int32(v[5]) || uint32(angle) != originalAngle || remaining != int(v[7]) || visible != (v[9] != 0) {
			t.Fatalf("chain differs at %v: x/y=%d/%d angle=%d delay=%d visible=%t", v[:8], x, y, angle, remaining, visible)
		}
		comparisons++
	}
	t.Logf("Matched %d native follower and path anchor states", comparisons)
	if comparisons < 10000 {
		t.Fatalf("incomplete chain comparisons:%d", comparisons)
	}
}
