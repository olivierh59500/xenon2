package visualassets

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

type TitleArt struct {
	X       int          `json:"x"`
	Y       int          `json:"y"`
	Width   int          `json:"width"`
	Height  int          `json:"height"`
	Palette [16][4]uint8 `json:"palette"`
	Image   *image.NRGBA `json:"-"`
}

// DecodeTitleArt reads the original 208 × 54 logo. Its four planes are grouped
// by sixteen-pixel words, unlike the row-interleaved tile and sprite banks.
func DecodeTitleArt(common []byte) (*TitleArt, error) {
	const pixels, paletteStart, width, height = 0xc966, 0x86b2, 208, 54
	if len(common) < pixels+width/8*4*height || len(common) < paletteStart+32 {
		return nil, fmt.Errorf("title artwork is truncated")
	}
	art := &TitleArt{X: 48, Y: 20, Width: width, Height: height, Image: image.NewNRGBA(image.Rect(0, 0, width, height))}
	for i := range art.Palette {
		value := binary.BigEndian.Uint16(common[paletteStart+i*2:])
		if value&0xf888 != 0 {
			return nil, fmt.Errorf("unsupported title palette")
		}
		art.Palette[i] = [4]uint8{uint8(value>>8&7) * 34, uint8(value>>4&7) * 34, uint8(value&7) * 34, 255}
	}
	for y := range height {
		for x := range width {
			index := 0
			wordGroup := pixels + y*(width/8*4) + (x/16)*8
			for plane := range 4 {
				if common[wordGroup+plane*2+(x%16)/8]&(1<<uint(7-x%8)) != 0 {
					index |= 1 << plane
				}
			}
			c := art.Palette[index]
			art.Image.SetNRGBA(x, y, color.NRGBA{R: c[0], G: c[1], B: c[2], A: c[3]})
		}
	}
	return art, nil
}
