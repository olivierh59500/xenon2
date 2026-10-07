package visualassets

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

type ShopCell struct {
	ID, Column, Row  int
	X, Y             int
	CursorX, CursorY int
}

type ShopControl struct {
	ID     string `json:"id"`
	Sprite string `json:"sprite"`
	X, Y   int
}

type ShopAmbient struct {
	X, Y      int
	Animation ItemAnimation `json:"animation"`
}

// ShopScene retains the original static composition, portrait, grid and font.
type ShopScene struct {
	CashFont                                                Font `json:"cash_font"`
	MoneyX, MoneyY, MoneyDigits                             int
	Messages                                                map[string]string `json:"messages"`
	Ambient                                                 []ShopAmbient     `json:"ambient"`
	Width, Height                                           int
	Cells                                                   []ShopCell    `json:"cells"`
	Controls                                                []ShopControl `json:"controls"`
	PortraitX, PortraitY, PortraitWidth, PortraitHeight     int
	PortraitFrames                                          int
	DialogueX, DialogueY, DialogueColumns, DialogueLineStep int
	Font                                                    Font         `json:"font"`
	ControlArt                                              SpriteAtlas  `json:"control_art"`
	Base                                                    *image.NRGBA `json:"-"`
	Portraits                                               *image.NRGBA `json:"-"`
}

// DecodeShopScene composes the six verified original word-plane copies and
// exports the merchant's four mouth poses with their exact clipping margins.
func DecodeShopScene(data []byte, palette [16][4]uint8) (*ShopScene, error) {
	const base = 0x54e00
	if len(data) < 0x6476e-base {
		return nil, fmt.Errorf("shop presentation source truncated")
	}
	s := &ShopScene{Width: 320, Height: 200, PortraitX: 215, PortraitY: 9, PortraitWidth: 96, PortraitHeight: 96, PortraitFrames: 4, DialogueX: 220, DialogueY: 118, DialogueColumns: 20, DialogueLineStep: 7, Base: image.NewNRGBA(image.Rect(0, 0, 320, 200)), Portraits: image.NewNRGBA(image.Rect(0, 0, 384, 96))}
	for i := 0; i < 6; i++ {
		at := 0x55224 - base + i*10
		source := int(binary.BigEndian.Uint32(data[at:]))
		destination := int(binary.BigEndian.Uint16(data[at+4:]))
		width := (int(binary.BigEndian.Uint16(data[at+6:])) + 1) * 16
		height := int(binary.BigEndian.Uint16(data[at+8:])) + 1
		picture, err := decodeWordPlanes(data, source-base, width, height, palette)
		if err != nil {
			return nil, err
		}
		x, y := (destination%160)*8, destination/160
		if x+width > 320 || y+height > 200 {
			return nil, fmt.Errorf("shop static copy leaves framebuffer")
		}
		draw.Draw(s.Base, image.Rect(x, y, x+width, y+height), picture, image.Point{}, draw.Src)
	}
	if binary.BigEndian.Uint32(data[0x55224-base+60:]) != 0 {
		return nil, fmt.Errorf("unsupported shop static copy count")
	}
	upper, err := decodeWordPlanes(data, 0x5f43e-base, 112, 52, palette)
	if err != nil {
		return nil, err
	}
	lower, err := decodeWordPlanes(data, 0x5ff9e-base, 112, 10, palette)
	if err != nil {
		return nil, err
	}
	for frame := 0; frame < 4; frame++ {
		mouth, err := decodeWordPlanes(data, 0x601ce-base+frame*0x770, 112, 34, palette)
		if err != nil {
			return nil, err
		}
		x := frame * 96
		draw.Draw(s.Portraits, image.Rect(x, 0, x+96, 52), upper, image.Pt(7, 0), draw.Src)
		draw.Draw(s.Portraits, image.Rect(x, 52, x+96, 86), mouth, image.Pt(7, 0), draw.Src)
		draw.Draw(s.Portraits, image.Rect(x, 86, x+96, 96), lower, image.Pt(7, 0), draw.Src)
	}
	for row := 0; row < 4; row++ {
		for column := 0; column < 5; column++ {
			id := row*5 + column
			at := 0x56ac2 - base + id*16
			x, y := int(binary.BigEndian.Uint16(data[at+2:])), int(binary.BigEndian.Uint16(data[at+4:]))
			s.Cells = append(s.Cells, ShopCell{ID: id, Column: column, Row: row, X: x, Y: y, CursorX: x + 10, CursorY: y + 31})
		}
	}
	var pictures []*Sprite
	for _, control := range []struct {
		id            string
		address, x, y int
	}{{"cursor", 0x57850, 0, 0}, {"exit", 0x5b218, 4, 168}, {"buy", 0x5b458, 45, 168}, {"sell", 0x5b698, 45, 168}} {
		picture, err := DecodeSprite(data, control.address-base, "shop-control-"+control.id, palette)
		if err != nil {
			return nil, err
		}
		pictures = append(pictures, picture)
		s.Controls = append(s.Controls, ShopControl{ID: control.id, Sprite: picture.Name, X: control.x, Y: control.y})
	}
	known := make(map[int]string)
	add := func(address int) (string, error) {
		if name, ok := known[address]; ok {
			return name, nil
		}
		name := fmt.Sprintf("shop-ambient-image-%03d", len(known))
		picture, err := DecodeSprite(data, address-base, name, palette)
		if err != nil {
			return "", err
		}
		known[address] = name
		pictures = append(pictures, picture)
		return name, nil
	}
	for index := 0; index < 7; index++ {
		at := 0x56eb8 - base + index*8
		root := int(binary.BigEndian.Uint32(data[at+4:])) - base
		animation := ItemAnimation{ID: fmt.Sprintf("merchant-ambient-%d", index)}
		cursor := root
		positions := make(map[int]int)
		for len(animation.Frames) < 256 {
			if cursor < 0 || cursor+8 > len(data) {
				return nil, fmt.Errorf("merchant animation is truncated")
			}
			positions[cursor] = len(animation.Frames)
			address := int(binary.BigEndian.Uint32(data[cursor:]))
			cursor += 4
			if address == 0 {
				loop := int(binary.BigEndian.Uint32(data[cursor:])) - base
				position, ok := positions[loop]
				if !ok {
					return nil, fmt.Errorf("merchant animation loop leaves its list")
				}
				animation.LoopFrom = position
				break
			}
			name, err := add(address)
			if err != nil {
				return nil, err
			}
			animation.Frames = append(animation.Frames, name)
		}
		if len(animation.Frames) == 0 || len(animation.Frames) == 256 {
			return nil, fmt.Errorf("invalid merchant animation")
		}
		s.Ambient = append(s.Ambient, ShopAmbient{X: int(binary.BigEndian.Uint16(data[at:])), Y: int(binary.BigEndian.Uint16(data[at+2:])), Animation: animation})
	}
	s.ControlArt = packSprites(pictures)
	s.Font = Font{Width: 4, Height: 8, Columns: 16, Image: image.NewNRGBA(image.Rect(0, 0, 64, 32))}
	for code := 32; code <= 90; code++ {
		s.Font.Characters += string(rune(code))
		glyph := code - 32
		source := 0x63fee - base + glyph*32
		if source+32 > len(data) {
			return nil, fmt.Errorf("shop font is truncated")
		}
		for y := 0; y < 8; y++ {
			for x := 0; x < 4; x++ {
				index := 0
				for plane := 0; plane < 4; plane++ {
					if data[source+y*4+plane]&(1<<uint(7-x)) != 0 {
						index |= 1 << uint(plane)
					}
				}
				c := palette[index]
				s.Font.Image.SetNRGBA((glyph%16)*4+x, (glyph/16)*8+y, color.NRGBA{R: c[0], G: c[1], B: c[2], A: c[3]})
			}
		}
	}
	s.MoneyX, s.MoneyY, s.MoneyDigits = 136, 172, 7
	s.Messages = make(map[string]string)
	for _, message := range []struct {
		id      string
		address int
	}{{"sell-question", 0x56e50}, {"buy-question", 0x56e6e}, {"out-of-stock", 0x55aa6}, {"pay-request", 0x55acd}, {"another-item", 0x55ae2}, {"sale-complete", 0x55c2e}, {"sale-price", 0x55dc8}, {"buy-price", 0x55dd6}, {"cannot-fit", 0x558ba}, {"last-level-welcome", 0x56fd8}, {"last-level-offer", 0x56ffc}, {"last-level-warning", 0x5700f}} {
		start := message.address - base
		end := start
		for end < len(data) && data[end] != 0 {
			end++
		}
		if end >= len(data) || end-start > 512 {
			return nil, fmt.Errorf("shop message is unterminated")
		}
		s.Messages[message.id] = string(data[start:end])
	}
	return s, nil
}

