// Command export-assets converts local decoded artwork into code-free PNG/JSON.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"xenon2/internal/assetimport"
	"xenon2/internal/visualassets"
)

func main() {
	input := flag.String("analysis", ".local/imported", "excluded decoded source directory")
	output := flag.String("output", "assets/runtime", "excluded exported resource directory")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected positional arguments"))
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		fail(err)
	}
	executable, err := os.ReadFile(filepath.Join(*input, assetimport.Executable.Name+".decoded"))
	if err != nil {
		fail(err)
	}
	firstLevel, err := os.ReadFile(filepath.Join(*input, assetimport.Levels[0].Name+".decoded"))
	if err != nil {
		fail(err)
	}
	firstTerrain, err := visualassets.DecodeTerrain(firstLevel)
	if err != nil {
		fail(err)
	}
	palette := firstTerrain.Palette
	shopData, err := os.ReadFile(filepath.Join(*input, assetimport.Levels[5].Name+".decoded"))
	if err != nil {
		fail(err)
	}
	shop, err := visualassets.DecodeShopCatalogue(shopData)
	if err != nil {
		fail(err)
	}
	commonArt, err := visualassets.DecodeCommonActorArtWithEquipment(executable, shopData, shop, palette)
	if err != nil {
		fail(err)
	}
	for index, resource := range assetimport.Levels[:5] {
		data, err := os.ReadFile(filepath.Join(*input, resource.Name+".decoded"))
		if err != nil {
			fail(err)
		}
		var fixed *visualassets.FixedTiles
		var extraTiles []uint16
		fixed, extraTiles, err = visualassets.DecodeFixedTiles(index+1, data)
		if err != nil {
			fail(err)
		}
		baseTerrain, err := visualassets.DecodeTerrain(data)
		if err != nil {
			fail(err)
		}
		guardians, guardianTiles, err := visualassets.DecodeGuardianArt(index+1, data, baseTerrain.Palette)
		if err != nil {
			fail(err)
		}
		extraTiles = append(extraTiles, guardianTiles...)
		groups, guardianParts, err := visualassets.DecodeCompoundGuardianArt(index+1, data, baseTerrain.Palette)
		if err != nil {
			fail(err)
		}
		extraTiles = append(extraTiles, visualassets.GuardianGroupTileCodes(groups)...)
		terrain, err := visualassets.DecodeTerrainWithTiles(data, extraTiles)
		if err != nil {
			fail(fmt.Errorf("level %d: %w", index+1, err))
		}
		prefix := filepath.Join(*output, fmt.Sprintf("level-%d", index+1))
		if guardians != nil {
			if err := visualassets.RemapGuardianTiles(guardians, terrain.SourceTileIDs); err != nil {
				fail(err)
			}
			if err := writeJSON(prefix+"-guardians.json", guardians); err != nil {
				fail(err)
			}
			if err := writePNG(prefix+"-guardians.png", guardians.Atlas.Image); err != nil {
				fail(err)
			}
		}
		if len(groups) > 0 {
			if err := visualassets.RemapGuardianGroupTiles(groups, terrain.SourceTileIDs); err != nil {
				fail(err)
			}
			if err := writeJSON(prefix+"-guardian-groups.json", struct {
				Groups []visualassets.GuardianGroup `json:"groups"`
				Atlas  visualassets.SpriteAtlas     `json:"atlas"`
			}{groups, guardianParts}); err != nil {
				fail(err)
			}
			if err := writePNG(prefix+"-guardian-parts.png", guardianParts.Image); err != nil {
				fail(err)
			}
		}
		if fixed != nil {
			if err := visualassets.RemapFixedTiles(fixed, terrain.SourceTileIDs); err != nil {
				fail(err)
			}
			if err := writeJSON(prefix+"-fixed-tiles.json", fixed); err != nil {
				fail(err)
			}
		}
		if err := writeJSON(prefix+".json", terrain); err != nil {
			fail(err)
		}
		if err := writePNG(prefix+"-tiles.png", terrain.Atlas); err != nil {
			fail(err)
		}
		if err := writePNG(prefix+"-background.png", terrain.Background); err != nil {
			fail(err)
		}
		paths, err := visualassets.DecodePaths(data, executable)
		if err != nil {
			fail(fmt.Errorf("level %d paths: %w", index+1, err))
		}
		if err := writeJSON(prefix+"-paths.json", paths); err != nil {
			fail(err)
		}
		encounters, err := visualassets.DecodeEncounters(data, paths)
		if err != nil {
			fail(fmt.Errorf("level %d encounters: %w", index+1, err))
		}
		if err := writeJSON(prefix+"-encounters.json", encounters); err != nil {
			fail(err)
		}
		rules, err := visualassets.DecodeLevelRules(index+1, data, executable, terrain.Palette, encounters)
		if err != nil {
			fail(fmt.Errorf("level %d rules: %w", index+1, err))
		}
		if err := writeJSON(prefix+"-rules.json", rules); err != nil {
			fail(err)
		}
		if err := writePNG(prefix+"-shots.png", rules.EnemyShots.Image); err != nil {
			fail(err)
		}
		actors, err := visualassets.DecodeWaveActors(data, terrain.Palette, encounters, &commonArt)
		if err != nil {
			fail(fmt.Errorf("level %d actors: %w", index+1, err))
		}
		if err := writeJSON(prefix+"-actors.json", actors); err != nil {
			fail(err)
		}
		if err := writePNG(prefix+"-actors.png", actors.Atlas.Image); err != nil {
			fail(err)
		}
		fixedSprites, err := visualassets.DecodeFixedSprites(index+1, data, terrain.Palette)
		if err != nil {
			fail(err)
		}
		if err := writeJSON(prefix+"-fixed-sprites.json", fixedSprites); err != nil {
			fail(err)
		}
		if err := writePNG(prefix+"-fixed-sprites.png", fixedSprites.Atlas.Image); err != nil {
			fail(err)
		}
		fmt.Printf("Exported level %d: %d tiles, %d map rows.\n", index+1, len(terrain.Tiles), terrain.Rows)
	}
	font, err := visualassets.DecodeFont(executable, palette)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(filepath.Join(*output, "font.json"), font); err != nil {
		fail(err)
	}
	if err := writePNG(filepath.Join(*output, "font.png"), font.Image); err != nil {
		fail(err)
	}
	ship, err := visualassets.DecodeActorSprite(executable, 0xebe2, "player-ship", palette)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(filepath.Join(*output, "player-ship.json"), ship); err != nil {
		fail(err)
	}
	if err := writePNG(filepath.Join(*output, "player-ship.png"), ship.Image); err != nil {
		fail(err)
	}
	if err := writeJSON(filepath.Join(*output, "shop.json"), shop); err != nil {
		fail(err)
	}
	shipArt, err := visualassets.DecodeShipArt(executable, palette)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(filepath.Join(*output, "ships.json"), shipArt); err != nil {
		fail(err)
	}
	if err := writePNG(filepath.Join(*output, "ships.png"), shipArt.Atlas.Image); err != nil {
		fail(err)
	}
	if err := writeJSON(filepath.Join(*output, "common-actors.json"), commonArt); err != nil {
		fail(err)
	}
	if err := writePNG(filepath.Join(*output, "common-actors.png"), commonArt.Image); err != nil {
		fail(err)
	}
	playerShots, err := visualassets.DecodePlayerShotArt(executable, palette)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(filepath.Join(*output, "player-shots.json"), playerShots); err != nil {
		fail(err)
	}
	if err := writePNG(filepath.Join(*output, "player-shots.png"), playerShots.Image); err != nil {
		fail(err)
	}
	shopArt, err := visualassets.DecodeShopArt(shopData, executable, shop)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(filepath.Join(*output, "shop-art.json"), shopArt); err != nil {
		fail(err)
	}
	if err := writePNG(filepath.Join(*output, "shop-art.png"), shopArt.Atlas.Image); err != nil {
		fail(err)
	}
	title, err := visualassets.DecodeTitleArt(executable)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(filepath.Join(*output, "title.json"), title); err != nil {
		fail(err)
	}
	if err := writePNG(filepath.Join(*output, "title.png"), title.Image); err != nil {
		fail(err)
	}
	stencil, err := visualassets.DecodePlayerTerrainStencil(executable)
	if err != nil {
		fail(err)
	}
	if err := writeJSON(filepath.Join(*output, "player-terrain-stencil.json"), stencil); err != nil {
		fail(err)
	}
	if err := writePNG(filepath.Join(*output, "player-terrain-stencil.png"), stencil.Image); err != nil {
		fail(err)
	}
	fmt.Println("Exported the original common font. No executable bytes were exported.")
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
func writePNG(path string, picture image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(file, picture); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
