// Package app presents the verified Go simulation through Ebitengine.
package app

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"io/fs"
	"os"

	runtimeassets "xenon2/assets/runtime"
	"xenon2/internal/audio"
	"xenon2/internal/visualassets"
)

// LevelAssets holds exported graphics and ordinary gameplay descriptors.
type LevelAssets struct {
	GuardianGroups []visualassets.GuardianGroup
	GuardianParts  *visualassets.SpriteAtlas
	Guardians      *visualassets.Guardians
	Terrain        visualassets.Terrain
	Paths          visualassets.Paths
	Encounters     visualassets.Encounters
	Actors         visualassets.Actors
	FixedSprites   visualassets.FixedSprites
	FixedTiles     visualassets.FixedTiles
	Rules          visualassets.LevelRules
}

// Bundle retains no packed disk resources, original program or machine state.
type Bundle struct {
	PlayerPresentation visualassets.PlayerPresentation
	Presentation       visualassets.Presentation
	ShopScene          visualassets.ShopScene
	Levels             [5]*LevelAssets
	Title              visualassets.TitleArt
	Font               visualassets.Font
	Ships              visualassets.ShipArt
	Stencil            visualassets.PlayerTerrainStencil
	Common             visualassets.SpriteAtlas
	Shop               visualassets.ShopCatalogue
	ShopArt            visualassets.ShopArt
	AudioBank          *audio.Bank
	Waveforms          map[string][]byte
}

func LoadEmbedded() (*Bundle, error) { return LoadFS(runtimeassets.Files) }