// DecodeShopCashFont recovers the original seven-digit eight-pixel display.
func DecodeShopCashFont(common []byte, palette [16][4]uint8) (Font, error) {
	const start = 0x1981e
	if len(common) < start+320 {
		return Font{}, fmt.Errorf("cash digit font truncated")
	}
	font := Font{Characters: "0123456789", Width: 8, Height: 8, Columns: 10, Image: image.NewNRGBA(image.Rect(0, 0, 80, 8))}
	for glyph := 0; glyph < 10; glyph++ {
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				index := 0
				for plane := 0; plane < 4; plane++ {
					if common[start+glyph*32+y*4+plane]&(1<<uint(7-x)) != 0 {
						index |= 1 << uint(plane)
					}
				}
				c := palette[index]
				font.Image.SetNRGBA(glyph*8+x, y, color.NRGBA{R: c[0], G: c[1], B: c[2], A: c[3]})
			}
		}
	}
	return font, nil
}

func decodeWordPlanes(data []byte, start, width, height int, palette [16][4]uint8) (*image.NRGBA, error) {
	if width < 16 || width%16 != 0 || height < 1 || start < 0 || start+width/8*4*height > len(data) {
		return nil, fmt.Errorf("word-plane artwork is truncated")
	}
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			word := start + y*(width/8*4) + (x/16)*8
			index := 0
			for plane := 0; plane < 4; plane++ {
				if data[word+plane*2+(x%16)/8]&(1<<uint(7-x%8)) != 0 {
					index |= 1 << uint(plane)
				}
			}
			c := palette[index]
			picture.SetNRGBA(x, y, color.NRGBA{R: c[0], G: c[1], B: c[2], A: c[3]})
		}
	}
	return picture, nil
}
