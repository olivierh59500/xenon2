// Command export-presentation recovers original menu and attract captions.
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
	output := flag.String("output", "assets/runtime", "exported resource directory")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected positional arguments"))
	}
	data, err := os.ReadFile(filepath.Join(*input, assetimport.Executable.Name+".decoded"))
	if err != nil {
		fail(err)
	}
	title, err := visualassets.DecodeTitleArt(data)
	if err != nil {
		fail(err)
	}
	p, err := visualassets.DecodePresentation(data, title.Palette)
	if err != nil {
		fail(err)
	}
	if err = os.MkdirAll(*output, 0755); err != nil {
		fail(err)
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		fail(err)
	}
	if err = os.WriteFile(filepath.Join(*output, "presentation.json"), append(b, '\n'), 0644); err != nil {
		fail(err)
	}
	f, err := os.Create(filepath.Join(*output, "presentation-font.png"))
	if err != nil {
		fail(err)
	}
	if err = png.Encode(f, p.Font.Image); err != nil {
		f.Close()
		fail(err)
	}
	if err = f.Close(); err != nil {
		fail(err)
	}

	f, err = os.Create(filepath.Join(*output, "caption-zoom.png"))
	if err != nil {
		fail(err)
	}
	if err = png.Encode(f, p.TextZoom.Image); err != nil {
		f.Close()
		fail(err)
	}
	if err = f.Close(); err != nil {
		fail(err)
	}
	f, err = os.Create(filepath.Join(*output, "title-zoom.png"))
	if err != nil {
		fail(err)
	}
	if err = png.Encode(f, p.LogoZoom.Image); err != nil {
		f.Close()
		fail(err)
	}
	if err = f.Close(); err != nil {
		fail(err)
	}
	fmt.Println("Exported original three-choice menu, credits and presentation font.")
	levelData, err := os.ReadFile(filepath.Join(*input, assetimport.Levels[0].Name+".decoded"))
	if err != nil {
		fail(err)
	}
	terrain, err := visualassets.DecodeTerrain(levelData)
	if err != nil {
		fail(err)
	}
	player, err := visualassets.DecodePlayerPresentation(data, terrain.Palette)
	if err != nil {
		fail(err)
	}
	for _, resource := range []struct {
		name    string
		picture image.Image
	}{{"hud-base", player.HUD}, {"hud-score-font", player.ScoreFont.Image}, {"hud-lives-font", player.LivesFont.Image}} {
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
	b, err = json.MarshalIndent(player, "", "  ")
	if err != nil {
		fail(err)
	}
	if err = os.WriteFile(filepath.Join(*output, "player-presentation.json"), append(b, '\n'), 0644); err != nil {
		fail(err)
	}
	f, err = os.Create(filepath.Join(*output, "player-presentation.png"))
	if err != nil {
		fail(err)
	}
	if err = png.Encode(f, player.Atlas.Image); err != nil {
		f.Close()
		fail(err)
	}
	if err = f.Close(); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
