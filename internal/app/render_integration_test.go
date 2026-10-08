package app

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"xenon2/internal/engine"
	"xenon2/internal/visualassets"
)

func renderIntegrationGame(t *testing.T) (*Game, *Bundle) {
	t.Helper()
	if logicOnlyTests {
		t.Skip("graphics fixture requires the normal graphics test runner")
	}
	root := os.Getenv("XENON2_RUNTIME_TEST_DIR")
	if root == "" {
		t.Skip("local exported resources not supplied")
	}
	bundle, err := LoadFS(os.DirFS(root))
	if err != nil {
		t.Fatal(err)
	}
	game, err := New(bundle)
	if err != nil {
		t.Fatal(err)
	}
	game.Screen = LevelScreen
	game.paused, game.pauseFraction = true, 1
	return game, bundle
}

func renderIntegrationPixels(t *testing.T, g *Game, name string) *image.NRGBA {
	t.Helper()
	if logicOnlyTests {
		// Frontend boundary tests use this helper for optional captures and
		// continue with logic assertions. Their assertions must still run.
		return nil
	}
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	defer screen.Dispose()
	g.Draw(screen)
	pixels := image.NewNRGBA(image.Rect(0, 0, ScreenWidth, ScreenHeight))
	screen.ReadPixels(pixels.Pix)
	if root := os.Getenv("XENON2_RENDER_CAPTURE_DIR"); root != "" {
		if err := os.MkdirAll(root, 0755); err != nil {
			t.Fatal(err)
		}
		if err := saveScreenshot(screen, filepath.Join(root, name+".png")); err != nil {
			t.Fatal(err)
		}
	}
	return pixels
}

func renderFixtureWorld(t *testing.T, bundle *Bundle, level, kind, scroll int) *worldDriver {
	t.Helper()
	l := bundle.Levels[level-1]
	encounters := &visualassets.Encounters{}
	if kind >= 0 {
		for _, record := range l.Encounters.Fixed {
			if record.EnemyKind == kind && (scroll < 0 || record.TriggerY == scroll) {
				encounters.Fixed = append(encounters.Fixed, record)
				if scroll < 0 {
					scroll = record.TriggerY
				}
				break
			}
		}
		if len(encounters.Fixed) != 1 {
			t.Fatalf("missing level%d fixture selector%d at%d", level, kind, scroll)
		}
	}
	data := engine.LevelData{Number: level, Terrain: &l.Terrain, Paths: &l.Paths, Encounters: encounters, Actors: &l.Actors, FixedSprites: &l.FixedSprites, FixedTiles: &l.FixedTiles, PlayerStencil: &bundle.Stencil, Rules: &l.Rules, Ships: &bundle.Ships, Common: &bundle.Common, Guardians: l.Guardians, GuardianGroups: l.GuardianGroups, GuardianParts: l.GuardianParts}
	w, err := engine.NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	// These fixtures isolate artwork and composition. They do not represent a
	// complete playthrough or assert terrain crushing/weapon gameplay.
	w.Coverage = nil
	w.ScrollY, w.PreviousScrollY, w.RenderScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = scroll, scroll, scroll, 0, 4607, 4607
	w.Ready = false
	w.InvulnerableFrames = 30000
	w.MaterializationFrames = 0
	if level == 1 && scroll < 2496 && w.FirstMiddle != nil {
		// A final-arena fixture has already crossed the source middle-shop gate.
		w.FirstMiddle.Crossed = true
	}
	return &worldDriver{world: w}
}

func assertRenderedActorsChangeBackdrop(t *testing.T, g *Game, name string) {
	t.Helper()
	frame := g.View
	g.View.Sprites = nil
	g.View.PlayerAlive = false
	g.View.PlayerSprite = ""
	backdrop := renderIntegrationPixels(t, g, name+"-backdrop")
	g.View = frame
	complete := renderIntegrationPixels(t, g, name)
	changed := 0
	for y := 0; y < PlayfieldHeight; y++ {
		for x := 0; x < ScreenWidth; x++ {
			if complete.NRGBAAt(x, y) != backdrop.NRGBAAt(x, y) {
				changed++
			}
		}
	}
	if changed < 24 {
		t.Fatalf("%s rendered only%d actor pixels", name, changed)
	}
	t.Logf("%s: %d actor pixels differ from the same production backdrop", name, changed)
}

