package engine

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"xenon2/internal/visualassets"
)

func TestFirstGuardianSegmentsNativeTraceOptional(t *testing.T) {
	dir := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if dir == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local guardian traces")
	}
	f, err := os.Open(filepath.Join(dir, "guardian-first-segments-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(filepath.Dir(dir)), "assets", "runtime", "level-1-paths.json"))
	if err != nil {
		t.Fatal(err)
	}
	var paths visualassets.Paths
	if err = json.Unmarshal(data, &paths); err != nil {
		t.Fatal(err)
	}
	controller := NewFirstGuardianState(30)
	segments := FirstGuardianSegments{}
	maximum, cursor := 4607, 1
	for frame := 1; frame <= 1200; frame++ {
		scroll := 48
		if frame < 40 {
			scroll = 500
		} else if frame < 80 {
			scroll = 448
		}
		if cursor < len(rows) {
			nextFrame, _ := strconv.Atoi(rows[cursor][0])
			if nextFrame == frame {
				var random []uint32
				for _, word := range strings.Split(rows[cursor+7][10], "|") {
					if word != "" {
						value, err := strconv.ParseUint(word, 16, 32)
						if err != nil {
							t.Fatal(err)
						}
						random = append(random, uint32(value))
					}
				}
				used := 0
				_, fired, err := segments.Advance(controller, scroll, 160, 176, &paths.SineTable, func() uint32 {
					if used >= len(random) {
						t.Fatalf("unexpected random call on guardian pass %d", frame)
					}
					value := random[used]
					used++
					return value
				})
				if err != nil || used != len(random) {
					t.Fatalf("guardian pass %d random use=%d/%d error=%v", frame, used, len(random), err)
				}
				for i := range 8 {
					row := rows[cursor+i]
					var v [10]int64
					for index := range v {
						v[index], err = strconv.ParseInt(row[index], 10, 64)
						if err != nil {
							t.Fatal(err)
						}
					}
					piece := segments.Pieces[i]
					angle := uint32(piece.AngleFixed)
					swapped := angle<<16 | angle>>16
					if piece.X != int32(v[2]) || piece.Y != int32(v[3]) || swapped != uint32(v[4]) || piece.AngularVelocity != int32(v[5]) || piece.AngularAcceleration != int32(v[6]) || piece.Budget != int(v[7]) {
						t.Fatalf("guardian pass %d segment %d got %+v native=%v", frame, i, piece, v)
					}
					if i == 7 && (segments.TailFire.Accumulator != uint8(v[8]) || fired != (v[9] != 0)) {
						t.Fatalf("guardian pass %d tail firing differs", frame)
					}
				}
				cursor += 8
			}
		}
		controller.Health = 10
		if frame < 400 {
			controller.Health = 30
		} else if frame < 800 {
			controller.Health = 25
		}
		var activate bool
		maximum, activate, _ = controller.Advance(uint64(frame), scroll, maximum)
		if activate {
			segments = NewFirstGuardianSegments()
		}
	}
	if cursor != len(rows) {
		t.Fatalf("unconsumed guardian segment rows: %d", len(rows)-cursor)
	}
}
