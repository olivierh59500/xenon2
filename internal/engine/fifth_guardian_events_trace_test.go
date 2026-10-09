package engine

import (
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/visualassets"
)

func fifthGuardianEventTraceArt(t *testing.T) []visualassets.GuardianGroup {
	t.Helper()
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare original fifth guardian events")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(raw)
	if err != nil {
		t.Fatal(err)
	}
	groups, _, err := visualassets.DecodeCompoundGuardianArt(5, raw, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	return groups
}

// The recorder's ordinary-shot counter resets for each component. Sound queue
// words persist across components, so only the final row represents the frame's
// final requests. Keep -1 distinct from sampled requests such as 0x84.
// The historical "lasers" counter also counts side-seeker/mouth factories and
// omits known growing-column births. Actual column factories have a separate
// completed-birth trace; this test makes no claim for the historical counter.
func TestFifthGuardianOrdinaryShotAndSoundEventsNativeTraceOptional(t *testing.T) {
	groups := fifthGuardianEventTraceArt(t)
	for _, final := range []bool{false, true} {
		name, file, parts, frames, wantShots := "middle", "guardian-fifth-middle-trace.csv", 10, 1500, 388
		if final {
			name, file, parts, frames, wantShots = "final", "guardian-fifth-final-trace.csv", 22, 1200, 619
		}
		t.Run(name, func(t *testing.T) {
			group := &groups[0]
			if final {
				group = &groups[1]
			}
			middle, err := NewFifthMiddleGuardianState(&groups[0], -8)
			if err != nil {
				t.Fatal(err)
			}
			last, err := NewFifthFinalGuardianState(&groups[1])
			if err != nil {
				t.Fatal(err)
			}
			random := NewRandomState()
			var event FifthGuardianEvents
			rows, compared, nativeShots, totalShots, queuedSounds := 0, 0, 0, 0, 0
			nativeCombatRows(t, file, func(v []int64) {
				if len(v) != 18 {
					t.Fatalf("unexpected native event schema: %d columns", len(v))
				}
				frame, index, scroll := int(v[0]), int(v[1]), int(v[2])
				if frame != rows/parts || index != rows%parts {
					t.Fatalf("native component order differs at row%d: frame%d part%d", rows, frame, index)
				}
				if index == 0 {
					nativeShots = 0
					if final {
						delta := 0
						if scroll > 0 {
							delta = 1
						}
						event = last.Advance(group, scroll, delta, 1, 416, frame, frame*3%320, 96+frame%80, &random).FifthGuardianEvents
					} else {
						delta := 1
						if frame%11 == 0 {
							delta = 0
						}
						event = middle.Advance(group, scroll, delta, scroll, frame*3%320, 96+frame%80, &random)
					}
				}
				nativeShots += int(v[14])
				rows++
				if index != parts-1 {
					return
				}
				ordinary := 0
				for _, shot := range event.Shots {
					switch shot.Animation {
					case "mount-shot":
						ordinary++
					case "middle-side-shot":
						if final {
							t.Fatal("final guardian emitted a middle side seeker")
						}
					default:
						t.Fatalf("unclassified native shot event %q", shot.Animation)
					}
				}
				if ordinary != nativeShots || event.Sound2 != int(v[16]) || event.Sound1 != int(v[17]) {
					t.Fatalf("frame%d ordinary shots%d want%d sound2%d want%d sound1%d want%d", frame, ordinary, nativeShots, event.Sound2, v[16], event.Sound1, v[17])
				}
				totalShots += ordinary
				if event.Sound2 >= 0 {
					queuedSounds++
				}
				if event.Sound1 >= 0 {
					queuedSounds++
				}
				compared++
			})
			if rows != parts*frames || compared != frames || totalShots != wantShots {
				t.Fatalf("incomplete native event coverage: rows%d frames%d shots%d", rows, compared, totalShots)
			}
			t.Logf("%d original frames: %d ordinary mount shots and %d queued sound requests", compared, totalShots, queuedSounds)
		})
	}
}