func TestIntegratedGuardianAndBeamGPUFixturesOptional(t *testing.T) {
	g, b := renderIntegrationGame(t)
	for _, fixture := range []struct {
		name                        string
		level, kind, scroll, passes int
	}{
		{"guardian-1-middle-defense-streams", 1, -1, 3000, 30},
		{"guardian-1-body-eye-chain", 1, -1, 0, 60},
		{"guardian-2-middle-defense-streams", 2, -1, 2600, 30},
		{"guardian-2-body-hatch", 2, -1, 80, 3},
		{"guardian-3-articulated-middle", 3, 3, 2816, 80},
		{"guardian-3-final-worm", 3, -1, 160, 60},
		{"guardian-4-articulated-middle", 4, 2, 2672, 80},
		{"guardian-4-tiled-eyes-arms", 4, 4, 144, 180},
		{"level-4-extending-beam", 4, 5, 2096, 80},
		{"guardian-5-tiled-middle", 5, 5, 2448, 240},
		{"guardian-5-body-mouth", 5, 6, 0, 160},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			d := renderFixtureWorld(t, b, fixture.level, fixture.kind, fixture.scroll)
			if fixture.level == 2 {
				d.world.Player.Y = 80
			}
			if fixture.level == 5 && fixture.kind == 5 {
				// Keep the ship outside the core: invulnerable player contact still
				// damages the guardian through its original collision callback.
				d.world.Player.X = 24
			}
			for pass := 0; pass < fixture.passes; pass++ {
				if err := d.world.Step(engine.Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if fixture.level == 2 && d.world.SecondGuardian != nil {
				d.world.SecondGuardian.HatchCountdown = 2
				if err := d.world.Step(engine.Input{}); err != nil {
					t.Fatal(err)
				}
			}
			g.View = d.Frame()
			if fixture.level == 5 && fixture.kind == 5 {
				visibleBody := false
				for _, sprite := range g.View.Sprites {
					visibleBody = visibleBody || sprite.Kind == "tiles" && sprite.Y >= 0 && sprite.Y < PlayfieldHeight
				}
				if d.world.FifthMiddle.Defeated || !visibleBody {
					t.Fatal("middle fixture omitted its living tiled body")
				}
			}
			g.rememberFrameHistory()
			if len(g.View.Sprites) == 0 {
				t.Fatal("source world produced no visible fixture actors")
			}
			if fixture.level == 4 && fixture.kind == 4 {
				visible := 0
				for _, sprite := range g.View.Sprites {
					if sprite.Kind == "tiles" && sprite.Y >= -16 && sprite.Y+float64(sprite.Patch.Rows*16) <= PlayfieldHeight {
						visible++
					}
				}
				if visible < 3 {
					t.Fatalf("final fixture shows only%d tiled body/eye parts", visible)
				}
			}
			assertRenderedActorsChangeBackdrop(t, g, fixture.name)
		})
	}
}

func TestOriginalBackgroundStarAlignmentGPUPixelsOptional(t *testing.T) {
	g, b := renderIntegrationGame(t)
	d := renderFixtureWorld(t, b, 1, -1, 4608)
	d.world.ResetBackgroundStars()
	if err := d.world.Step(engine.Input{}); err != nil {
		t.Fatal(err)
	}
	g.View = d.Frame()
	stars := g.View.BackgroundStars
	if stars == nil {
		t.Fatal("source48-star bands missing")
	}
	g.View.BackgroundStars = nil
	g.View.PlayerAlive = false
	g.View.PlayerSprite = ""
	g.View.Sprites = nil
	g.View.TerrainMap = make([]uint16, 6000)
	g.rememberFrameHistory()
	base := renderIntegrationPixels(t, g, "background-star-source-backdrop")
	g.View.BackgroundStars = stars
	actual := renderIntegrationPixels(t, g, "background-star-source-bands")
	points := map[image.Point]uint8{}
	for _, star := range stars.Stars {
		p := image.Pt(star.X, star.Y)
		if _, exists := points[p]; !exists {
			points[p] = star.Color
		}
	}
	visible, blocked := 0, 0
	for y := 0; y < PlayfieldHeight; y++ {
		for x := 0; x < ScreenWidth; x++ {
			old := base.NRGBAAt(x, y)
			want := old
			if index, exists := points[image.Pt(x, y)]; exists {
				if old.R == 0 && old.G == 0 && old.B == 0 {
					c := b.Levels[0].Terrain.Palette[index]
					want = color.NRGBA{R: c[0], G: c[1], B: c[2], A: c[3]}
					visible++
				} else {
					blocked++
				}
			}
			if actual.NRGBAAt(x, y) != want {
				t.Fatalf("original star alignment(%d,%d): got%v want%v", x, y, actual.NRGBAAt(x, y), want)
			}
		}
	}
	if visible == 0 || blocked == 0 {
		t.Fatalf("source star fixture lacks black/nonblack backdrop cases: %d/%d", visible, blocked)
	}
	t.Logf("Checked all48 source stars: %d black-background points and%d blocked points", visible, blocked)
}

