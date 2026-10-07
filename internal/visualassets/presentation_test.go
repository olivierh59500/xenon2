package visualassets

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateOriginalPresentationResources(t *testing.T) {
	dir := os.Getenv("XENON2_SHOP_TEST_DIR")
	if dir == "" {
		t.Skip("local original presentation not supplied")
	}
	common, err := os.ReadFile(filepath.Join(dir, "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	title, err := DecodeTitleArt(common)
	if err != nil {
		t.Fatal(err)
	}
	p, err := DecodePresentation(common, title.Palette)
	if err != nil {
		t.Fatal(err)
	}
	if p.Font.Width != 16 || p.Font.Height != 22 || len(p.Font.Characters) != 41 || len(p.Menu) != 3 || len(p.Credits) != 12 || len(p.HighScoreLines) != 10 {
		t.Fatal("original presentation inventory changed")
	}
	if p.Menu[0].CenterY != 60 || p.Menu[1].CenterY != 84 || p.Menu[2].CenterY != 108 || strings.TrimSpace(p.Ready) != "GET READY PLAYER 1" {
		t.Fatal("presentation geometry or captions changed")
	}
	for _, line := range p.HighScoreLines {
		if len(line) != 15 || !strings.HasSuffix(line, ":::") {
			t.Fatalf("bad original score row %q", line)
		}
	}
	if len(p.StarColors) != 8 || p.StarColors[0] != 7 || p.StarColors[7] != 9 {
		t.Fatal("original star colors changed")
	}
	for step, region := range p.LogoZoom.Sprites {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "analysis", fmt.Sprintf("logo-zoom-native-%d.bin", step+1)))
		if err != nil {
			t.Fatal(err)
		}
		for y := 0; y < region.Height; y++ {
			for x := 0; x < region.Width; x++ {
				got := p.LogoZoom.Image.NRGBAAt(region.X+x, region.Y+y)
				want := originalShopPixel(data, p.LogoZoomX[step]+x, p.LogoZoomY[step]+y, title.Palette)
				if got != want {
					t.Fatalf("logo zoom %d at %d,%d: export %v original %v", step+1, x, y, got, want)
				}
			}
		}
	}
	byName := make(map[string]SpriteRegion)
	for _, region := range p.TextZoom.Sprites {
		byName[region.Name] = region
	}
	for step := 1; step <= 16; step++ {
		picture := image.NewNRGBA(image.Rect(0, 0, 320, 200))
		draw.Draw(picture, picture.Bounds(), &image.Uniform{C: color.NRGBA{A: 255}}, image.Point{}, draw.Src)
		x, y := 160-step*10, (200-step)/2
		if step == 16 {
			y = 92
		}
		for _, char := range p.Ready {
			glyph := strings.IndexRune(p.Font.Characters, char)
			if glyph >= 0 {
				region := byName[p.TextZoomRegions[step-1][glyph]]
				draw.Draw(picture, image.Rect(x, y, x+region.Width, y+region.Height), p.TextZoom.Image, image.Pt(region.X, region.Y), draw.Src)
			}
			x += step
		}
		data, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "analysis", fmt.Sprintf("text-zoom-native-%d.bin", step)))
		if err != nil {
			t.Fatal(err)
		}
		for yy := 0; yy < 200; yy++ {
			for xx := 0; xx < 320; xx++ {
				got := picture.NRGBAAt(xx, yy)
				want := originalShopPixel(data, xx, yy, title.Palette)
				if got != want {
					t.Fatalf("text zoom %d at %d,%d: export %v original %v", step, xx, yy, got, want)
				}
			}
		}
	}
}

func TestPresentationRejectsTruncatedInput(t *testing.T) {
	if _, err := DecodePresentation([]byte{0}, [16][4]uint8{}); err == nil {
		t.Fatal("truncated presentation accepted")
	}
}
