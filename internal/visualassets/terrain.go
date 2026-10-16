// Package visualassets exports original artwork into code-free game resources.
package visualassets

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"sort"
)

const (
	levelBase  = 0x54e00
	MapColumns = 20
	MapRows    = 300
	TileSize   = 16
)

type Tile struct {
	ID     uint16 `json:"id"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Masked bool   `json:"masked,omitempty"`
}

// Terrain carries dimensions, stable atlas IDs and colors rather than Amiga
// addresses or instructions. Tile ID zero denotes untouched background.
type Terrain struct {
	Columns           int               `json:"columns"`
	Rows              int               `json:"rows"`
	TileSize          int               `json:"tile_size"`
	MidShopStockLimit int               `json:"mid_shop_stock_limit"`
	EndShopStockLimit int               `json:"end_shop_stock_limit"`
	Palette           [16][4]uint8      `json:"palette"`
	Tiles             []Tile            `json:"tiles"`
	Map               []uint16          `json:"map"`
	Atlas             *image.NRGBA      `json:"-"`
	Background        *image.NRGBA      `json:"-"`
	SourceTileIDs     map[uint16]uint16 `json:"-"`
}

// DecodeTerrain reads only the documented graphics and map fields from a
// decoded level. The branch vectors and gameplay code are not copied out.
func DecodeTerrain(data []byte) (*Terrain, error) {
	return DecodeTerrainWithTiles(data, nil)
}

// DecodeTerrainWithTiles also includes verified tile frames used by mutable
// terrain encounters. Extra references come from their decoded frame tables.
func DecodeTerrainWithTiles(data []byte, extra []uint16) (*Terrain, error) {
	if len(data) < 0x40 {
		return nil, fmt.Errorf("decoded level header is truncated")
	}
	background, err := offset(data, 0, 320*192/4)
	if err != nil {
		return nil, err
	}
	mapStart, err := offset(data, 4, MapColumns*MapRows*2)
	if err != nil {
		return nil, err
	}
	graphics, err := offset(data, 8, 1)
	if err != nil {
		return nil, err
	}
	paletteStart, err := offset(data, 0x1c, 32)
	if err != nil {
		return nil, err
	}
	terrain := &Terrain{Columns: MapColumns, Rows: MapRows, TileSize: TileSize, Map: make([]uint16, MapColumns*MapRows)}
	terrain.EndShopStockLimit = int(binary.BigEndian.Uint16(data[0x3c:]))
	terrain.MidShopStockLimit = int(binary.BigEndian.Uint16(data[0x3e:]))
	for i := range terrain.Palette {
		word := binary.BigEndian.Uint16(data[paletteStart+i*2:])
		if word&0xf888 != 0 {
			return nil, fmt.Errorf("palette entry %d is not an original 3-bit RGB color", i)
		}
		terrain.Palette[i] = [4]uint8{uint8(word>>8&7) * 34, uint8(word>>4&7) * 34, uint8(word&7) * 34, 255}
	}
	unique := map[uint16]bool{}
	for i := range terrain.Map {
		raw := binary.BigEndian.Uint16(data[mapStart+i*2:])
		if raw != 0 {
			unique[raw] = true
		}
	}
	for _, code := range extra {
		if code != 0 {
			unique[code] = true
		}
	}
	codes := make([]int, 0, len(unique))
	for code := range unique {
		codes = append(codes, int(code))
	}
	sort.Ints(codes)
	terrain.Atlas = image.NewNRGBA(image.Rect(0, 0, 16*TileSize, ((len(codes)+15)/16)*TileSize))
	ids := make(map[uint16]uint16, len(codes))
	terrain.SourceTileIDs = ids
	for ordinal, code := range codes {
		id := uint16(ordinal + 1)
		masked := code&0x8000 != 0
		start := graphics + (code&0x7fff)*16
		rowBytes := 8
		if masked {
			rowBytes = 10
		}
		if start < 0 || start+16*rowBytes > len(data) {
			return nil, fmt.Errorf("tile %d exceeds decoded graphics", ordinal)
		}
		x, y := (ordinal%16)*16, (ordinal/16)*16
		if err := drawPlanar(terrain.Atlas, image.Pt(x, y), data[start:start+16*rowBytes], 16, 16, 4, masked, terrain.Palette); err != nil {
			return nil, err
		}
		terrain.Tiles = append(terrain.Tiles, Tile{ID: id, X: x, Y: y, Masked: masked})
		ids[uint16(code)] = id
	}
	for i := range terrain.Map {
		terrain.Map[i] = ids[binary.BigEndian.Uint16(data[mapStart+i*2:])]
	}
	terrain.Background = image.NewNRGBA(image.Rect(0, 0, 320, 192))
	if err := drawPlanar(terrain.Background, image.Point{}, data[background:background+15360], 320, 192, 2, false, terrain.Palette); err != nil {
		return nil, err
	}
	return terrain, nil
}

func offset(data []byte, field, length int) (int, error) {
	address := uint64(binary.BigEndian.Uint32(data[field:]))
	if address < levelBase || address-levelBase+uint64(length) > uint64(len(data)) {
		return 0, fmt.Errorf("visual field 0x%x is outside the decoded level", field)
	}
	return int(address - levelBase), nil
}

// drawPlanar handles row-interleaved Amiga planes. A mask one selects sprite
// pixels, while zero preserves the already drawn background at that pixel.
func drawPlanar(destination *image.NRGBA, point image.Point, data []byte, width, height, planes int, masked bool, palette [16][4]uint8) error {
	return drawPlanarLayout(destination, point, data, width, height, planes, masked, false, palette)
}

// Terrain tiles put their mask first; ordinary masked sprites put it last.
func drawPlanarLayout(destination *image.NRGBA, point image.Point, data []byte, width, height, planes int, masked, maskLast bool, palette [16][4]uint8) error {
	if width <= 0 || width%16 != 0 || height <= 0 || planes < 1 || planes > 4 {
		return fmt.Errorf("invalid planar dimensions")
	}
	planeBytes := width / 8
	rowBytes := planeBytes * planes
	if masked {
		rowBytes += planeBytes
	}
	if len(data) < rowBytes*height {
		return fmt.Errorf("planar image is truncated")
	}
	for y := range height {
		row := data[y*rowBytes : (y+1)*rowBytes]
		planeStart := 0
		maskStart := 0
		if masked && !maskLast {
			planeStart = planeBytes
		}
		if masked && maskLast {
			maskStart = planeBytes * planes
		}
		for x := range width {
			bit := byte(1 << uint(7-x%8))
			index := 0
			for plane := range planes {
				if row[planeStart+plane*planeBytes+x/8]&bit != 0 {
					index |= 1 << plane
				}
			}
			c := palette[index]
			if masked && row[maskStart+x/8]&bit == 0 {
				c[3] = 0
			}
			destination.SetNRGBA(point.X+x, point.Y+y, color.NRGBA{R: c[0], G: c[1], B: c[2], A: c[3]})
		}
	}
	return nil
}

type Font struct {
	Characters string       `json:"characters"`
	Width      int          `json:"width"`
	Height     int          `json:"height"`
	Columns    int          `json:"columns"`
	Image      *image.NRGBA `json:"-"`
}

// DecodeFont exports the common 16-pixel font using its original ASCII table.
func DecodeFont(executable []byte, palette [16][4]uint8) (*Font, error) {
	const lookup, bank, count = 0x86fc, 0xb666, 38
	if len(executable) < bank+count*128 {
		return nil, fmt.Errorf("common font bank is truncated")
	}
	characters := string(executable[lookup : lookup+count])
	if characters != "ABCDEFGHIJKLMNOPQRSTUVWXYZ.:0123456789" {
		return nil, fmt.Errorf("unsupported common font character table")
	}
	font := &Font{Characters: characters, Width: 16, Height: 16, Columns: 10, Image: image.NewNRGBA(image.Rect(0, 0, 160, 64))}
	for i := range count {
		if err := drawPlanar(font.Image, image.Pt(i%10*16, i/10*16), executable[bank+i*128:bank+(i+1)*128], 16, 16, 4, false, palette); err != nil {
			return nil, err
		}
	}
	return font, nil
}