func integrationSpritePixels(b *Bundle, s SpriteView, level int) (*image.NRGBA, visualassets.SpriteRegion) {
	bank := b.Ships.Atlas
	if s.Atlas == "common" {
		bank = b.Common
	}
	for _, region := range bank.Sprites {
		if region.Name == s.Sprite {
			return bank.Image, region
		}
	}
	return nil, visualassets.SpriteRegion{}
}

func TestMaterializationUsesActorMapAndCameraGPUPixelsOptional(t *testing.T) {
	g, b := renderIntegrationGame(t)
	d := renderFixtureWorld(t, b, 1, -1, 0)
	g.View = d.Frame()
	g.View.PlayerAlive = false
	g.View.PlayerSprite = ""
	g.View.Sprites = nil
	g.View.BackgroundStars = nil
	g.View.CameraY, g.View.ActorCameraY = 0, 16
	g.View.TerrainMap = make([]uint16, 6000)
	g.View.ActorTerrainMap = make([]uint16, 6000)
	var opaque uint16
	for _, tile := range b.Levels[0].Terrain.Tiles {
		if tile.ID != 0 && !tile.Masked {
			opaque = tile.ID
			break
		}
	}
	if opaque == 0 {
		t.Fatal("original opaque tile not found")
	}
	g.View.ActorTerrainMap[5*20+10] = opaque
	g.rememberFrameHistory()
	backdrop := renderIntegrationPixels(t, g, "materialization-backdrop")
	sprite := SpriteView{Atlas: "ships", Sprite: b.Ships.SteeringFrames[6], Layer: "moving", Materializing: true}
	source, region := integrationSpritePixels(b, sprite, 1)
	if source == nil {
		t.Fatal("original ship sprite missing")
	}
	sprite.X, sprite.Y = float64(152+region.AnchorX), float64(56+region.AnchorY)
	g.View.Sprites = []SpriteView{sprite}
	actual := renderIntegrationPixels(t, g, "materialization-live-mask")
	maskPixels := make([]byte, ScreenWidth*PlayfieldHeight*4)
	g.graphics.terrainMask.ReadPixels(maskPixels)
	covered, visible := 0, 0
	for y := 0; y < region.Height; y++ {
		for x := 0; x < region.Width; x++ {
			original := source.NRGBAAt(region.X+x, region.Y+y)
			if original.A == 0 {
				continue
			}
			px, py := 152+x, 56+y
			if py >= PlayfieldHeight || px >= ScreenWidth {
				continue
			}
			want := original
			if px >= 160 && px < 176 && py >= 64 && py < 80 {
				want = backdrop.NRGBAAt(px, py)
				covered++
			} else {
				visible++
			}
			if actual.NRGBAAt(px, py) != want {
				t.Fatalf("live-mask pixel(%d,%d): got%v want%v maskAlpha%d", px, py, actual.NRGBAAt(px, py), want, maskPixels[(py*ScreenWidth+px)*4+3])
			}
		}
	}
	if covered < 8 || visible < 8 {
		t.Fatalf("mask fixture lacks both source coverage classes: %d/%d", covered, visible)
	}
}

