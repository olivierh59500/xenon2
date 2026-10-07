package visualassets

import (
	"encoding/binary"
	"fmt"
	"image"
)

// Sprite describes the original masked image and its positioning anchor.
type Sprite struct {
	Name      string        `json:"name"`
	AnchorX   int           `json:"anchor_x"`
	AnchorY   int           `json:"anchor_y"`
	Width     int           `json:"width"`
	Height    int           `json:"height"`
	Collision *CollisionBox `json:"collision,omitempty"`
	Image     *image.NRGBA  `json:"-"`
}

// CollisionBox is relative to the actor anchor, independent of atlas storage.
type CollisionBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DecodeActorSprite also exports the actor's four-word collision prefix.
func DecodeActorSprite(data []byte, start int, name string, palette [16][4]uint8) (*Sprite, error) {
	sprite, err := DecodeSprite(data, start, name, palette)
	if err != nil {
		return nil, err
	}
	if start < 8 {
		return nil, fmt.Errorf("actor %s collision prefix is missing", name)
	}
	value := func(index int) int { return int(int16(binary.BigEndian.Uint16(data[start-8+index*2:]))) }
	sprite.Collision = &CollisionBox{X: value(0) - sprite.AnchorX, Y: value(1) - sprite.AnchorY, Width: value(2), Height: value(3)}
	if sprite.Collision.Width < 0 || sprite.Collision.Height < 0 {
		return nil, fmt.Errorf("actor %s has invalid collision bounds", name)
	}
	return sprite, nil
}

// DecodeSprite reads a four-word image header followed by row-interleaved four
// color planes and a final mask. Storage rounds widths up to a sixteen-pixel word.
func DecodeSprite(data []byte, start int, name string, palette [16][4]uint8) (*Sprite, error) {
	if start < 0 || start+8 > len(data) {
		return nil, fmt.Errorf("sprite %s header is outside its source", name)
	}
	word := func(i int) int { return int(binary.BigEndian.Uint16(data[start+i*2:])) }
	width, height := word(2), word(3)+1
	if width < 1 || width > 32 || height < 1 || height > 256 {
		return nil, fmt.Errorf("sprite %s has invalid dimensions %dx%d", name, width, height)
	}
	storageWidth := (width + 15) / 16 * 16
	bytes := storageWidth / 8 * 5 * height
	if start+8+bytes > len(data) {
		return nil, fmt.Errorf("sprite %s pixels are truncated", name)
	}
	picture := image.NewNRGBA(image.Rect(0, 0, storageWidth, height))
	if err := drawPlanarLayout(picture, image.Point{}, data[start+8:start+8+bytes], storageWidth, height, 4, true, true, palette); err != nil {
		return nil, err
	}
	return &Sprite{Name: name, AnchorX: int(int16(word(0))), AnchorY: int(int16(word(1))), Width: width, Height: height, Image: picture}, nil
}
