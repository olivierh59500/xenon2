package engine

import (
	"encoding/csv"
	"math/bits"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestPrivateOriginalBackgroundStars(t *testing.T) {
	dir := os.Getenv("XENON2_AUDIO_TEST_DIR")
	if dir == "" {
		t.Skip("local original background-star reference not supplied")
	}
	f, err := os.Open(filepath.Join(dir, "analysis", "background-stars-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var stars BackgroundStarfield
	var random RandomState
	previousDelta, previousFrame := 999, 999
	for _, row := range rows[1:] {
		number := func(at int) int {
			v, err := strconv.ParseInt(row[at], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			return int(v)
		}
		delta, frame, index := number(0), number(1), number(2)
		if delta != previousDelta {
			random = NewRandomState()
			stars = NewBackgroundStarfield(&random)
			previousDelta = delta
			previousFrame = -1
		}
		if frame != previousFrame {
			stars.Advance(delta)
			previousFrame = frame
		}
		offset, mask := number(3), number(4)
		wantX := (offset%160)*8 + 15 - bits.TrailingZeros16(uint16(mask))
		wantY := offset / 160
		got := stars.Stars[index]
		if got.X != wantX || got.Y != wantY || got.Fraction != number(5) || random.A != uint32(number(6)) || random.B != uint32(number(7)) {
			t.Fatalf("delta %d frame %d star %d Go %+v native %d/%d/%d RNG %x/%x", delta, frame, index, got, wantX, wantY, number(5), number(6), number(7))
		}
	}
	t.Logf("Compared %d original gameplay-star states across all 33 displacement values.", len(rows)-1)
}
