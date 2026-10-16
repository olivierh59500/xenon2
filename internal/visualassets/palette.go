package visualassets

import (
	"image"
	"image/color"
)

// RemapPalette recolors exported opaque pixels without changing source images
// or transparent coverage. The supported source palette has unique colors.
func RemapPalette(picture *image.NRGBA, from, to [16][4]uint8) *image.NRGBA {
	if from == to {
		return picture
	}
	colors := make(map[color.NRGBA]color.NRGBA, 16)
	for i, a := range from {
		b := to[i]
		colors[color.NRGBA{R: a[0], G: a[1], B: a[2], A: a[3]}] = color.NRGBA{R: b[0], G: b[1], B: b[2], A: b[3]}
	}
	out := image.NewNRGBA(picture.Bounds())
	for y := picture.Bounds().Min.Y; y < picture.Bounds().Max.Y; y++ {
		for x := picture.Bounds().Min.X; x < picture.Bounds().Max.X; x++ {
			c := picture.NRGBAAt(x, y)
			if mapped, ok := colors[c]; ok {
				c = mapped
			}
			out.SetNRGBA(x, y, c)
		}
	}
	return out
}
