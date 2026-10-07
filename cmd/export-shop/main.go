// Command export-shop recovers the original shop composition and small font.
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
	output := flag.String("output", "assets/runtime", "excluded exported resources")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected positional arguments"))
	}
	data, err := os.ReadFile(filepath.Join(*input, assetimport.Levels[5].Name+".decoded"))
	if err != nil {
		fail(err)
	}
	common, err := os.ReadFile(filepath.Join(*input, assetimport.Executable.Name+".decoded"))
	if err != nil {
		fail(err)
	}
	catalogue, err := visualassets.DecodeShopCatalogue(data)
	if err != nil {
		fail(err)
	}
	art, err := visualassets.DecodeShopArt(data, common, catalogue)
	if err != nil {
		fail(err)
	}
	scene, err := visualassets.DecodeShopScene(data, art.Palette)
	if err != nil {
		fail(err)
	}
	scene.CashFont, err = visualassets.DecodeShopCashFont(common, art.Palette)
	if err != nil {
		fail(err)
	}
	if err = os.MkdirAll(*output, 0755); err != nil {
		fail(err)
	}
	for _, resource := range []struct {
		name    string
		picture image.Image
	}{{"shop-base", scene.Base}, {"shop-portraits", scene.Portraits}, {"shop-font", scene.Font.Image}, {"shop-controls", scene.ControlArt.Image}, {"shop-cash-font", scene.CashFont.Image}} {
		file, err := os.Create(filepath.Join(*output, resource.name+".png"))
		if err != nil {
			fail(err)
		}
		if err = png.Encode(file, resource.picture); err != nil {
			file.Close()
			fail(err)
		}
		if err = file.Close(); err != nil {
			fail(err)
		}
	}
	encoded, err := json.MarshalIndent(scene, "", "  ")
	if err != nil {
		fail(err)
	}
	if err = os.WriteFile(filepath.Join(*output, "shop-scene.json"), append(encoded, '\n'), 0644); err != nil {
		fail(err)
	}
	fmt.Printf("Exported original shop: %d cells, %d portrait poses and %d small glyphs.\n", len(scene.Cells), scene.PortraitFrames, len(scene.Font.Characters))
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
