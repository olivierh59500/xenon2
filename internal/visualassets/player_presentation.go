package visualassets

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

type PlayerPresentation struct {
	HUD                                           *image.NRGBA `json:"-"`
	ScoreFont, LivesFont                          Font
	ScoreX, ScoreY, LivesX, LivesY                int
	ShieldX, ShieldY, ShieldColumns, ShieldHeight int
	DiveFrames                                    []string    `json:"dive_frames"`
	Atlas                                         SpriteAtlas `json:"atlas"`
}

// DecodePlayerPresentation recovers the four original dive silhouettes.
func DecodePlayerPresentation(common []byte, palette [16][4]uint8) (*PlayerPresentation, error) {
	if len(common) < 0x614a+16 {
		return nil, fmt.Errorf("player dive table truncated")
	}
	p := &PlayerPresentation{}
	var pictures []*Sprite
	for phase := 1; phase <= 4; phase++ {
		address := int(binary.BigEndian.Uint32(common[0x614a+(phase-1)*4:]))
		name := fmt.Sprintf("player-dive-%d", phase)
		picture, err := DecodeActorSprite(common, address, name, palette)
		if err != nil {
			return nil, err
		}
		pictures = append(pictures, picture)
		p.DiveFrames = append(p.DiveFrames, name)
	}
	p.Atlas = packSprites(pictures)
	var err error
	p.HUD, err = decodeWordPlanes(common, 0x19a9e, 320, 8, palette)
	if err != nil {
		return nil, err
	}
	p.ScoreFont, err = decodeByteFont(common, 0x1981e, "0123456789", palette)
	if err != nil {
		return nil, err
	}
	p.LivesFont, err = decodeByteFont(common, 0x1995e, "0123456789", palette)
	if err != nil {
		return nil, err
	}
	p.ScoreX, p.ScoreY, p.LivesX, p.LivesY = 24, 192, 136, 193
	p.ShieldX, p.ShieldY, p.ShieldColumns, p.ShieldHeight = 91, 195, 39, 3
	return p, nil
}

func decodeByteFont(data []byte, start int, characters string, palette [16][4]uint8) (Font, error) {
	if start < 0 || start+len(characters)*32 > len(data) {
		return Font{}, fmt.Errorf("HUD byte font truncated")
	}
	f := Font{Characters: characters, Width: 8, Height: 8, Columns: len(characters), Image: image.NewNRGBA(image.Rect(0, 0, len(characters)*8, 8))}
	for glyph := range characters {
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				index := 0
				for plane := 0; plane < 4; plane++ {
					if data[start+glyph*32+y*4+plane]&(1<<uint(7-x)) != 0 {
						index |= 1 << uint(plane)
					}
				}
				c := palette[index]
				f.Image.SetNRGBA(glyph*8+x, y, color.NRGBA{R: c[0], G: c[1], B: c[2], A: c[3]})
			}
		}
	}
	return f, nil
}
