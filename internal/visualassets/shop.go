package visualassets

import (
	"encoding/binary"
	"fmt"
	"strings"
)

type ShopItem struct {
	ID    string   `json:"id"`
	Order int      `json:"order"`
	Name  string   `json:"name"`
	Lines []string `json:"lines"`
	Price int      `json:"price"`
}

type ShopCatalogue struct {
	Items []ShopItem `json:"items"`
}

// DecodeShopCatalogue exports labels and base prices. Purchase routines and
// hardware addresses are deliberately absent from the exported catalogue.
func DecodeShopCatalogue(data []byte) (*ShopCatalogue, error) {
	const names, prices, count = 0x557b8 - levelBase, 0x55888 - levelBase, 25
	if len(data) < prices+count*2 {
		return nil, fmt.Errorf("shop catalogue is truncated")
	}
	ids := []string{"forward-shot", "advice", "speedup", "health-power-1", "autofire", "super-nashwan-power", "health-power-2", "rear-shot", "mine-small", "side-shot", "electro-ball", "powerup", "mine-large", "double-shot", "cannon", "dive", "missile-launcher", "laser", "drone", "flamer", "bomb", "extra-life", "homing-missile", "protection", "bitmap-shades"}
	catalogue := &ShopCatalogue{}
	for i, id := range ids {
		start, err := offset(data, names+i*4, 1)
		if err != nil {
			return nil, err
		}
		end := start
		for end < len(data) && data[end] != 0 && end-start < 64 {
			end++
		}
		if end == len(data) || end-start == 64 {
			return nil, fmt.Errorf("shop item %d label is unterminated", i+1)
		}
		text := string(data[start:end])
		for _, c := range text {
			if c != '\r' && (c < 32 || c > 126) {
				return nil, fmt.Errorf("shop item %d label is not printable ASCII", i+1)
			}
		}
		lines := strings.Split(text, "\r")
		catalogue.Items = append(catalogue.Items, ShopItem{ID: id, Order: i + 1, Name: strings.Join(lines, " "), Lines: lines, Price: int(binary.BigEndian.Uint16(data[prices+i*2:]))})
	}
	if catalogue.Items[0].Name != "FORWARD SHOT" || catalogue.Items[24].Name != "BITMAP SHADES" {
		return nil, fmt.Errorf("unsupported shop catalogue version")
	}
	return catalogue, nil
}
