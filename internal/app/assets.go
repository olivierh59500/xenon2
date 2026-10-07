// Package app presents the verified Go simulation through Ebitengine.
package app

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"io/fs"

	runtimeassets "xenon2/assets/runtime"
	"xenon2/internal/audio"
	"xenon2/internal/visualassets"
)

// LevelAssets holds exported graphics and ordinary gameplay descriptors.
type LevelAssets struct {
	Terrain      visualassets.Terrain
	Paths        visualassets.Paths
	Encounters   visualassets.Encounters
	Actors       visualassets.Actors
	FixedSprites visualassets.FixedSprites
	FixedTiles   visualassets.FixedTiles
	Rules        visualassets.LevelRules
}

// Bundle retains no packed disk resources, original program or machine state.
type Bundle struct {
	Levels    [5]*LevelAssets
	Title     visualassets.TitleArt
	Font      visualassets.Font
	Ships     visualassets.ShipArt
	Stencil   visualassets.PlayerTerrainStencil
	Common    visualassets.SpriteAtlas
	Shop      visualassets.ShopCatalogue
	ShopArt   visualassets.ShopArt
	AudioBank *audio.Bank
	Waveforms map[string][]byte
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
	}
	var err error
	b.AudioBank, b.Waveforms, err = audio.LoadFS(resources, "audio")
	if err != nil {
		return nil, fmt.Errorf("audio resources: %w", err)
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
	for _, l := range b.Levels {
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