// LoadFS reads only the public PNG/JSON/PCM export schema from its root.
func LoadFS(resources fs.FS) (*Bundle, error) {
	b := &Bundle{}
	common := []struct {
		name     string
		metadata any
		picture  **image.NRGBA
	}{
		{"title", &b.Title, &b.Title.Image},
		{"font", &b.Font, &b.Font.Image},
		{"ships", &b.Ships, &b.Ships.Atlas.Image},
		{"player-terrain-stencil", &b.Stencil, &b.Stencil.Image},
		{"common-actors", &b.Common, &b.Common.Image},
		{"shop-art", &b.ShopArt, &b.ShopArt.Atlas.Image},
	}
	for _, resource := range common {
		if err := readJSON(resources, resource.name+".json", resource.metadata); err != nil {
			return nil, err
		}
		picture, err := readPNG(resources, resource.name+".png")
		if err != nil {
			return nil, err
		}
		*resource.picture = picture
	}
	if err := readJSON(resources, "shop.json", &b.Shop); err != nil {
		return nil, err
	}
	if err := readJSON(resources, "presentation.json", &b.Presentation); err != nil {
		return nil, err
	}
	font, fontErr := readPNG(resources, "presentation-font.png")
	if fontErr != nil {
		return nil, fontErr
	}
	b.Presentation.Font.Image = font
	zoom, zoomErr := readPNG(resources, "title-zoom.png")
	if zoomErr != nil {
		return nil, zoomErr
	}
	b.Presentation.LogoZoom.Image = zoom
	textZoom, textErr := readPNG(resources, "caption-zoom.png")
	if textErr != nil {
		return nil, textErr
	}
	b.Presentation.TextZoom.Image = textZoom
	if err := readJSON(resources, "player-presentation.json", &b.PlayerPresentation); err != nil {
		return nil, err
	}
	playerArt, playerErr := readPNG(resources, "player-presentation.png")
	if playerErr != nil {
		return nil, playerErr
	}
	b.PlayerPresentation.Atlas.Image = playerArt
	for _, resource := range []struct {
		name  string
		image **image.NRGBA
	}{{"hud-base", &b.PlayerPresentation.HUD}, {"hud-score-font", &b.PlayerPresentation.ScoreFont.Image}, {"hud-lives-font", &b.PlayerPresentation.LivesFont.Image}} {
		picture, err := readPNG(resources, resource.name+".png")
		if err != nil {
			return nil, err
		}
		*resource.image = picture
	}
	if err := readJSON(resources, "shop-scene.json", &b.ShopScene); err != nil {
		return nil, err
	}
	for _, resource := range []struct {
		name    string
		picture **image.NRGBA
	}{{"shop-base", &b.ShopScene.Base}, {"shop-portraits", &b.ShopScene.Portraits}, {"shop-controls", &b.ShopScene.ControlArt.Image}, {"shop-font", &b.ShopScene.Font.Image}, {"shop-cash-font", &b.ShopScene.CashFont.Image}, {"shop-transition", &b.ShopScene.TransitionArt.Image}} {
		picture, err := readPNG(resources, resource.name+".png")
		if err != nil {
			return nil, err
		}
		*resource.picture = picture
	}
	for index := range b.Levels {
		l := &LevelAssets{}
		prefix := fmt.Sprintf("level-%d", index+1)
		metadata := []struct {
			suffix string
			value  any
		}{{"", &l.Terrain}, {"-paths", &l.Paths}, {"-encounters", &l.Encounters}, {"-actors", &l.Actors}, {"-fixed-sprites", &l.FixedSprites}, {"-fixed-tiles", &l.FixedTiles}, {"-rules", &l.Rules}}
		for _, resource := range metadata {
			if err := readJSON(resources, prefix+resource.suffix+".json", resource.value); err != nil {
				return nil, err
			}
		}
		for _, resource := range []struct {
			suffix  string
			picture **image.NRGBA
		}{{"-tiles", &l.Terrain.Atlas}, {"-background", &l.Terrain.Background}, {"-actors", &l.Actors.Atlas.Image}, {"-fixed-sprites", &l.FixedSprites.Atlas.Image}, {"-shots", &l.Rules.EnemyShots.Image}} {
			picture, err := readPNG(resources, prefix+resource.suffix+".png")
			if err != nil {
				return nil, err
			}
			*resource.picture = picture
		}
		if err := validateLevel(l); err != nil {
			return nil, fmt.Errorf("level %d: %w", index+1, err)
		}
		b.Levels[index] = l
		guardianName := prefix + "-guardians.json"
		if _, err := fs.Stat(resources, guardianName); err == nil {
			l.Guardians = &visualassets.Guardians{}
			if err = readJSON(resources, guardianName, l.Guardians); err != nil {
				return nil, err
			}
			picture, err := readPNG(resources, prefix+"-guardians.png")
			if err != nil {
				return nil, err
			}
			l.Guardians.Atlas.Image = picture
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		groupName := prefix + "-guardian-groups.json"
		if _, err := fs.Stat(resources, groupName); err == nil {
			var decoded struct {
				Groups []visualassets.GuardianGroup `json:"groups"`
				Atlas  visualassets.SpriteAtlas     `json:"atlas"`
			}
			if err = readJSON(resources, groupName, &decoded); err != nil {
				return nil, err
			}
			picture, err := readPNG(resources, prefix+"-guardian-parts.png")
			if err != nil {
				return nil, err
			}
			decoded.Atlas.Image = picture
			l.GuardianGroups, l.GuardianParts = decoded.Groups, &decoded.Atlas
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var err error
	b.AudioBank, b.Waveforms, err = audio.LoadFS(resources, "audio")
	if err != nil {
		return nil, fmt.Errorf("audio resources: %w", err)
	}
	shopBank, shopWaves, err := audio.LoadFS(resources, "shop-audio")
	if err != nil {
		return nil, fmt.Errorf("shop audio resources: %w", err)
	}
	b.AudioBank.Samples = append(b.AudioBank.Samples, shopBank.Samples...)
	b.AudioBank.Effects = append(b.AudioBank.Effects, shopBank.Effects...)
	for id, pcm := range shopWaves {
		if _, exists := b.Waveforms[id]; exists {
			return nil, fmt.Errorf("duplicate shop sound %q", id)
		}
		b.Waveforms[id] = pcm
	}
	if err = b.Validate(); err != nil {
		return nil, err
	}
	return b, nil
}

func readJSON(resources fs.FS, name string, destination any) error {
	data, err := fs.ReadFile(resources, name)
	if err != nil {
		return fmt.Errorf("exported resource %s unavailable; run scripts/prepare-assets.sh: %w", name, err)
	}
	if err = json.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("resource %s: %w", name, err)
	}
	return nil
}

func readPNG(resources fs.FS, name string) (*image.NRGBA, error) {
	file, err := resources.Open(name)
	if err != nil {
		return nil, fmt.Errorf("exported image %s unavailable: %w", name, err)
	}
	defer file.Close()
	picture, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("image %s: %w", name, err)
	}
	bounds := picture.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 || bounds.Dx() > 8192 || bounds.Dy() > 8192 {
		return nil, fmt.Errorf("invalid exported image dimensions")
	}
	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(out, out.Bounds(), picture, bounds.Min, draw.Src)
	return out, nil
}

func (b *Bundle) Validate() error {
	if b == nil || b.Title.Image == nil || b.Font.Image == nil || b.Font.Width < 1 || b.Font.Height < 1 || b.Font.Columns < 1 || len(b.Font.Characters) == 0 {
		return fmt.Errorf("incomplete presentation resources")
	}
	if b.Title.Image.Bounds().Dx() != b.Title.Width || b.Title.Image.Bounds().Dy() != b.Title.Height {
		return fmt.Errorf("title dimensions do not match exported artwork")
	}
	for _, atlas := range []*visualassets.SpriteAtlas{&b.Ships.Atlas, &b.Common, &b.ShopArt.Atlas} {
		if err := validateAtlas(atlas); err != nil {
			return err
		}
	}
	if len(b.Ships.SteeringFrames) != 13 || len(b.Shop.Items) != 25 {
		return fmt.Errorf("incomplete ship or shop resource catalogue")
	}
	if b.Stencil.Width < 1 || b.Stencil.Width > 32 || b.Stencil.Height < 1 || len(b.Stencil.Rows) != b.Stencil.Height {
		return fmt.Errorf("invalid player terrain stencil")
	}
	if b.ShopScene.Base == nil || b.ShopScene.Portraits == nil || b.ShopScene.Font.Image == nil || b.ShopScene.CashFont.Image == nil || len(b.ShopScene.Cells) != 20 || b.ShopScene.PortraitFrames != 4 || b.ShopScene.MoneyDigits != 7 {
		return fmt.Errorf("shop composition incomplete")
	}
	if err := validateAtlas(&b.ShopScene.ControlArt); err != nil {
		return err
	}
	for _, l := range b.Levels {
		if l != nil && l.GuardianParts != nil {
			if err := validateAtlas(l.GuardianParts); err != nil {
				return err
			}
		}
		if err := validateLevel(l); err != nil {
			return err
		}
	}
	if b.AudioBank == nil {
		return fmt.Errorf("audio resources missing")
	}
	return b.AudioBank.Validate(b.Waveforms)
}

func validateLevel(l *LevelAssets) error {
	if l == nil {
		return fmt.Errorf("level resources missing")
	}
	t := &l.Terrain
	if t.Columns != 20 || t.Rows != 300 || t.TileSize != 16 || len(t.Map) != t.Columns*t.Rows || t.Atlas == nil || t.Background == nil || t.Background.Bounds().Dx() != 320 || t.Background.Bounds().Dy() != 192 {
		return fmt.Errorf("invalid level terrain dimensions")
	}
	ids := make(map[uint16]bool, len(t.Tiles))
	colors := make(map[[4]uint8]bool, 16)
	for _, color := range t.Palette {
		if colors[color] {
			return fmt.Errorf("palette effects require unambiguous color indices")
		}
		colors[color] = true
	}
	for _, tile := range t.Tiles {
		if tile.ID == 0 || ids[tile.ID] || tile.X < 0 || tile.Y < 0 || tile.X+16 > t.Atlas.Bounds().Dx() || tile.Y+16 > t.Atlas.Bounds().Dy() {
			return fmt.Errorf("invalid terrain tile")
		}
		ids[tile.ID] = true
	}
	for _, id := range t.Map {
		if id != 0 && !ids[id] {
			return fmt.Errorf("terrain refers to unknown tile")
		}
	}
	for _, atlas := range []*visualassets.SpriteAtlas{&l.Actors.Atlas, &l.FixedSprites.Atlas, &l.Rules.EnemyShots} {
		if err := validateAtlas(atlas); err != nil {
			return err
		}
	}
	return nil
}

func validateAtlas(atlas *visualassets.SpriteAtlas) error {
	if atlas == nil || atlas.Image == nil {
		return fmt.Errorf("sprite atlas image missing")
	}
	names := make(map[string]bool, len(atlas.Sprites))
	for _, sprite := range atlas.Sprites {
		if sprite.Name == "" || names[sprite.Name] || sprite.Width < 1 || sprite.Height < 1 || sprite.X < 0 || sprite.Y < 0 || sprite.X+sprite.Width > atlas.Image.Bounds().Dx() || sprite.Y+sprite.Height > atlas.Image.Bounds().Dy() {
			return fmt.Errorf("invalid sprite atlas region %q", sprite.Name)
		}
		names[sprite.Name] = true
	}
	return nil
}
