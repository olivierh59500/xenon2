package visualassets

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateLevelArtworkOptional(t *testing.T) {
	directory := os.Getenv("XENON2_PRIVATE_DECODED_DIR")
	if directory == "" {
		t.Skip("set XENON2_PRIVATE_DECODED_DIR to the local imported analysis directory")
	}
	files := []struct {
		name  string
		tiles int
	}{{"000B00E5", 176}, {"00FA00FE", 210}, {"02020113", 182}, {"031F0159", 351}, {"04820138", 157}}
	var palette [16][4]uint8
	for _, file := range files {
		t.Run(file.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(directory, file.name+".decoded"))
			if err != nil {
				t.Fatal(err)
			}
			terrain, err := DecodeTerrain(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(terrain.Tiles) != file.tiles || len(terrain.Map) != 6000 || terrain.Background.Bounds().Dx() != 320 || terrain.Background.Bounds().Dy() != 192 {
				t.Fatal("original artwork dimensions differ")
			}
			for _, id := range terrain.Map {
				if int(id) > len(terrain.Tiles) {
					t.Fatalf("map tile ID%d outside atlas", id)
				}
			}
			palette = terrain.Palette
			binary.BigEndian.PutUint32(data[4:], 0xffffffff)
			if _, err := DecodeTerrain(data); err == nil {
				t.Fatal("out-of-range map accepted")
			}
		})
	}
	executable, err := os.ReadFile(filepath.Join(directory, "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	font, err := DecodeFont(executable, palette)
	if err != nil {
		t.Fatal(err)
	}
	if len(font.Characters) != 38 || font.Image.Bounds().Dx() != 160 || font.Image.Bounds().Dy() != 64 {
		t.Fatal("original font dimensions differ")
	}
}
