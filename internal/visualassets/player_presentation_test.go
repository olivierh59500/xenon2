package visualassets

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateOriginalHUDPixels(t *testing.T) {
	dir := os.Getenv("XENON2_SHOP_TEST_DIR")
	if dir == "" {
		t.Skip("local original HUD data not supplied")
	}
	common, err := os.ReadFile(filepath.Join(dir, "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	initialLevel, err := os.ReadFile(filepath.Join(dir, "000B00E5.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	initialTerrain, err := DecodeTerrain(initialLevel)
	if err != nil {
		t.Fatal(err)
	}
	initialHUD, err := DecodePlayerPresentation(common, initialTerrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	cases := [][3]int{{0, 3, 39}, {1234567, 2, 20}, {9999999, 1, 0}, {7654321, 4, 17}}
	for _, resource := range []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"} {
		level, err := os.ReadFile(filepath.Join(dir, resource+".decoded"))
		if err != nil {
			t.Fatal(err)
		}
		terrain, err := DecodeTerrain(level)
		if err != nil {
			t.Fatal(err)
		}
		p := *initialHUD
		p.HUD = RemapPalette(p.HUD, initialTerrain.Palette, terrain.Palette)
		p.ScoreFont.Image = RemapPalette(p.ScoreFont.Image, initialTerrain.Palette, terrain.Palette)
		p.LivesFont.Image = RemapPalette(p.LivesFont.Image, initialTerrain.Palette, terrain.Palette)
		for index, v := range cases {
			picture := image.NewNRGBA(image.Rect(0, 0, 320, 200))
			draw.Draw(picture, image.Rect(0, 192, 320, 200), p.HUD, image.Point{}, draw.Src)
			paint := func(font Font, text string, x, y int) {
				for _, char := range text {
					glyph := int(char - '0')
					draw.Draw(picture, image.Rect(x, y, x+8, y+8), font.Image, image.Pt(glyph*8, 0), draw.Src)
					x += 8
				}
			}
			xscore, xlives, xshield := p.ScoreX, p.LivesX, p.ShieldX
			if index == 3 {
				xscore, xlives, xshield = 240, 176, 190
			}
			paint(p.ScoreFont, fmt.Sprintf("%07d", v[0]), xscore, p.ScoreY)
			paint(p.LivesFont, fmt.Sprint(v[1]), xlives, p.LivesY)
			for col := 0; col < 39; col++ {
				c := color.NRGBA{A: 255}
				if col < v[2] {
					a := terrain.Palette[14]
					c = color.NRGBA{R: a[0], G: a[1], B: a[2], A: 255}
				}
				draw.Draw(picture, image.Rect(xshield+col, p.ShieldY, xshield+col+1, p.ShieldY+3), &image.Uniform{C: c}, image.Point{}, draw.Src)
			}
			data, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "analysis", fmt.Sprintf("hud-native-%d.bin", index)))
			if err != nil {
				t.Fatal(err)
			}
			for y := 192; y < 200; y++ {
				for x := 0; x < 320; x++ {
					got := picture.NRGBAAt(x, y)
					want := originalShopPixel(data, x, y, terrain.Palette)
					if got != want {
						t.Fatalf("HUD %s case %d at %d,%d Go %v original %v", resource, index, x, y, got, want)
					}
				}
			}
		}
	}
}
