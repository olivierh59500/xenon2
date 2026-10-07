package visualassets

import (
	"image"
	"image/color"
	"testing"
)

func TestSharedPaletteRemappingPreservesTransparencyAndSource(t *testing.T) {
	var from, to [16][4]uint8
	from[1] = [4]uint8{34, 34, 34, 255}
	to[1] = [4]uint8{68, 170, 34, 255}
	picture := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	picture.SetNRGBA(0, 0, color.NRGBA{34, 34, 34, 255})
	picture.SetNRGBA(1, 0, color.NRGBA{34, 34, 34, 0})
	remapped := RemapPalette(picture, from, to)
	if got := remapped.NRGBAAt(0, 0); got != (color.NRGBA{68, 170, 34, 255}) {
		t.Fatal(got)
	}
	if got := remapped.NRGBAAt(1, 0); got != (color.NRGBA{34, 34, 34, 0}) {
		t.Fatalf("transparent mask changed: %v", got)
	}
	if got := picture.NRGBAAt(0, 0); got != (color.NRGBA{34, 34, 34, 255}) {
		t.Fatal("shared source image mutated")
	}
}
