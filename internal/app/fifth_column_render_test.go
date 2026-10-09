package app

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// The native recorder executes the original planar shaft rasterizer and records
// its cap calls. Cap references use those arguments and the raw masked images,
// independently of the exported atlas or the production sprite decoder.
func TestFifthColumnNativeGPUPixelsOptional(t *testing.T) {
	trace := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if trace == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare original column rasterizations")
	}
	root := filepath.Join(filepath.Dir(trace), "fifth-column-render")
	file, err := os.Open(filepath.Join(root, "cases.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(filepath.Dir(trace), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	g, b := renderIntegrationGame(t)
	groups := g.Bundle.Levels[4].GuardianGroups
	actual := ebiten.NewImage(ScreenWidth, PlayfieldHeight)
	defer actual.Dispose()
	background := make([]byte, ScreenWidth*PlayfieldHeight*4)
	pixels, expected := make([]byte, len(background)), make([]byte, len(background))
	palette := b.Levels[4].Terrain.Palette
	cases, compared := 0, 0
	for _, row := range rows[1:] {
		if len(row) != 10 {
			t.Fatal("invalid native column case")
		}
		var v [10]int
		for i, field := range row {
			v[i], err = strconv.Atoi(field)
			if err != nil {
				t.Fatal(err)
			}
		}
		native, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("shaft-%03d.bin", v[0])))
		if err != nil {
			t.Fatal(err)
		}
		if len(native) != 30720 {
			t.Fatalf("case %d has %d planar bytes", v[0], len(native))
		}
		for y := range PlayfieldHeight {
			for x := range ScreenWidth {
				index := 0
				if v[5] != 0 {
					index = (x/7 + y*3) % 16
				}
				c := palette[index]
				copy(background[(y*ScreenWidth+x)*4:], c[:])
				index = 0
				for plane := range 4 {
					bit := (native[y*160+plane*40+x/8] >> (7 - x%8)) & 1
					index |= int(bit) << plane
				}
				c = palette[index]
				copy(expected[(y*ScreenWidth+x)*4:], c[:])
			}
		}
		tier := 0
		if v[4] < 0 {
			tier = 1
		}
		for _, caps := range []bool{false, true} {
			g.Bundle.Levels[4].GuardianGroups = nil
			if caps {
				g.Bundle.Levels[4].GuardianGroups = groups
				// Both original caps lie outside the shaft's rows, so its native
				// raster remains unchanged when their masked pixels are added.
				paintOriginalMaskedSpriteReference(t, expected, source, 0x5f2fe, v[6], v[7], image.Rect(0, 0, ScreenWidth, PlayfieldHeight), palette)
				paintOriginalMaskedSpriteReference(t, expected, source, 0x5f340, v[8], v[9], image.Rect(0, 0, ScreenWidth, PlayfieldHeight), palette)
			}
			actual.WritePixels(background)
			g.drawSprite(actual, SpriteView{Kind: "fifth-column", X: float64(v[1]), Y: float64(v[2]), Length: v[3], Tier: tier}, 5, 1)
			actual.ReadPixels(pixels)
			for offset := 0; offset < len(pixels); offset += 4 {
				got := [4]uint8{pixels[offset], pixels[offset+1], pixels[offset+2], pixels[offset+3]}
				want := [4]uint8{expected[offset], expected[offset+1], expected[offset+2], expected[offset+3]}
				if got != want {
					x, y := offset/4%ScreenWidth, offset/4/ScreenWidth
					t.Fatalf("case %d caps %v at (%d,%d), column (%d,%d) length %d speed %d background %d: got %v, original %v", v[0], caps, x, y, v[1], v[2], v[3], v[4], v[5], got, want)
				}
				compared++
			}
		}
		cases++
	}
	if cases != 160 || compared != 19660800 {
		t.Fatalf("incomplete original column coverage: %d cases, %d pixels", cases, compared)
	}
	t.Logf("Compared %d original shaft rasters with and without caps, %d GPU pixels", cases, compared)
}

func paintOriginalMaskedSpriteReference(t *testing.T, pixels, source []byte, address, x, y int, clip image.Rectangle, palette [16][4]uint8) {
	t.Helper()
	if x == 1000 && y == 1000 {
		return // The original renderer did not call this cap.
	}
	at := address - 0x54e00
	if at < 0 || at+8 > len(source) {
		t.Fatal("original masked sprite header is truncated")
	}
	word := func(offset int) uint16 { return binary.BigEndian.Uint16(source[at+offset:]) }
	x -= int(int16(word(0)))
	y -= int(int16(word(2)))
	width, height := int(word(4)), int(word(6))+1
	rowBytes := ((width + 15) / 16) * 2
	if width < 1 || width > 32 || height < 1 || height > 256 || at+8+rowBytes*5*height > len(source) {
		t.Fatal("original masked sprite dimensions are invalid")
	}
	for sy := range height {
		for sx := range width {
			px, py := x+sx, y+sy
			if !image.Pt(px, py).In(clip) {
				continue
			}
			start, bit := at+8+sy*rowBytes*5+sx/8, uint(7-sx%8)
			if source[start+4*rowBytes]&(1<<bit) == 0 {
				continue
			}
			index := 0
			for plane := range 4 {
				index |= int(source[start+plane*rowBytes]>>bit&1) << plane
			}
			c := palette[index]
			copy(pixels[(py*ScreenWidth+px)*4:], c[:])
		}
	}
}
