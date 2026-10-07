package visualassets

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

func decodeLaserSprites(common []byte, palette [16][4]uint8) ([]*Sprite, error) {
	if len(common) < 0x6494+12 {
		return nil, fmt.Errorf("laser cap tables are truncated")
	}
	var sprites []*Sprite
	for tier := range 3 {
		for _, cap := range []struct {
			name  string
			table int
		}{{"top", 0x6494}, {"bottom", 0x6488}} {
			address := int(binary.BigEndian.Uint32(common[cap.table+tier*4:]))
			name := fmt.Sprintf("laser-beam-%s-%d", cap.name, tier)
			picture, err := DecodeSprite(common, address, name, palette)
			if err != nil {
				return nil, err
			}
			sprites = append(sprites, picture)
		}
		width := 8 + tier*4
		picture := image.NewNRGBA(image.Rect(0, 0, width, 1))
		indices := make([]int, width)
		for i := range indices {
			indices[i] = 7
		}
		indices[0], indices[1], indices[2] = 5, 6, 8
		indices[width-3], indices[width-2], indices[width-1] = 8, 6, 5
		for x, index := range indices {
			c := palette[index]
			picture.SetNRGBA(x, 0, color.NRGBA{R: c[0], G: c[1], B: c[2], A: 255})
		}
		sprites = append(sprites, &Sprite{Name: fmt.Sprintf("laser-beam-stem-%d", tier), Width: width, Height: 1, Image: picture})
	}
	return sprites, nil
}
