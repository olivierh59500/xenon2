package visualassets

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateOriginalShopScene(t *testing.T) {
	dir := os.Getenv("XENON2_SHOP_TEST_DIR")
	if dir == "" {
		t.Skip("local original shop references not supplied")
	}
	shop, err := os.ReadFile(filepath.Join(dir, "05c400f8.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	common, err := os.ReadFile(filepath.Join(dir, "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	catalogue, err := DecodeShopCatalogue(shop)
	if err != nil {
		t.Fatal(err)
	}
	art, err := DecodeShopArt(shop, common, catalogue)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := DecodeShopScene(shop, art.Palette)
	if err != nil {
		t.Fatal(err)
	}
	scene.CashFont, err = DecodeShopCashFont(common, art.Palette)
	if err != nil {
		t.Fatal(err)
	}
	if len(scene.Cells) != 20 || len(scene.Ambient) != 7 || scene.PortraitFrames != 4 || len(scene.Font.Characters) != 59 || scene.MoneyX != 136 || scene.MoneyY != 172 || scene.MoneyDigits != 7 {
		t.Fatal("original shop geometry incomplete")
	}
	if scene.Cells[0].X != 4 || scene.Cells[19].X != 168 || scene.Cells[19].Y != 127 {
		t.Fatal("original five-column grid changed")
	}
	if scene.Base.NRGBAAt(8, 8).A != 0 {
		t.Fatal("static copies overwrite untouched monitor pixels")
	}
	if scene.Messages["sell-question"] != "WHAT DO YOU WANT TO SELL ME ?" || scene.Messages["buy-question"] != "O.K. WHAT DO YOU WANT TO BUY ?" {
		t.Fatal("original dialogue changed")
	}
	b, err := json.Marshal(scene)
	if err != nil {
		t.Fatal(err)
	}
	var portable ShopScene
	if err = json.Unmarshal(b, &portable); err != nil {
		t.Fatal(err)
	}
	if portable.PortraitX != 215 || portable.PortraitWidth != 96 || portable.CashFont.Width != 8 || len(portable.ControlArt.Sprites) < 4 {
		t.Fatal("export loses shop geometry")
	}
	for _, sprite := range scene.ControlArt.Sprites {
		if sprite.X+sprite.Width > scene.ControlArt.Image.Bounds().Dx() || sprite.Y+sprite.Height > scene.ControlArt.Image.Bounds().Dy() {
			t.Fatal("merchant animation leaves atlas")
		}
	}
	oracle := filepath.Dir(dir) + "/analysis"
	frame, err := os.ReadFile(filepath.Join(oracle, "shop-native-base.bin"))
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			got := scene.Base.NRGBAAt(x, y)
			if got.A == 0 {
				got = color.NRGBA{A: 255}
			}
			if want := originalShopPixel(frame, x, y, art.Palette); got != want {
				t.Fatalf("static copy at %d,%d: export %v original %v", x, y, got, want)
			}
		}
	}
	for pose := 0; pose < 4; pose++ {
		frame, err = os.ReadFile(filepath.Join(oracle, fmt.Sprintf("shop-native-portrait-%d.bin", pose)))
		if err != nil {
			t.Fatal(err)
		}
		for y := 0; y < 96; y++ {
			for x := 0; x < 96; x++ {
				got := scene.Portraits.NRGBAAt(pose*96+x, y)
				want := originalShopPixel(frame, 215+x, 9+y, art.Palette)
				if got != want {
					t.Fatalf("merchant pose %d at %d,%d: export %v original %v", pose, x, y, got, want)
				}
			}
		}
	}
}

func originalShopPixel(frame []byte, x, y int, palette [16][4]uint8) color.NRGBA {
	index := 0
	for plane := 0; plane < 4; plane++ {
		if frame[y*160+plane*40+x/8]&(1<<uint(7-x%8)) != 0 {
			index |= 1 << uint(plane)
		}
	}
	c := palette[index]
	return color.NRGBA{R: c[0], G: c[1], B: c[2], A: c[3]}
}

func TestShopSceneRejectsTruncatedInputs(t *testing.T) {
	if _, err := DecodeShopScene(make([]byte, 16), [16][4]uint8{}); err == nil {
		t.Fatal("truncated shop accepted")
	}
	if _, err := DecodeShopCashFont(make([]byte, 16), [16][4]uint8{}); err == nil {
		t.Fatal("truncated cash font accepted")
	}
}
