package visualassets

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
)

type SpriteRegion struct {
	Name      string        `json:"name"`
	X         int           `json:"x"`
	Y         int           `json:"y"`
	AnchorX   int           `json:"anchor_x"`
	AnchorY   int           `json:"anchor_y"`
	Width     int           `json:"width"`
	Height    int           `json:"height"`
	Collision *CollisionBox `json:"collision,omitempty"`
}

type SpriteAtlas struct {
	Sprites []SpriteRegion `json:"sprites"`
	Image   *image.NRGBA   `json:"-"`
}

type ItemAnimation struct {
	ID       string   `json:"id"`
	Frames   []string `json:"frames"`
	LoopFrom int      `json:"loop_from"`
}

type ShopArt struct {
	Palette    [16][4]uint8    `json:"palette"`
	Animations []ItemAnimation `json:"animations"`
	Atlas      SpriteAtlas     `json:"atlas"`
}

// DecodeShopArt follows only the original image lists and their checked loop
// references. The exported animation uses named sprite frames and loop indices.
func DecodeShopArt(shop, common []byte, catalogue *ShopCatalogue) (*ShopArt, error) {
	const paletteStart = 0x6474e - levelBase
	if len(shop) < paletteStart+32 {
		return nil, fmt.Errorf("shop palette is truncated")
	}
	art := &ShopArt{}
	for i := range art.Palette {
		word := binary.BigEndian.Uint16(shop[paletteStart+i*2:])
		if word&0xf888 != 0 {
			return nil, fmt.Errorf("invalid shop color")
		}
		art.Palette[i] = [4]uint8{uint8(word>>8&7) * 34, uint8(word>>4&7) * 34, uint8(word&7) * 34, 255}
	}
	images := make([]*Sprite, 0)
	imageNames := map[uint32]string{}
	add := func(address uint32, name string) (string, error) {
		if existing, ok := imageNames[address]; ok {
			return existing, nil
		}
		data, start := common, int(address)
		if address >= levelBase {
			data, start = shop, int(address-levelBase)
		}
		picture, err := DecodeSprite(data, start, name, art.Palette)
		if err != nil {
			return "", err
		}
		images = append(images, picture)
		imageNames[address] = name
		return name, nil
	}
	for index, address := range []uint32{0x5bdbc, 0x5c0ec} {
		if _, err := add(address, fmt.Sprintf("shop-background-%d", index+1)); err != nil {
			return nil, err
		}
	}
	for itemIndex, item := range catalogue.Items {
		field := 0x5581c - levelBase + itemIndex*4
		start, err := offset(shop, field, 4)
		if err != nil {
			return nil, err
		}
		cursor := start
		frameOffsets := map[int]int{}
		animation := ItemAnimation{ID: item.ID}
		for len(animation.Frames) < 256 {
			if cursor+4 > len(shop) {
				return nil, fmt.Errorf("shop %s animation is truncated", item.ID)
			}
			address := binary.BigEndian.Uint32(shop[cursor:])
			frameOffsets[cursor] = len(animation.Frames)
			cursor += 4
			if address == 0 {
				if cursor+4 > len(shop) {
					return nil, fmt.Errorf("shop animation loop is truncated")
				}
				loop := binary.BigEndian.Uint32(shop[cursor:])
				loopIndex, ok := frameOffsets[int(loop)-levelBase]
				if !ok || len(animation.Frames) == 0 {
					return nil, fmt.Errorf("shop %s animation loop leaves its frame list", item.ID)
				}
				animation.LoopFrom = loopIndex
				break
			}
			name, err := add(address, fmt.Sprintf("shop-image-%03d", len(images)))
			if err != nil {
				return nil, err
			}
			animation.Frames = append(animation.Frames, name)
		}
		if len(animation.Frames) == 256 {
			return nil, fmt.Errorf("shop %s animation is too long", item.ID)
		}
		art.Animations = append(art.Animations, animation)
	}
	art.Atlas = packSprites(images)
	return art, nil
}

type ShipArt struct {
	SteeringFrames []string    `json:"steering_frames"`
	Atlas          SpriteAtlas `json:"atlas"`
}

// DecodeShipArt retains all five banking images and the thirteen original
// steering lookup positions; the central frame is not the complete ship bank.
func DecodeShipArt(common []byte, palette [16][4]uint8) (*ShipArt, error) {
	const table = 0x60d0
	if len(common) < table+13*4 {
		return nil, fmt.Errorf("ship steering table is truncated")
	}
	art := &ShipArt{}
	images := make([]*Sprite, 0, 5)
	names := map[uint32]string{}
	for i := range 13 {
		address := binary.BigEndian.Uint32(common[table+i*4:])
		name, ok := names[address]
		if !ok {
			name = fmt.Sprintf("player-ship-%d", len(images))
			picture, err := DecodeActorSprite(common, int(address), name, palette)
			if err != nil {
				return nil, err
			}
			images = append(images, picture)
			names[address] = name
		}
		art.SteeringFrames = append(art.SteeringFrames, name)
	}
	if len(images) != 5 {
		return nil, fmt.Errorf("unsupported ship steering bank")
	}
	art.Atlas = packSprites(images)
	return art, nil
}

func packSprites(sprites []*Sprite) SpriteAtlas {
	const columns, cellWidth = 16, 32
	rowHeight := 1
	for _, sprite := range sprites {
		rowHeight = max(rowHeight, sprite.Height)
	}
	atlas := SpriteAtlas{Image: image.NewNRGBA(image.Rect(0, 0, columns*cellWidth, ((len(sprites)+columns-1)/columns)*rowHeight))}
	for i, sprite := range sprites {
		x, y := (i%columns)*cellWidth, (i/columns)*rowHeight
		draw.Draw(atlas.Image, image.Rect(x, y, x+sprite.Image.Bounds().Dx(), y+sprite.Height), sprite.Image, image.Point{}, draw.Src)
		atlas.Sprites = append(atlas.Sprites, SpriteRegion{Name: sprite.Name, X: x, Y: y, AnchorX: sprite.AnchorX, AnchorY: sprite.AnchorY, Width: sprite.Width, Height: sprite.Height, Collision: sprite.Collision})
	}
	return atlas
}
