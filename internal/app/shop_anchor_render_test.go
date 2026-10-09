package app

import (
	"encoding/csv"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"xenon2/internal/visualassets"
)

// The reference records actual original draw arguments for every cursor/button
// selection and both complete hand sequences. Individual images are isolated
// here; these checks do not claim a full original shop framebuffer comparison.
func TestShopNativeAnchorsAndHandClipGPUPixelsOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare original shop draw arguments")
	}
	file, err := os.Open(filepath.Join(root, "shop-anchors-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "05c400f8.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	g, b := renderIntegrationGame(t)
	scene := &b.ShopScene
	expand := func(frames []visualassets.ShopHandFrame) []visualassets.ShopHandFrame {
		var result []visualassets.ShopHandFrame
		for _, frame := range frames {
			for range frame.Duration {
				result = append(result, frame)
			}
		}
		return result
	}
	intro, sale := expand(scene.IntroHand), expand(scene.SaleHand)
	actual := ebiten.NewImage(ScreenWidth, ScreenHeight)
	defer actual.Dispose()
	pixels := make([]byte, ScreenWidth*ScreenHeight*4)
	expected := make([]byte, len(pixels))
	var counts [4]int
	compared := 0
	for _, row := range rows[1:] {
		if len(row) != 10 {
			t.Fatal("invalid original shop draw row")
		}
		var v [10]int
		for i, value := range row {
			v[i], err = strconv.Atoi(value)
			if err != nil {
				t.Fatal(err)
			}
		}
		var name string
		x, y, bottom := 0, 0, ScreenHeight-1
		if v[1] < 2 {
			if v[2] < 4 {
				cell := scene.Cells[v[2]*5+v[3]]
				x, y, name = cell.CursorX, cell.CursorY, "shop-control-cursor"
				if v[1] == 0 {
					name += "-active"
				}
			} else {
				action := "buy"
				if v[4] != 0 {
					action = "sell"
				}
				if v[3] == 0 {
					action = "exit"
				}
				if v[1] == 0 {
					action += "-active"
				}
				for _, control := range scene.Controls {
					if control.ID == action {
						x, y, name = control.X, control.Y, control.Sprite
						break
					}
				}
			}
		} else {
			frames := intro
			if v[1] == 3 {
				frames = sale
			}
			if v[5] < 0 || v[5] >= len(frames) {
				t.Fatalf("original hand pass %d is outside %d exported passes", v[5], len(frames))
			}
			frame := frames[v[5]]
			x, y, name, bottom = frame.X, frame.Y, frame.Sprite, 104
		}
		if name == "" || x != v[7] || y != v[8] || bottom != v[9] {
			t.Fatalf("original draw %v: exported %s at (%d,%d), clip %d", v, name, x, y, bottom)
		}
		actual.Clear()
		clear(expected)
		if v[1] < 2 {
			g.drawAtlasSprite(actual, g.graphics.shopControls, name, float64(x), float64(y))
		} else {
			g.drawShopHand(actual, name, float64(x), float64(y))
		}
		paintOriginalMaskedSpriteReference(t, expected, source, v[6], v[7], v[8], image.Rect(0, 0, ScreenWidth, bottom+1), b.ShopArt.Palette)
		actual.ReadPixels(pixels)
		for at := 0; at < len(pixels); at += 4 {
			got := [4]uint8{pixels[at], pixels[at+1], pixels[at+2], pixels[at+3]}
			want := [4]uint8{expected[at], expected[at+1], expected[at+2], expected[at+3]}
			if got != want {
				t.Fatalf("original draw %v, %s at pixel (%d,%d): got %v want %v", v, name, at/4%ScreenWidth, at/4/ScreenWidth, got, want)
			}
			compared++
		}
		counts[v[1]]++
	}
	if counts != [4]int{50, 50, 35, 47} || compared != 11648000 {
		t.Fatalf("incomplete original shop draw coverage: %v cases, %d pixels", counts, compared)
	}
	t.Logf("Compared %v original cursor/button/intro-hand/sale-hand draws, %d GPU pixels", counts, compared)
}
