package visualassets

import (
	"image"
	"testing"
)

func TestPlanarPixelAndMaskSemantics(t *testing.T) {
	var palette [16][4]uint8
	for i := range palette {
		palette[i] = [4]uint8{uint8(i), 0, 0, 255}
	}
	data := make([]byte, 10)
	data[0] = 0x80 // Only the leftmost pixel is visible.
	data[2] = 0xc0 // Plane zero on the first two pixels.
	data[6] = 0x80 // Plane two on the first pixel: palette index five.
	image := image.NewNRGBA(image.Rect(0, 0, 16, 1))
	if err := drawPlanar(image, image.Bounds().Min, data, 16, 1, 4, true, palette); err != nil {
		t.Fatal(err)
	}
	if got := image.NRGBAAt(0, 0); got.R != 5 || got.A != 255 {
		t.Fatalf("visible pixel=%v", got)
	}
	if got := image.NRGBAAt(1, 0); got.R != 1 || got.A != 0 {
		t.Fatalf("masked pixel=%v", got)
	}
	if err := drawPlanar(image, image.Bounds().Min, data[:9], 16, 1, 4, true, palette); err == nil {
		t.Fatal("truncated image accepted")
	}
}
