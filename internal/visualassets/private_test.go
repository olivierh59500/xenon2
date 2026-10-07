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
	for levelIndex, file := range files {
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
			fixed, extra, err := DecodeFixedTiles(levelIndex+1, data)
			if err != nil {
				t.Fatal(err)
			}
			if fixed != nil {
				expanded, err := DecodeTerrainWithTiles(data, extra)
				if err != nil {
					t.Fatal(err)
				}
				if err := RemapFixedTiles(fixed, expanded.SourceTileIDs); err != nil {
					t.Fatal(err)
				}
				for _, kind := range fixed.Kinds {
					for _, variant := range kind.Variants {
						patches := append([]TilePatch{variant.Initial, variant.Destroyed}, variant.Frames...)
						for _, patch := range patches {
							if len(patch.Tiles) != patch.Columns*patch.Rows {
								t.Fatal("fixed tile dimensions differ")
							}
							for _, id := range patch.Tiles {
								if int(id) > len(expanded.Tiles) {
									t.Fatal("fixed tile ID outside expanded atlas")
								}
							}
						}
					}
				}
			}
			paths, err := DecodePaths(data, mustReadPrivate(t, filepath.Join(directory, "XenonII.decoded")))
			if err != nil {
				t.Fatal(err)
			}
			encounters, err := DecodeEncounters(data, paths)
			if err != nil {
				t.Fatal(err)
			}
			actors, err := DecodeWaveActors(data, terrain.Palette, encounters)
			if err != nil {
				t.Fatal(err)
			}
			names := map[string]bool{}
			for _, sprite := range actors.Atlas.Sprites {
				if names[sprite.Name] || sprite.Collision == nil {
					t.Fatalf("invalid actor sprite%+v", sprite)
				}
				names[sprite.Name] = true
			}
			for _, kind := range actors.Kinds {
				for _, part := range kind.Parts {
					for _, frame := range part.Animation.Frames {
						if !names[frame.Sprite] {
							t.Fatalf("missing animation frame%s", frame.Sprite)
						}
					}
					for _, frame := range part.HeadingFrames {
						if !names[frame] {
							t.Fatalf("missing heading frame%s", frame)
						}
					}
					for _, animation := range part.EntryAnimations {
						for _, frame := range animation.Frames {
							if !names[frame.Sprite] {
								t.Fatalf("missing edge frame%s", frame.Sprite)
							}
						}
					}
				}
			}
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
	title, err := DecodeTitleArt(executable)
	if err != nil {
		t.Fatal(err)
	}
	if title.Width != 208 || title.Height != 54 || title.X != 48 || title.Y != 20 {
		t.Fatal("original title placement differs")
	}
	shopData := mustReadPrivate(t, filepath.Join(directory, "05c400f8.decoded"))
	catalogue, err := DecodeShopCatalogue(shopData)
	if err != nil {
		t.Fatal(err)
	}
	commonArt, err := DecodeCommonActorArtWithEquipment(executable, shopData, catalogue, palette)
	if err != nil {
		t.Fatal(err)
	}
	if len(commonArt.Sprites) != 198 || len(commonArt.Equipment) != 25 {
		t.Fatal("common equipment image bank differs")
	}
}

func mustReadPrivate(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