func TestOriginalAuraArtworkGPUPixelsOptional(t *testing.T) {
	g, b := renderIntegrationGame(t)
	d := renderFixtureWorld(t, b, 1, -1, 4608)
	if err := d.world.Step(engine.Input{}); err != nil {
		t.Fatal(err)
	}
	g.View = d.Frame()
	g.rememberFrameHistory()
	var name string
	for _, clip := range b.Common.Animations {
		if clip.ID == "player-invulnerability" && len(clip.Animation.Frames) > 0 {
			name = clip.Animation.Frames[0].Sprite
			break
		}
	}
	if name == "" {
		t.Fatal("original aura animation missing")
	}
	sprite := SpriteView{Atlas: "common", Sprite: name, Layer: "effects", X: float64(g.View.Player.X), Y: float64(g.View.Player.Y)}
	source, region := integrationSpritePixels(b, sprite, 1)
	if source == nil {
		t.Fatal("original aura image missing")
	}
	backdrop := renderIntegrationPixels(t, g, "aura-source-backdrop")
	g.View.Sprites = append(g.View.Sprites, sprite)
	actual := renderIntegrationPixels(t, g, "aura-source-artwork")
	visible, transparent := 0, 0
	for y := 0; y < region.Height; y++ {
		for x := 0; x < region.Width; x++ {
			px, py := int(sprite.X)-region.AnchorX+x, int(sprite.Y)-region.AnchorY+y
			if px < 0 || px >= ScreenWidth || py < 0 || py >= PlayfieldHeight {
				continue
			}
			original := source.NRGBAAt(region.X+x, region.Y+y)
			want := original
			if original.A == 0 {
				want = backdrop.NRGBAAt(px, py)
				transparent++
			} else {
				visible++
			}
			if actual.NRGBAAt(px, py) != want {
				t.Fatalf("aura pixel(%d,%d): got%v want%v", px, py, actual.NRGBAAt(px, py), want)
			}
		}
	}
	if visible < 8 || transparent < 8 {
		t.Fatalf("aura fixture lacks both source alpha classes: %d/%d", visible, transparent)
	}
}

func TestLevelPaletteStrobeAndShadesGPUPixelsOptional(t *testing.T) {
	g, b := renderIntegrationGame(t)
	for level := 1; level <= 5; level++ {
		d := renderFixtureWorld(t, b, level, -1, 4608)
		if err := d.world.Step(engine.Input{}); err != nil {
			t.Fatal(err)
		}
		g.View = d.Frame()
		g.rememberFrameHistory()
		base := renderIntegrationPixels(t, g, fmt.Sprintf("palette-%d-source", level))
		palette := b.Levels[level-1].Terrain.Palette
		indices := map[color.NRGBA]int{}
		for index, c := range palette {
			indices[color.NRGBA{R: c[0], G: c[1], B: c[2], A: 255}] = index
		}
		for _, shades := range []bool{false, true} {
			g.View.FreezeInterpolation, g.View.Shades = !shades, shades
			g.View.PaletteMask = 0xa55a
			name := fmt.Sprintf("palette-%d-strobe", level)
			if shades {
				name = fmt.Sprintf("palette-%d-shades", level)
			}
			actual := renderIntegrationPixels(t, g, name)
			for y := 0; y < PlayfieldHeight; y++ {
				for x := 0; x < ScreenWidth; x++ {
					old := base.NRGBAAt(x, y)
					index, ok := indices[old]
					if !ok {
						t.Fatalf("level%d source pixel(%d,%d) is outside its original palette: %v", level, x, y, old)
					}
					want := color.NRGBA{A: 255}
					if shades {
						want.R = old.R / 68 * 34
						want.G = old.G / 68 * 34
						want.B = old.B / 68 * 34
					} else if (g.View.PaletteMask>>index)&1 != 0 {
						want.R, want.G, want.B = 238, 238, 238
					}
					if actual.NRGBAAt(x, y) != want {
						t.Fatalf("%s pixel(%d,%d): got%v want%v", name, x, y, actual.NRGBAAt(x, y), want)
					}
				}
			}
		}
	}
}

