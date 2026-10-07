package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestFifthFinalGuardianNativeTraceOptional(t *testing.T) {
	dir := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if dir == "" {
		t.Skip("local fifth final guardian reference not supplied")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(raw)
	if err != nil {
		t.Fatal(err)
	}
	groups, atlas, err := visualassets.DecodeCompoundGuardianArt(5, raw, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	group := &groups[1]
	state, err := NewFifthFinalGuardianState(group)
	if err != nil {
		t.Fatal(err)
	}
	random := NewRandomState()
	f, err := os.Open(filepath.Join(dir, "guardian-fifth-final-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows[1:] {
		number := func(at int) int {
			value, err := strconv.ParseInt(row[at], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			return int(value)
		}
		frame, index, scroll := number(0), number(1), number(2)
		if index == 0 {
			delta := 0
			if scroll > 0 {
				delta = 1
			}
			state.Advance(group, scroll, delta, 1, 416, frame, frame*3%320, 96+frame%80, &random)
		}
		part := state.Parts[index]
		if part.X != number(3) || part.Y != number(4) || part.Clock != number(5) || part.Health != number(7) || part.FireAccumulator != uint8(number(9)) || part.SecondaryAccumulator != uint8(number(10)) {
			t.Fatalf("frame %d part %d Go %+v native %v", frame, index, part, row)
		}
		if group.Components[index].RenderMode == "sprite" && part.Sprite != atlas.SourceSpriteNames[number(11)] {
			t.Fatalf("frame %d part %d sprite Go %s native %s", frame, index, part.Sprite, atlas.SourceSpriteNames[number(11)])
		}
		if index == 21 && (random.A != uint32(number(12)) || random.B != uint32(number(13))) {
			t.Fatalf("frame %d shared RNG Go %x/%x native %x/%x", frame, random.A, random.B, number(12), number(13))
		}
	}
	t.Logf("Compared %d fifth final guardian part states.", len(rows)-1)
}
