package visualassets

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateOriginalCreditOverlaps(t *testing.T) {
	dir := os.Getenv("XENON2_SHOP_TEST_DIR")
	if dir == "" {
		t.Skip("local original credit references not supplied")
	}
	common, err := os.ReadFile(filepath.Join(dir, "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	title, err := DecodeTitleArt(common)
	if err != nil {
		t.Fatal(err)
	}
	data, err := DecodePresentation(common, title.Palette)
	if err != nil {
		t.Fatal(err)
	}
	for pair := 0; pair < 6; pair++ {
		image := ComposeCreditOverlap(data.Font, data.Credits[pair*2], title.Palette)
		raw, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "analysis", fmt.Sprintf("credit-overlap-native-%d.bin", pair)))
		if err != nil {
			t.Fatal(err)
		}
		for y := 0; y < 22; y++ {
			for x := 0; x < 320; x++ {
				got := image.NRGBAAt(x, y)
				want := originalShopPixel(raw, x, y+112, title.Palette)
				if got != want {
					t.Fatalf("pair %d at %d,%d Go %v original %v", pair, x, y, got, want)
				}
			}
		}
	}
	t.Log("Compared all pixels of the six overlapping credit transitions.")
}
