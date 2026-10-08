package engine

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"xenon2/internal/visualassets"
)

// The local original trace executes the ship callback and final scroll routine.
// These arranged boundaries isolate terrain history from enemy combat; they do
// not represent a stage victory or a complete Amiga playthrough comparison.
func TestWorldDiveTerrainRewindNativeBoundariesOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to local original comparisons")
	}
	file, err := os.Open(filepath.Join(root, "dive-terrain-boundary-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	comparisons, underwaterRewinds, underwaterCrushes := 0, 0, 0
	for level := 1; level <= 3; level++ {
		t.Run(fmt.Sprintf("level%d", level), func(t *testing.T) {
			data := playableOriginalWorldData(t, level)
			data.Encounters = &visualassets.Encounters{}
			cases := 0
			for _, row := range rows[1:] {
				if len(row) != 22 {
					t.Fatalf("original boundary has %d columns, want 22", len(row))
				}
				var value [21]int
				for index := range value {
					value[index], err = strconv.Atoi(row[index])
					if err != nil {
						t.Fatal(err)
					}
				}
				if value[0] != level {
					continue
				}
				w, err := NewWorld(data)
				if err != nil {
					t.Fatal(err)
				}
				w.Ready, w.MaterializationFrames = false, 0
				w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = value[4], value[4]+32, value[4]+32
				w.Player.X, w.Player.Y, w.Player.Inertia = value[2], value[3], -3
				w.Equipment.SpeedTier = value[10]
				w.Dive = DiveState{Phase: value[6], Remaining: 136}
				if w.Dive.Phase == 1 {
					w.Dive.Direction = 1
				}
				w.Rewind.Timer = value[7]
				for index := range w.Rewind.History {
					w.Rewind.History[index] = ShipHistory{ScrollY: value[4] + index, X: 100 + 2*index, Y: value[8] + index}
				}
				if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *data.PlayerStencil) != (value[5] != 0) {
					t.Fatalf("original initial contact differs: %v", row)
				}
				input := value[9]
				motion := MotionInput{Up: input&1 != 0, Down: input&2 != 0, Left: input&4 != 0, Right: input&8 != 0}
				if err := w.Step(Input{Motion: motion}); err != nil {
					t.Fatal(err)
				}
				if w.PlayerAlive != (value[20] == 0) {
					t.Fatalf("original crushing boundary differs: %v, alive%v timer%d", row, w.PlayerAlive, w.Rewind.Timer)
				}
				if value[20] == 0 {
					hash := uint64(0xcbf29ce484222325)
					for _, position := range w.Rewind.History {
						for _, word := range []int{position.ScrollY, position.X, position.Y} {
							hash ^= uint64(uint16(word))
							hash *= 0x100000001b3
						}
					}
					wantHash, err := strconv.ParseUint(row[21], 10, 64)
					if err != nil {
						t.Fatal(err)
					}
					got := [9]int{w.Player.X, w.Player.Y, w.Player.Inertia, w.Rewind.Timer, w.Player.ScrollStep, w.ScrollY, w.MaximumScrollY, w.ScrollDeviationPasses, w.ScrollDelta}
					want := [9]int{value[11], value[12], value[13], value[14], value[15], value[16], value[17], value[18], value[19]}
					if got != want || hash != wantHash {
						t.Fatalf("original dive/terrain boundary differs: %v; got%v hash%d, want%v hash%d", row, got, hash, want, wantHash)
					}
				}
				if value[6] != 0 && value[7] != 0 {
					underwaterRewinds++
					if value[20] != 0 {
						underwaterCrushes++
					}
				}
				cases++
				comparisons++
			}
			if cases != 216 {
				t.Fatalf("incomplete original level comparison: %d", cases)
			}
		})
	}
	if comparisons != 648 || underwaterRewinds == 0 || underwaterCrushes == 0 {
		t.Fatalf("incomplete dive/terrain evidence: %d cases, %d underwater rewinds, %d underwater crushes", comparisons, underwaterRewinds, underwaterCrushes)
	}
	t.Logf("%d original ship/scroll boundaries, including %d underwater rewind entries and %d underwater crushes", comparisons, underwaterRewinds, underwaterCrushes)
}
