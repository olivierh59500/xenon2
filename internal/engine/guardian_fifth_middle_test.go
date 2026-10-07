package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestFifthMiddleGuardianNativeTraceOptional(t *testing.T) {
	dir := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if dir == "" {
		t.Skip("local fifth guardian reference not supplied")
	}
	level, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(level)
	if err != nil {
		t.Fatal(err)
	}
	groups, atlas, err := visualassets.DecodeCompoundGuardianArt(5, level, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	group := &groups[0]
	state, err := NewFifthMiddleGuardianState(group, -8)
	if err != nil {
		t.Fatal(err)
	}
	random := NewRandomState()
	f, err := os.Open(filepath.Join(dir, "guardian-fifth-middle-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var events FifthGuardianEvents
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
			delta := 1
			if frame%11 == 0 {
				delta = 0
			}
			events = state.Advance(group, scroll, delta, scroll, frame*3%320, 96+frame%80, &random)
		}
		part := state.Parts[index]
		if part.X != number(3) || part.Y != number(4) || part.Clock != number(5) || part.Health != number(7) || part.FireAccumulator != uint8(number(9)) || part.SecondaryAccumulator != uint8(number(10)) {
			t.Fatalf("frame %d part %d Go %+v native %v", frame, index, part, row)
		}
		if index == 0 && part.MoveRemaining != number(6) {
			t.Fatalf("frame %d body movement Go %d native %d", frame, part.MoveRemaining, number(6))
		}
		if index != 0 && part.Sprite != atlas.SourceSpriteNames[number(11)] {
			t.Fatalf("frame %d part %d sprite Go %s native %s", frame, index, part.Sprite, atlas.SourceSpriteNames[number(11)])
		}
		if index == 9 && (random.A != uint32(number(12)) || random.B != uint32(number(13))) {
			t.Fatalf("frame %d shared random stream Go %x/%x native %x/%x events %+v", frame, random.A, random.B, number(12), number(13), events)
		}
	}
	t.Logf("Compared %d fifth middle guardian part states.", len(rows)-1)
}
