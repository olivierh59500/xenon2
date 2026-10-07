package presentation

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/engine"
)

func TestPrivateOriginalStarfieldState(t *testing.T) {
	dir := os.Getenv("XENON2_AUDIO_TEST_DIR")
	if dir == "" {
		t.Skip("local original star reference not supplied")
	}
	f, err := os.Open(filepath.Join(dir, "analysis", "stars-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	r := engine.NewRandomState()
	s := NewStarfield(&r, []uint8{7, 7, 8, 8, 6, 5, 4, 9})
	for frame := 0; frame < (len(rows)-1)/48; frame++ {
		s.Advance()
		for index, star := range s.Stars {
			row := rows[1+frame*48+index]
			number := func(column int) int {
				v, err := strconv.Atoi(row[column])
				if err != nil {
					t.Fatal(err)
				}
				return v
			}
			if int(star.X) != number(2) || int(star.Y) != number(3) || int(star.Depth) != number(4) {
				t.Fatalf("frame %d star %d Go %d/%d/%d original %s/%s/%s", frame, index, star.X, star.Y, star.Depth, row[2], row[3], row[4])
			}
		}
	}
	t.Logf("Compared %d original starfield passes on 48 stars.", (len(rows)-1)/48)
}