func TestMaskedTileBodyPlacementAndFlashGPUPixelsOptional(t *testing.T) {
	g, b := renderIntegrationGame(t)
	d := renderFixtureWorld(t, b, 1, -1, 4608)
	g.View = d.Frame()
	g.rememberFrameHistory()
	var tile visualassets.Tile
	found := false
	for _, candidate := range b.Levels[0].Terrain.Tiles {
		if !candidate.Masked {
			continue
		}
		covered, transparent := 0, 0
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				if b.Levels[0].Terrain.Atlas.NRGBAAt(candidate.X+x, candidate.Y+y).A == 0 {
					transparent++
				} else {
					covered++
				}
			}
		}
		if covered > 8 && transparent > 8 {
			tile, found = candidate, true
			break
		}
	}
	if !found {
		t.Fatal("original masked body tile not found")
	}
	backdrop := renderIntegrationPixels(t, g, "tile-body-backdrop")
	for _, flash := range []bool{false, true} {
		g.View.Sprites = []SpriteView{{Kind: "tiles", Layer: "moving", X: 48, Y: 48, Flash: flash, Patch: visualassets.TilePatch{Columns: 2, Rows: 2, Tiles: []uint16{tile.ID, tile.ID, tile.ID, tile.ID}}}}
		name := "masked-tile-body"
		if flash {
			name = "masked-tile-body-flash"
		}
		actual := renderIntegrationPixels(t, g, name)
		c := b.Levels[0].Terrain.Palette[15]
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				original := b.Levels[0].Terrain.Atlas.NRGBAAt(tile.X+x%16, tile.Y+y%16)
				want := original
				if original.A == 0 {
					want = backdrop.NRGBAAt(48+x, 48+y)
				} else if flash {
					want = color.NRGBA{R: c[0], G: c[1], B: c[2], A: original.A}
				}
				if actual.NRGBAAt(48+x, 48+y) != want {
					t.Fatalf("%s pixel(%d,%d): got%v want%v", name, 48+x, 48+y, actual.NRGBAAt(48+x, 48+y), want)
				}
			}
		}
	}
}

func TestSourceMovingBeforeEffectsPainterOrderGPUPixelsOptional(t *testing.T) {
	g, b := renderIntegrationGame(t)
	d := renderFixtureWorld(t, b, 1, -1, 4608)
	g.View = d.Frame()
	g.rememberFrameHistory()
	a := SpriteView{Atlas: "ships", Sprite: b.Ships.SteeringFrames[6], Layer: "moving", X: 150, Y: 100}
	bb := SpriteView{Atlas: "ships", Sprite: b.Ships.SteeringFrames[12], Layer: "effects", X: 152, Y: 100, Flash: true}
	sourceA, regionA := integrationSpritePixels(b, a, 1)
	sourceB, regionB := integrationSpritePixels(b, bb, 1)
	if sourceA == nil || sourceB == nil {
		t.Fatal("original overlap sprites missing")
	}
	// Deliberately reverse slice order. Source layer traversal still draws
	// moving actors before projectile/effect actors.
	g.View.Sprites = []SpriteView{bb, a}
	actual := renderIntegrationPixels(t, g, "moving-before-effects-overlap")
	c := b.Levels[0].Terrain.Palette[15]
	want := color.NRGBA{R: c[0], G: c[1], B: c[2], A: 255}
	overlaps := 0
	for y := 0; y < regionB.Height; y++ {
		for x := 0; x < regionB.Width; x++ {
			if sourceB.NRGBAAt(regionB.X+x, regionB.Y+y).A == 0 {
				continue
			}
			px, py := int(bb.X)-regionB.AnchorX+x, int(bb.Y)-regionB.AnchorY+y
			ax, ay := px-(int(a.X)-regionA.AnchorX), py-(int(a.Y)-regionA.AnchorY)
			if ax < 0 || ax >= regionA.Width || ay < 0 || ay >= regionA.Height || sourceA.NRGBAAt(regionA.X+ax, regionA.Y+ay).A == 0 {
				continue
			}
			if actual.NRGBAAt(px, py) != want {
				t.Fatalf("later source effect lost overlap(%d,%d): got%v want%v", px, py, actual.NRGBAAt(px, py), want)
			}
			overlaps++
		}
	}
	if overlaps < 8 {
		t.Fatalf("source painter fixture has only%d overlapping pixels", overlaps)
	}
}
