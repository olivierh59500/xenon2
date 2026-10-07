package app

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/presentation"
)

// These optional references are window-only captures of the supplied disk.
// The fixed viewport calibration is shared by every caption and comes from
// the logo's raster correspondence. It is not refitted to individual captions.
func TestOriginalAttractCapturesMatchProductionGPUOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_ATTRACT_DIR")
	if root == "" {
		t.Skip("local native attract captures not supplied")
	}
	for _, sample := range []struct{ nativeFrame, sourcePass int }{{10, 155}, {25, 177}, {65, 239}} {
		file, err := os.Open(filepath.Join(root, fmt.Sprintf("%05d.png", sample.nativeFrame)))
		if err != nil {
			t.Fatal(err)
		}
		native, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if native.Bounds().Dx() != 960 || native.Bounds().Dy() != 628 {
			t.Fatal("capture does not use the calibrated 960 by 628 window geometry")
		}
		g, _ := renderIntegrationGame(t)
		g.Screen = PresentationScreen
		g.fade = nil
		g.director = presentation.NewDirector(&g.Bundle.Presentation)
		for pass := 0; pass <= sample.sourcePass; pass++ {
			g.director.Advance(presentation.Input{})
		}
		actual := renderIntegrationPixels(t, g, fmt.Sprintf("native-attract-pass-%d", sample.sourcePass))
		differences, colored := 0, 0
		for y := 80; y < 180; y++ {
			for x := 0; x < ScreenWidth; x++ {
				nx, ny := nativeAttractPixel(x, y)
				a, b := introRGB(actual, x, y), introRGB(native, nx, ny)
				if introOrange(a) != introOrange(b) {
					differences++
				}
				if sample.nativeFrame == 25 && a != ([3]uint8{}) && nativeFlatPatch(native, nx, ny, b) {
					if a != b {
						t.Fatalf("flat native caption palette differs at %d,%d: Go%v native%v", x, y, a, b)
					}
					colored++
				}
			}
		}
		if differences != 0 {
			t.Fatalf("native frame%d/source pass%d differs at%d caption mask pixels", sample.nativeFrame, sample.sourcePass, differences)
		}
		if sample.nativeFrame == 25 && colored < 500 {
			t.Fatalf("too few flat palette samples: %d", colored)
		}
		t.Logf("Native frame%d/source pass%d: exact caption mask, %d flat color samples", sample.nativeFrame, sample.sourcePass, colored)
	}
}

func nativeAttractPixel(x, y int) (int, int) {
	return int(math.Floor(124.25 + (float64(x)+.5)*2.22)), int(math.Floor(75 + (float64(y)+.5)*20/9))
}

func introRGB(picture image.Image, x, y int) [3]uint8 {
	r, g, b, _ := picture.At(x, y).RGBA()
	return [3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
}

func introOrange(c [3]uint8) bool {
	return c[0] > 80 && int(c[0]) > int(c[1])+20 && int(c[1]) > int(c[2])+10
}

func nativeFlatPatch(picture image.Image, x, y int, color [3]uint8) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if introRGB(picture, x+dx, y+dy) != color {
				return false
			}
		}
	}
	return true
}
