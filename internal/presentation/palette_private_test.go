package presentation

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestPrivateOriginalPaletteFadeWrites(t *testing.T) {
	dir := os.Getenv("XENON2_SHOP_TEST_DIR")
	if dir == "" {
		t.Skip("local original palette references not supplied")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	art, err := visualassets.DecodeTitleArt(raw)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(filepath.Dir(dir), "analysis", "palette-trace.csv"))
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
			v, err := strconv.Atoi(row[at])
			if err != nil {
				t.Fatal(err)
			}
			return v
		}
		deduction := 6 - number(1)
		if row[0] == "out" {
			deduction = number(1) + 1
		}
		palette := FadePalette(art.Palette, deduction)
		value := number(3)
		want := [4]uint8{uint8(value>>8&7) * 34, uint8(value>>4&7) * 34, uint8(value&7) * 34, 255}
		if got := palette[number(2)]; got != want {
			t.Fatalf("%s step %s color %s Go %v native %v", row[0], row[1], row[2], got, want)
		}
	}
	t.Logf("Compared %d original palette-register states.", len(rows)-1)
}
