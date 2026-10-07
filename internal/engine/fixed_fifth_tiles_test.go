package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestFifthFixedTileNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local fifth fixed-tile reference not supplied")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	art, _, err := visualassets.DecodeFixedTiles(5, raw)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[int]visualassets.FixedTileKind{}
	for _, kind := range art.Kinds {
		kinds[kind.Kind] = kind
	}
	f, err := os.Open(filepath.Join(root, "fixed-fifth-tiles-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var state FifthTileState
	var random RandomState
	for _, row := range rows[1:] {
		n := func(i int) int {
			value, err := strconv.Atoi(row[i])
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
		family, variant, frame := n(0), n(1), n(2)
		kind := kinds[1]
		if family == 1 {
			kind = kinds[3]
		}
		if frame == 0 {
			state = FifthTileState{X: 96, WorldY: 1000, Part: variant, Heading: uint8(variant)}
			if family == 0 {
				state.X += kind.Parts[variant].OffsetX
				state.Heading = 0
			}
			random = NewRandomState()
		}
		var event FixedTileEvents
		if family == 0 {
			event = state.AdvanceBarrier(kind, n(3))
		} else {
			event = state.AdvanceAimingTurret(kind, n(3), n(4), n(5), n(6), &random)
		}
		if state.X != n(7) || state.WorldY != n(8) || state.Phase != n(9) || int(state.Heading) != n(10) || int(state.Accumulator) != n(11) || state.Removed != (family == 1 && n(12) != 244) || uint64(random.A) != uint64(n(23)) || uint64(random.B) != uint64(n(24)) {
			t.Fatalf("state differs: Go%+v RNG%+v native%v", state, random, row)
		}
		if !state.Removed && (event.Collision.Left != n(13) || event.Collision.Top != n(14) || event.Collision.Right != n(15) || event.Collision.Bottom != n(16)) {
			t.Fatalf("collision Go%+v native%v", event.Collision, row)
		}
		if event.Shot != (n(17) != 0) || event.Shot && (event.ShotX != n(18) || event.ShotY != n(19) || int(event.ShotDirection) != n(20) || event.ShotSpeed != n(21)) {
			t.Fatalf("shot Go%+v native%v", event, row)
		}
		if event.Shot {
			choice := int(state.Heading) & 3
			if frame&1 == 0 {
				choice += 4
			}
			if n(22) != int(uint32(raw[0x572ca-0x54e00+choice*4])<<24|uint32(raw[0x572cb-0x54e00+choice*4])<<16|uint32(raw[0x572cc-0x54e00+choice*4])<<8|uint32(raw[0x572cd-0x54e00+choice*4])) {
				t.Fatalf("shot image choice differs: %v", row)
			}
		}
		if event.WriteTiles {
			var patch visualassets.TilePatch
			if family == 0 {
				patch = kind.Parts[variant].Frames[event.Frame]
			} else {
				patch = kind.Variants[variant].Frames[event.Frame]
			}
			for y := 0; y < patch.Rows; y++ {
				for x := 0; x < patch.Columns; x++ {
					if int(patch.Tiles[y*patch.Columns+x]) != n(25+y*2+x) {
						t.Fatalf("tile Go%+v native%v", patch, row)
					}
				}
			}
		}
	}
	if len(rows)-1 != 7744 {
		t.Fatalf("incomplete source comparison: %d", len(rows)-1)
	}
	t.Logf("Compared %d original fifth fixed-tile states.", len(rows)-1)
}
