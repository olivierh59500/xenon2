package visualassets

import (
	"encoding/binary"
	"testing"
)

func TestActorCollisionPrefixRemainsAnchorRelative(t *testing.T) {
	// Prefix x=2,y=3,w=5,h=6; image anchor x=7,y=8,w=9,h=1.
	words := []int16{2, 3, 5, 6, 7, 8, 9, 0}
	data := make([]byte, 26)
	for i, word := range words {
		binary.BigEndian.PutUint16(data[i*2:], uint16(word))
	}
	data[16] = 0x80 // First color plane.
	data[24] = 0x80 // Mask follows all four color planes.
	var palette [16][4]uint8
	for i := range palette {
		palette[i][3] = 255
	}
	sprite, err := DecodeActorSprite(data, 8, "test", palette)
	if err != nil {
		t.Fatal(err)
	}
	if sprite.Collision.X != -5 || sprite.Collision.Y != -5 || sprite.Collision.Width != 5 || sprite.Collision.Height != 6 || sprite.Width != 9 || sprite.Image.Bounds().Dx() != 16 {
		t.Fatalf("decoded actor=%+v collision=%+v", sprite, sprite.Collision)
	}
	if sprite.Image.NRGBAAt(0, 0).A != 255 || sprite.Image.NRGBAAt(1, 0).A != 0 {
		t.Fatal("sprite mask position differs from the native blitter")
	}
	// Empty original boxes are meaningful animation states, not malformed data.
	binary.BigEndian.PutUint16(data[4:], 0)
	binary.BigEndian.PutUint16(data[6:], 0)
	if _, err := DecodeActorSprite(data, 8, "test", palette); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeActorSprite(data[:25], 8, "test", palette); err == nil {
		t.Fatal("truncated masked pixels accepted")
	}
}

func TestWideSpriteKeepsColorPlanesBeforeMask(t *testing.T) {
	data := make([]byte, 36)
	words := []int16{0, 0, 20, 1, 0, 0, 20, 0}
	for i, word := range words {
		binary.BigEndian.PutUint16(data[i*2:], uint16(word))
	}
	data[16] = 0x80
	data[18] = 0x40 // Plane0 pixels0 and17.
	data[24] = 0x80 // Plane2 pixel0: color5.
	data[28] = 0x40 // Plane3 pixel1 is hidden by the final mask.
	data[32] = 0x80
	data[34] = 0x40
	var palette [16][4]uint8
	for i := range palette {
		palette[i] = [4]uint8{uint8(i), 0, 0, 255}
	}
	sprite, err := DecodeActorSprite(data, 8, "wide", palette)
	if err != nil {
		t.Fatal(err)
	}
	if pixel := sprite.Image.NRGBAAt(0, 0); pixel.R != 5 || pixel.A != 255 {
		t.Fatalf("first word pixel=%v", pixel)
	}
	if pixel := sprite.Image.NRGBAAt(17, 0); pixel.R != 1 || pixel.A != 255 {
		t.Fatalf("second word pixel=%v", pixel)
	}
	if pixel := sprite.Image.NRGBAAt(1, 0); pixel.R != 8 || pixel.A != 0 {
		t.Fatalf("masked plane3 pixel=%v", pixel)
	}
}
