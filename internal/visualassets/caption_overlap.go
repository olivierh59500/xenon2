package visualassets

import (
	"image"
	"image/color"
	"strings"
)

// ComposeCreditOverlap retains the single credit transition that copies the
// complete line before OR-ing its first shrinking pass into the same words.
// It consumes exported font pixels, not original code or binary state.
func ComposeCreditOverlap(font Font, text string, palette [16][4]uint8) *image.NRGBA {
	const width, height = 320, 22
	indices := make([]uint8, width*height)
	colors := make(map[color.NRGBA]uint8, 16)
	for index, c := range palette {
		colors[color.NRGBA{c[0], c[1], c[2], c[3]}] = uint8(index)
	}
	for letter, character := range text {
		if letter >= 20 {
			break
		}
		glyph := strings.IndexRune(font.Characters, character)
		if glyph < 0 {
			continue
		}
		sx, sy := glyph%font.Columns*font.Width, glyph/font.Columns*font.Height
		for y := 0; y < font.Height; y++ {
			for x := 0; x < font.Width; x++ {
				indices[y*width+letter*16+x] = colors[font.Image.NRGBAAt(sx+x, sy+y)]
			}
		}
	}
	// The fifteen-column mask drops source column/row eight in each word.
	xs, ys := []int{}, []int{}
	for x := 0; x < 16; x++ {
		if uint16(0xff7f)&(1<<uint(15-x)) != 0 {
			xs = append(xs, x)
		}
	}
	for y := 0; y < 22; y++ {
		if uint16(0xff7f)&(1<<uint(15-y%16)) != 0 {
			ys = append(ys, y)
		}
	}
	for letter, character := range text {
		if letter >= 20 {
			break
		}
		glyph := strings.IndexRune(font.Characters, character)
		if glyph < 0 {
			continue
		}
		sx, sy := glyph%font.Columns*font.Width, glyph/font.Columns*font.Height
		left := 10 + letter*15
		nextWord := (left/16 + 1) * 16
		for y, sourceY := range ys {
			if left%16+15 > 16 {
				for x := nextWord; x < min(width, nextWord+16); x++ {
					indices[y*width+x] = 0
				}
			}
			for x, sourceX := range xs {
				dst := left + x
				if dst >= width {
					continue
				}
				value := colors[font.Image.NRGBAAt(sx+sourceX, sy+sourceY)]
				if dst < nextWord {
					indices[y*width+dst] |= value
				} else {
					indices[y*width+dst] = value
				}
			}
		}
	}
	result := image.NewNRGBA(image.Rect(0, 0, width, height))
	for at, index := range indices {
		c := palette[index]
		copy(result.Pix[at*4:at*4+4], c[:])
	}
	return result
}
