package app

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"xenon2/internal/shopui"
	"xenon2/internal/visualassets"
)

type spriteGraphic struct {
	flash                           *ebiten.Image
	image                           *ebiten.Image
	anchorX, anchorY, width, height int
}
type atlasGraphics map[string]spriteGraphic
type levelGraphics struct {
	sparkUniforms                       map[string]any
	flashTiles                          map[uint16]*ebiten.Image
	guardians                           atlasGraphics
	guardianParts                       atlasGraphics
	dive                                atlasGraphics
	uniforms                            map[string]any
	paletteMask, shades                 []float32
	background                          *ebiten.Image
	tiles                               map[uint16]*ebiten.Image
	moving, fixed, ships, common, shots atlasGraphics
}
type graphics struct {
	fadeShader                                   *ebiten.Shader
	fadeScene                                    *ebiten.Image
	fadeAmount                                   []float32
	fadeUniforms                                 map[string]any
	materialShader                               *ebiten.Shader
	terrainMask                                  *ebiten.Image
	terrainMaskTiles                             [5]map[uint16]*ebiten.Image
	materialUniforms                             map[string]any
	materialPosition                             []float32
	backgroundStarShader                         *ebiten.Shader
	backgroundStarBackdrop, backgroundStarPoints *ebiten.Image
	backgroundStarPixels                         []byte
	creditOverlaps                               map[string]*ebiten.Image
	creditOverlapPixels                          map[string]*image.NRGBA
	sparkShader                                  *ebiten.Shader
	sparkScratch                                 *ebiten.Image
	transitionStrips                             map[string]*ebiten.Image
	shopTransition                               atlasGraphics
	textZoomRegions                              map[string]visualassets.SpriteRegion
	hudBase                                      *ebiten.Image
	hudScore, hudLives                           map[rune]*ebiten.Image
	logoZoom, textZoom                           atlasGraphics
	presentationFont                             map[rune]*ebiten.Image
	shopNoise                                    [20]*ebiten.Image
	shopBase                                     *ebiten.Image
	portraits                                    []*ebiten.Image
	smallFont, cashFont                          map[rune]*ebiten.Image
	shopControls                                 atlasGraphics
	paletteShader                                *ebiten.Shader
	title                                        *ebiten.Image
	font                                         map[rune]*ebiten.Image
	ships, common, shop                          atlasGraphics
	levels                                       [5]levelGraphics
	playfield                                    *ebiten.Image
}

func prepareGraphics(b *Bundle) graphics {
	g := graphics{title: ebiten.NewImageFromImage(b.Title.Image), font: make(map[rune]*ebiten.Image), ships: prepareAtlas(b.Ships.Atlas), common: prepareAtlas(b.Common), shop: prepareAtlas(b.ShopArt.Atlas), playfield: ebiten.NewImage(ScreenWidth, ScreenHeight)}
	font := ebiten.NewImageFromImage(remapPalette(b.Font.Image, b.Levels[0].Terrain.Palette, b.Title.Palette))
	for index, char := range b.Font.Characters {
		x, y := (index%b.Font.Columns)*b.Font.Width, (index/b.Font.Columns)*b.Font.Height
		g.font[char] = font.SubImage(image.Rect(x, y, x+b.Font.Width, y+b.Font.Height)).(*ebiten.Image)
	}
	g.fadeScene = ebiten.NewImage(ScreenWidth, ScreenHeight)
	g.fadeAmount = make([]float32, 1)
	g.fadeUniforms = map[string]any{"Deduction": g.fadeAmount}
	g.terrainMask = ebiten.NewImage(ScreenWidth, PlayfieldHeight)
	g.materialPosition = make([]float32, 2)
	g.materialUniforms = map[string]any{"Position": g.materialPosition}
	g.backgroundStarBackdrop = ebiten.NewImage(ScreenWidth, PlayfieldHeight)
	g.backgroundStarPoints = ebiten.NewImage(ScreenWidth, PlayfieldHeight)
	g.backgroundStarPixels = make([]byte, ScreenWidth*PlayfieldHeight*4)
	g.shopBase = ebiten.NewImageFromImage(b.ShopScene.Base)
	portraitImage := ebiten.NewImageFromImage(b.ShopScene.Portraits)
	for i := 0; i < b.ShopScene.PortraitFrames; i++ {
		x := i * b.ShopScene.PortraitWidth
		g.portraits = append(g.portraits, portraitImage.SubImage(image.Rect(x, 0, x+b.ShopScene.PortraitWidth, b.ShopScene.PortraitHeight)).(*ebiten.Image))
	}
	g.smallFont = prepareFont(b.ShopScene.Font)
	g.cashFont = prepareFont(b.ShopScene.CashFont)
	g.shopControls = prepareAtlas(b.ShopScene.ControlArt)
	g.shopTransition = prepareAtlas(b.ShopScene.TransitionArt)
	g.transitionStrips = make(map[string]*ebiten.Image)
	for _, name := range []string{"shop-transition-strip-0", "shop-transition-strip-1", "shop-transition-strip-2"} {
		sprite := g.shopTransition[name]
		for source := 0; source < sprite.height; source += 4 {
			for height := 4; height <= sprite.height-source; height += 4 {
				key := fmt.Sprintf("%s:%d:%d", name, source, height)
				g.transitionStrips[key] = sprite.image.SubImage(image.Rect(7, source, 103, source+height)).(*ebiten.Image)
			}
		}
	}
	g.presentationFont = prepareFont(b.Presentation.Font)
	g.hudBase = ebiten.NewImageFromImage(b.PlayerPresentation.HUD)
	g.hudScore = prepareFont(b.PlayerPresentation.ScoreFont)
	g.hudLives = prepareFont(b.PlayerPresentation.LivesFont)
	g.logoZoom = prepareAtlas(b.Presentation.LogoZoom)
	g.textZoom = prepareAtlas(b.Presentation.TextZoom)
	g.creditOverlaps = make(map[string]*ebiten.Image, 6)
	g.creditOverlapPixels = make(map[string]*image.NRGBA, 6)
	for pair := 0; pair < 6; pair++ {
		text := b.Presentation.Credits[pair*2]
		pixels := visualassets.ComposeCreditOverlap(b.Presentation.Font, text, b.Title.Palette)
		g.creditOverlapPixels[text] = pixels
		g.creditOverlaps[text] = ebiten.NewImageFromImage(pixels)
	}
	g.textZoomRegions = make(map[string]visualassets.SpriteRegion, len(b.Presentation.TextZoom.Sprites))
	for _, region := range b.Presentation.TextZoom.Sprites {
		g.textZoomRegions[region.Name] = region
	}
	for i := range g.shopNoise {
		g.shopNoise[i] = ebiten.NewImage(32, 28)
	}
	for i, l := range b.Levels {
		atlas := ebiten.NewImageFromImage(l.Terrain.Atlas)
		v := levelGraphics{background: ebiten.NewImageFromImage(l.Terrain.Background), tiles: make(map[uint16]*ebiten.Image, len(l.Terrain.Tiles)), moving: prepareAtlas(l.Actors.Atlas), fixed: prepareAtlas(l.FixedSprites.Atlas)}
		v.shots = prepareAtlas(l.Rules.EnemyShots)
		if l.Guardians != nil {
			v.guardians = prepareAtlas(l.Guardians.Atlas)
		}
		if l.GuardianParts != nil {
			v.guardianParts = prepareAtlas(*l.GuardianParts)
		}
		palette := make([]float32, 64)
		for index, c := range l.Terrain.Palette {
			for component, value := range c {
				palette[index*4+component] = float32(value) / 255
			}
		}
		v.paletteMask = make([]float32, 1)
		v.shades = make([]float32, 1)
		v.uniforms = map[string]any{"Palette": palette, "Mask": v.paletteMask, "Shades": v.shades}
		v.sparkUniforms = map[string]any{"Palette": palette}
		ships, common := b.Ships.Atlas, b.Common
		ships.Image = remapPalette(ships.Image, b.Levels[0].Terrain.Palette, l.Terrain.Palette)
		common.Image = remapPalette(common.Image, b.Levels[0].Terrain.Palette, l.Terrain.Palette)
		v.ships, v.common = prepareAtlas(ships), prepareAtlas(common)
		dive := b.PlayerPresentation.Atlas
		dive.Image = remapPalette(dive.Image, b.Levels[0].Terrain.Palette, l.Terrain.Palette)
		v.dive = prepareAtlas(dive)
		for _, tile := range l.Terrain.Tiles {
			v.tiles[tile.ID] = atlas.SubImage(image.Rect(tile.X, tile.Y, tile.X+16, tile.Y+16)).(*ebiten.Image)
		}
		g.terrainMaskTiles[i] = make(map[uint16]*ebiten.Image, len(l.Terrain.Tiles))
		for _, tile := range l.Terrain.Tiles {
			pixels := flashPixels(l.Terrain.Atlas, image.Rect(tile.X, tile.Y, tile.X+16, tile.Y+16), [4]uint8{255, 255, 255, 255})
			g.terrainMaskTiles[i][tile.ID] = ebiten.NewImageFromImage(pixels)
		}
		v.flashTiles = make(map[uint16]*ebiten.Image, len(l.Terrain.Tiles))
		for _, tile := range l.Terrain.Tiles {
			v.flashTiles[tile.ID] = ebiten.NewImageFromImage(flashPixels(l.Terrain.Atlas, image.Rect(tile.X, tile.Y, tile.X+16, tile.Y+16), l.Terrain.Palette[15]))
		}
		for _, bank := range []struct {
			graphics atlasGraphics
			source   visualassets.SpriteAtlas
		}{{v.moving, l.Actors.Atlas}, {v.fixed, l.FixedSprites.Atlas}, {v.common, common}, {v.ships, ships}, {v.shots, l.Rules.EnemyShots}} {
			for _, sprite := range bank.source.Sprites {
				graphic := bank.graphics[sprite.Name]
				graphic.flash = ebiten.NewImageFromImage(flashPixels(bank.source.Image, image.Rect(sprite.X, sprite.Y, sprite.X+sprite.Width, sprite.Y+sprite.Height), l.Terrain.Palette[15]))
				bank.graphics[sprite.Name] = graphic
			}
		}
		if l.Guardians != nil {
			for _, sprite := range l.Guardians.Atlas.Sprites {
				graphic := v.guardians[sprite.Name]
				graphic.flash = ebiten.NewImageFromImage(flashPixels(l.Guardians.Atlas.Image, image.Rect(sprite.X, sprite.Y, sprite.X+sprite.Width, sprite.Y+sprite.Height), l.Terrain.Palette[15]))
				v.guardians[sprite.Name] = graphic
			}
		}
		if l.GuardianParts != nil {
			for _, sprite := range l.GuardianParts.Sprites {
				graphic := v.guardianParts[sprite.Name]
				graphic.flash = ebiten.NewImageFromImage(flashPixels(l.GuardianParts.Image, image.Rect(sprite.X, sprite.Y, sprite.X+sprite.Width, sprite.Y+sprite.Height), l.Terrain.Palette[15]))
				v.guardianParts[sprite.Name] = graphic
			}
		}
		g.levels[i] = v
	}
	return g
}

func prepareFont(font visualassets.Font) map[rune]*ebiten.Image {
	texture := ebiten.NewImageFromImage(font.Image)
	glyphs := make(map[rune]*ebiten.Image, len(font.Characters))
	for index, char := range font.Characters {
		x, y := (index%font.Columns)*font.Width, (index/font.Columns)*font.Height
		glyphs[char] = texture.SubImage(image.Rect(x, y, x+font.Width, y+font.Height)).(*ebiten.Image)
	}
	return glyphs
}

func prepareAtlas(a visualassets.SpriteAtlas) atlasGraphics {
	texture := ebiten.NewImageFromImage(a.Image)
	regions := make(atlasGraphics, len(a.Sprites))
	for _, s := range a.Sprites {
		regions[s.Name] = spriteGraphic{image: texture.SubImage(image.Rect(s.X, s.Y, s.X+s.Width, s.Y+s.Height)).(*ebiten.Image), anchorX: s.AnchorX, anchorY: s.AnchorY, width: s.Width, height: s.Height}
	}
	return regions
}

func (g *Game) drawScreen(screen *ebiten.Image) {
	screen.Fill(color.Black)
	switch g.Screen {
	case TitleScreen:
		g.drawTitle(screen)
	case ShopScreen:
		g.drawShop(screen)
	case LevelScreen:
		g.drawLevel(screen)
	case PresentationScreen:
		g.drawPresentation(screen)
	}
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	p := &g.Bundle.Presentation
	g.drawGlyphs(screen, g.graphics.presentationFont, p.MenuHeading, 0, 4, p.Font.Width)
	for i, line := range p.Menu {
		text := line.Text
		if g.menu == i {
			text = line.SelectedText
		}
		if line.ID == "music" {
			if g.music {
				text = p.MusicOn
				if g.menu == i {
					text = p.MusicOnSelected
				}
			} else {
				text = p.MusicOff
				if g.menu == i {
					text = p.MusicOffSelected
				}
			}
		}
		g.drawGlyphs(screen, g.graphics.presentationFont, text, 0, line.CenterY-8, p.Font.Width)
	}
	g.drawGlyphs(screen, g.graphics.font, g.Bundle.Presentation.CreditsCaption, 176, 184, 8)
	g.drawStarfield(screen)
	if g.Driver == nil || g.View.Diagnostic {
		ebitenutil.DebugPrintAt(screen, "REFERENCE BUILD - GAME RULES IN PROGRESS", 8, 184)
	}
}

func (g *Game) drawStarfield(screen *ebiten.Image) {
	if g.starfield == nil {
		return
	}
	for _, star := range g.starfield.Stars {
		if star.Depth&32768 != 0 {
			continue
		}
		c := g.Bundle.Title.Palette[star.Color]
		vector.DrawFilledRect(screen, float32(star.ScreenX), float32(star.ScreenY), 1, 1, color.NRGBA{R: c[0], G: c[1], B: c[2], A: c[3]}, false)
	}
}

func (g *Game) drawLevel(screen *ebiten.Image) {
	if g.View.Ready {
		g.drawOriginalMessage(screen, "GET READY PLAYER 1")
		return
	}
	if g.View.GameOver {
		g.drawOriginalMessage(screen, "GAME OVER")
		return
	}
	level := g.View.Level
	if level < 1 || level > 5 {
		return
	}
	l := g.Bundle.Levels[level-1]
	gpu := &g.graphics.levels[level-1]
	alpha := g.clock.Fraction()
	if g.View.FreezeInterpolation || g.backdropOnly {
		alpha = 1
	}
	camera := lerp(g.previous.CameraY, g.View.CameraY, alpha)
	if g.previous.Level != g.View.Level {
		camera = g.View.CameraY
	}
	field := g.graphics.playfield
	field.Fill(color.Black)
	backgroundY := wrapLerp(g.previous.BackgroundY, g.View.BackgroundY, alpha, PlayfieldHeight)
	if backgroundY < 0 {
		backgroundY += PlayfieldHeight
	}
	for _, y := range []float64{backgroundY, backgroundY - PlayfieldHeight} {
		op := ebiten.DrawImageOptions{}
		op.GeoM.Translate(g.View.BackgroundX, y)
		field.DrawImage(gpu.background, &op)
	}
	tiles := l.Terrain.Map
	if len(g.View.TerrainMap) == len(tiles) {
		tiles = g.View.TerrainMap
	}
	first := int(math.Floor(camera / 16))
	last := int(math.Ceil((camera + PlayfieldHeight) / 16))
	// The special source renderer clips against map coverage, independently of
	// already drawn actors. Build this mask in a separate pass to keep batching.
	needsMask := g.View.DivePhase == 4
	for _, sprite := range g.View.Sprites {
		needsMask = needsMask || sprite.Materializing
	}
	if needsMask {
		g.graphics.terrainMask.Clear()
		for row := max(0, first); row < min(l.Terrain.Rows, last); row++ {
			for column := 0; column < l.Terrain.Columns; column++ {
				tile := g.graphics.terrainMaskTiles[level-1][tiles[row*l.Terrain.Columns+column]]
				if tile == nil {
					continue
				}
				op := ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(column*16), float64(row*16)-camera)
				g.graphics.terrainMask.DrawImage(tile, &op)
			}
		}
	}
	for row := max(0, first); row < min(l.Terrain.Rows, last); row++ {
		for column := 0; column < l.Terrain.Columns; column++ {
			id := tiles[row*l.Terrain.Columns+column]
			tile := gpu.tiles[id]
			if tile == nil {
				continue
			}
			op := ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(column*16), float64(row*16)-camera)
			field.DrawImage(tile, &op)
		}
	}
	g.drawBackgroundStars(field, level, alpha)
	if !g.backdropOnly {
		g.drawPlayer(field, level, alpha)
		for _, source := range []string{"shadows", "equipment", "moving", "effects", "scenery", "sparks"} {
			for _, sprite := range g.View.Sprites {
				if sprite.Layer == source {
					g.drawSprite(field, sprite, level, alpha)
				}
			}
		}
	}
	for _, sprite := range g.View.HUD {
		g.drawSprite(field, sprite, level, alpha)
	}
	g.drawOriginalHUD(field)
	if g.View.FreezeInterpolation || g.View.Shades {
		gpu.paletteMask[0] = float32(g.View.PaletteMask)
		if !g.View.FreezeInterpolation {
			gpu.paletteMask[0] = -1
		}
		gpu.shades[0] = 0
		if g.View.Shades {
			gpu.shades[0] = 1
		}
		op := ebiten.DrawRectShaderOptions{Uniforms: gpu.uniforms}
		op.Images[0] = field
		screen.DrawRectShader(ScreenWidth, ScreenHeight, g.graphics.paletteShader, &op)
	} else {
		screen.DrawImage(field, nil)
	}
	if g.View.Diagnostic && g.Config.Frames > 0 {
		ebitenutil.DebugPrintAt(screen, "REFERENCE", 0, 181)
	}
}

func (g *Game) drawOriginalHUD(screen *ebiten.Image) {
	p := &g.Bundle.PlayerPresentation
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 192)
	screen.DrawImage(g.graphics.hudBase, &op)
	count := max(1, g.View.PlayerCount)
	for player := 0; player < count; player++ {
		score, lives, shield := g.View.PlayerScores[player], g.View.PlayerLives[player], g.View.PlayerShields[player]
		if g.View.PlayerCount == 0 {
			score, lives, shield = g.View.Score, g.View.Lives, g.View.Shield
		}
		xscore, xlives, xshield := p.ScoreX, p.LivesX, p.ShieldX
		if player == 1 {
			xscore, xlives, xshield = 240, 176, 190
		}
		g.drawGlyphs(screen, g.graphics.hudScore, fmt.Sprintf("%07d", score), xscore, p.ScoreY, 8)
		g.drawGlyphs(screen, g.graphics.hudLives, fmt.Sprint(min(9, max(0, lives))), xlives, p.LivesY, 8)
		for column := 0; column < p.ShieldColumns; column++ {
			c := color.NRGBA{A: 255}
			if column < min(p.ShieldColumns, max(0, shield)) {
				v := g.Bundle.Levels[g.View.Level-1].Terrain.Palette[14]
				c = color.NRGBA{R: v[0], G: v[1], B: v[2], A: 255}
			}
			vector.DrawFilledRect(screen, float32(xshield+column), float32(p.ShieldY), 1, float32(p.ShieldHeight), c, false)
		}
	}
}

func (g *Game) drawPlayer(field *ebiten.Image, level int, alpha float64) {
	x, y := lerp(float64(g.previous.Player.X), float64(g.View.Player.X), alpha), lerp(float64(g.previous.Player.Y), float64(g.View.Player.Y), alpha)
	if g.View.PlayerSprite != "" {
		g.drawSprite(field, SpriteView{Atlas: "common", Sprite: g.View.PlayerSprite, X: x, Y: y}, level, alpha)
		return
	}
	if !g.View.PlayerAlive {
		return
	}
	if g.View.DivePhase > 0 && g.View.DivePhase <= 4 {
		g.drawSprite(field, SpriteView{Materializing: g.View.DivePhase == 4, Atlas: "dive", Sprite: g.Bundle.PlayerPresentation.DiveFrames[g.View.DivePhase-1], X: x, Y: y}, level, alpha)
		return
	}
	index := max(0, min(12, g.View.Player.Inertia+6))
	g.drawSprite(field, SpriteView{Atlas: "ships", Sprite: g.Bundle.Ships.SteeringFrames[index], X: x, Y: y}, level, alpha)
}

func (g *Game) drawOriginalMessage(screen *ebiten.Image, text string) {
	screen.Fill(color.Black)
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(g.Bundle.Title.X), float64(g.Bundle.Title.Y))
	screen.DrawImage(g.graphics.title, &op)
	caption := g.Bundle.Presentation.Ready
	if text == "GAME OVER" {
		caption = g.Bundle.Presentation.GameOver
	}
	g.drawGlyphs(screen, g.graphics.presentationFont, caption, 0, 92, g.Bundle.Presentation.Font.Width)
}

func (g *Game) drawSprite(destination *ebiten.Image, sprite SpriteView, level int, alpha float64) {
	if sprite.Kind == "fifth-column" {
		g.drawFifthColumn(destination, sprite, level, alpha)
		return
	}
	if sprite.Kind == "laser" {
		g.drawLaser(destination, sprite, level, alpha)
		return
	}
	if sprite.Kind == "spark" {
		g.drawSpark(destination, sprite, level, alpha)
		return
	}
	if sprite.Kind == "tiles" {
		g.drawTileBody(destination, sprite, level, alpha)
		return
	}
	var atlas atlasGraphics
	switch sprite.Atlas {
	case "ships":
		atlas = g.graphics.levels[level-1].ships
	case "dive":
		atlas = g.graphics.levels[level-1].dive
	case "guardian-parts":
		atlas = g.graphics.levels[level-1].guardianParts
	case "guardians":
		atlas = g.graphics.levels[level-1].guardians
	case "common":
		atlas = g.graphics.levels[level-1].common
	case "shop":
		atlas = g.graphics.shop
	case "moving", "level":
		atlas = g.graphics.levels[level-1].moving
	case "fixed":
		atlas = g.graphics.levels[level-1].fixed
	case "shots", "enemy-shots":
		atlas = g.graphics.levels[level-1].shots
	default:
		return
	}
	image, ok := atlas[sprite.Sprite]
	if !ok {
		return
	}
	x, y := sprite.X, sprite.Y
	if sprite.Interpolate {
		x = lerp(sprite.PreviousX, x, alpha)
		y = lerp(sprite.PreviousY, y, alpha)
	}
	x -= float64(image.anchorX)
	y -= float64(image.anchorY)
	if x+float64(image.width) <= 0 || y+float64(image.height) <= 0 || x >= float64(destination.Bounds().Dx()) || y >= float64(destination.Bounds().Dy()) {
		return
	}
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	picture := image.image
	if sprite.Flash && image.flash != nil {
		picture = image.flash
	}
	if sprite.Materializing {
		g.graphics.materialPosition[0], g.graphics.materialPosition[1] = float32(x), float32(y)
		shaderOp := ebiten.DrawRectShaderOptions{Uniforms: g.graphics.materialUniforms}
		shaderOp.Images[0], shaderOp.Images[1] = picture, g.graphics.terrainMask
		shaderOp.GeoM.Translate(x, y)
		destination.DrawRectShader(image.width, image.height, g.graphics.materialShader, &shaderOp)
	} else {
		destination.DrawImage(picture, &op)
	}
}

func (g *Game) drawLaser(destination *ebiten.Image, beam SpriteView, level int, alpha float64) {
	if beam.Length < 0 {
		return
	}
	tier := max(0, min(2, beam.Tier))
	x, y := beam.X, beam.Y
	if beam.Interpolate {
		x = lerp(beam.PreviousX, x, alpha)
		y = lerp(beam.PreviousY, y, alpha)
	}
	x += float64(4 - tier*2)
	atlas := g.graphics.levels[level-1].common
	if beam.Length >= 65 {
		g.drawAtlasSprite(destination, atlas, fmt.Sprintf("laser-beam-bottom-%d", tier), x, y+1)
	}
	g.drawAtlasSprite(destination, atlas, fmt.Sprintf("laser-beam-top-%d", tier), x, y-float64(beam.Length))
	stem := atlas[fmt.Sprintf("laser-beam-stem-%d", tier)]
	if stem.image == nil {
		return
	}
	if beam.Length > 0 {
		op := ebiten.DrawImageOptions{}
		op.GeoM.Scale(1, float64(beam.Length))
		op.GeoM.Translate(x, y-float64(beam.Length)+1)
		destination.DrawImage(stem.image, &op)
	}
}

func (g *Game) drawSpark(destination *ebiten.Image, spark SpriteView, level int, alpha float64) {
	x, y := spark.X, spark.Y
	if spark.Interpolate {
		x = lerp(spark.PreviousX, x, alpha)
		y = lerp(spark.PreviousY, y, alpha)
	}
	px, py := int(x), int(y)
	if px < 0 || py < 0 || px+3 > destination.Bounds().Dx() || py+3 > destination.Bounds().Dy() {
		return
	}
	g.graphics.sparkScratch.Clear()
	copyOp := ebiten.DrawImageOptions{}
	copyOp.GeoM.Translate(float64(-px), float64(-py))
	g.graphics.sparkScratch.DrawImage(destination, &copyOp)
	op := ebiten.DrawRectShaderOptions{Uniforms: g.graphics.levels[level-1].sparkUniforms}
	op.Images[0] = g.graphics.sparkScratch
	op.GeoM.Translate(float64(px), float64(py))
	destination.DrawRectShader(3, 3, g.graphics.sparkShader, &op)
}

func (g *Game) drawTileBody(destination *ebiten.Image, body SpriteView, level int, alpha float64) {
	x, y := body.X, body.Y
	if body.Interpolate {
		x = lerp(body.PreviousX, x, alpha)
		y = lerp(body.PreviousY, y, alpha)
	}
	for row := 0; row < body.Patch.Rows; row++ {
		for column := 0; column < body.Patch.Columns; column++ {
			index := row*body.Patch.Columns + column
			if index >= len(body.Patch.Tiles) {
				return
			}
			tile := g.graphics.levels[level-1].tiles[body.Patch.Tiles[index]]
			if body.Flash {
				tile = g.graphics.levels[level-1].flashTiles[body.Patch.Tiles[index]]
			}
			if tile == nil {
				continue
			}
			op := ebiten.DrawImageOptions{}
			op.GeoM.Translate(x+float64(column*16), y+float64(row*16))
			destination.DrawImage(tile, &op)
		}
	}
}

func flashPixels(source *image.NRGBA, region image.Rectangle, c [4]uint8) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, region.Dx(), region.Dy()))
	for y := 0; y < region.Dy(); y++ {
		for x := 0; x < region.Dx(); x++ {
			old := source.NRGBAAt(region.Min.X+x, region.Min.Y+y)
			if old.A != 0 {
				out.SetNRGBA(x, y, color.NRGBA{R: c[0], G: c[1], B: c[2], A: old.A})
			}
		}
	}
	return out
}

func (g *Game) drawShop(screen *ebiten.Image) {
	if g.shop != nil {
		switch g.shop.Phase {
		case shopui.EndingDot, shopui.EndingDotFade:
			pattern := [3][4]uint8{{5, 7, 7, 5}, {7, 7, 7, 7}, {5, 7, 7, 5}}
			for y, row := range pattern {
				for x, index := range row {
					c := g.Bundle.ShopScene.EndingPalette[index]
					vector.FillRect(screen, float32(172+x), float32(100+y), 1, 1, color.RGBA{c[0], c[1], c[2], c[3]}, false)
				}
			}
			return
		case shopui.EndingWait:
			return
		}
	}
	if g.shop == nil {
		return
	}
	s, scene := g.shop, &g.Bundle.ShopScene
	for row := 0; row < 4; row++ {
		for column := 0; column < 5; column++ {
			x, y := float64(column*41), float64(row*41)
			g.drawAtlasSprite(screen, g.graphics.shop, "shop-background-1", x, y)
			g.drawAtlasSprite(screen, g.graphics.shop, "shop-background-2", x, y)
		}
	}
	screen.DrawImage(g.graphics.shopBase, nil)
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(scene.PortraitX), float64(scene.PortraitY))
	screen.DrawImage(g.graphics.portraits[s.Mouth], &op)
	for _, ambient := range scene.Ambient {
		g.drawAnchoredAtlasSprite(screen, g.graphics.shopControls, animationImage(ambient.Animation, s.Frame), float64(ambient.X), float64(ambient.Y))
	}
	if s.BlinkFrames > 0 {
		for _, control := range scene.Controls {
			if control.ID == "blink-left" || control.ID == "blink-right" {
				g.drawAnchoredAtlasSprite(screen, g.graphics.shopControls, control.Sprite, float64(control.X), float64(control.Y))
			}
		}
	}
	for index, entry := range s.Entries {
		cell := scene.Cells[index]
		if s.Television[index] <= 2 || !entry.Available {
			if s.Television[index] != -8 {
				g.drawShopNoise(screen, index, cell.X, cell.Y)
			}
			if counter := s.Television[index]; counter < 0 && counter >= -9 && len(scene.TVTransition) > counter+9 {
				g.drawAnchoredAtlasSprite(screen, g.graphics.shopControls, scene.TVTransition[counter+9], float64(cell.X), float64(cell.Y))
			}
			continue
		}
		name := ""
		if entry.More {
			name = animationImage(scene.PageAnimation, max(0, s.IconPasses[index]-1))
		} else {
			id := g.Bundle.Shop.Items[int(entry.Item)-1].ID
			for _, animation := range g.Bundle.ShopArt.Animations {
				if animation.ID == id {
					name = animationImage(animation, max(0, s.IconPasses[index]-1))
					break
				}
			}
		}
		atlas := g.graphics.shop
		if entry.More {
			atlas = g.graphics.shopControls
		}
		sprite, ok := atlas[name]
		if !ok {
			continue
		}
		g.drawAtlasSprite(screen, atlas, name, float64(cell.X+16-sprite.width/2), float64(cell.Y+14-sprite.height/2))
	}
	action := "buy"
	if s.Selling {
		action = "sell"
	}
	for _, control := range scene.Controls {
		selected := s.Row == 4 && ((s.Column == 0 && strings.HasPrefix(control.ID, "exit")) || (s.Column != 0 && strings.HasPrefix(control.ID, action)))
		if (control.ID == "exit" || control.ID == action) && !selected || (control.ID == "exit-active" || control.ID == action+"-active") && selected {
			g.drawAnchoredAtlasSprite(screen, g.graphics.shopControls, control.Sprite, float64(control.X), float64(control.Y))
		}
	}
	if s.Row < 4 {
		cell := scene.Cells[s.Row*5+s.Column]
		g.drawAnchoredAtlasSprite(screen, g.graphics.shopControls, "shop-control-cursor-active", float64(cell.CursorX), float64(cell.CursorY))
	}
	amount := fmt.Sprintf("%07d", s.DisplayMoney)
	g.drawGlyphs(screen, g.graphics.cashFont, amount, scene.MoneyX, scene.MoneyY, scene.CashFont.Width)
	for _, glyph := range s.Dialogue[:s.Revealed] {
		g.drawGlyphs(screen, g.graphics.smallFont, string(glyph.Character), glyph.X, glyph.Y, scene.Font.Width)
	}
	if s.HandRemaining > 0 && s.HandFrame < len(scene.SaleHand) {
		frame := scene.SaleHand[s.HandFrame]
		g.drawAnchoredAtlasSprite(screen, g.graphics.shopControls, frame.Sprite, float64(frame.X), float64(frame.Y))
	}
	if s.DisplayPhase == shopui.HeadphoneHand && s.IntroHandFrame < len(scene.IntroHand) {
		frame := scene.IntroHand[s.IntroHandFrame]
		g.drawAnchoredAtlasSprite(screen, g.graphics.shopControls, frame.Sprite, float64(frame.X), float64(frame.Y))
	}
	g.drawShopTransition(screen)
}

func (g *Game) drawShopTransition(screen *ebiten.Image) {
	s := g.shop
	if s == nil {
		return
	}
	if s.Headphones > 8 {
		g.drawTransitionStrip(screen, "shop-transition-strip-0", 48-s.Headphones, s.Headphones-8, 215, 9)
		g.drawTransitionStrip(screen, "shop-transition-strip-1", 0, s.Headphones-8, 215, 113-s.Headphones)
	}
	if s.Headphones > 0 {
		for index, x := range []int{215, 247, 279} {
			g.drawAtlasSprite(screen, g.graphics.shopTransition, fmt.Sprintf("shop-headphones-%d", index), float64(x), float64(s.Headphones+1))
			g.drawAtlasSprite(screen, g.graphics.shopTransition, fmt.Sprintf("shop-headphones-%d", index+3), float64(x), float64(112-s.Headphones))
		}
	}
	if s.Headphones > 0 && s.Headphones < 8 {
		g.drawAtlasSprite(screen, g.graphics.shopTransition, "shop-transition-border-0", 208, 0)
		g.drawAtlasSprite(screen, g.graphics.shopTransition, "shop-transition-border-1", 208, 105)
	}
	if s.LowerOverlay > 0 {
		g.drawTransitionStrip(screen, "shop-transition-strip-2", 0, s.LowerOverlay, 215, 114+68-s.LowerOverlay)
	}
}

func (g *Game) drawTransitionStrip(screen *ebiten.Image, name string, sourceY, height, x, y int) {
	sprite, ok := g.graphics.shopTransition[name]
	if !ok || height <= 0 || sourceY < 0 || sourceY+height > sprite.height {
		return
	}
	region := g.graphics.transitionStrips[fmt.Sprintf("%s:%d:%d", name, sourceY, height)]
	if region == nil {
		return
	}
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(region, &op)
}

func animationImage(animation visualassets.ItemAnimation, frame int) string {
	if len(animation.Frames) == 0 {
		return ""
	}
	if frame >= len(animation.Frames) {
		frame = animation.LoopFrom + (frame-animation.LoopFrom)%(len(animation.Frames)-animation.LoopFrom)
	}
	return animation.Frames[frame]
}

func (g *Game) drawAtlasSprite(screen *ebiten.Image, atlas atlasGraphics, name string, x, y float64) {
	sprite, ok := atlas[name]
	if !ok {
		return
	}
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(x-float64(sprite.anchorX), y-float64(sprite.anchorY))
	screen.DrawImage(sprite.image, &op)
}

func (g *Game) drawGlyphs(screen *ebiten.Image, font map[rune]*ebiten.Image, text string, x, y, advance int) {
	for _, character := range text {
		if glyph := font[character]; glyph != nil {
			op := ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(glyph, &op)
		}
		x += advance
	}
}

func (g *Game) drawShopNoise(screen *ebiten.Image, index, x, y int) {
	pixels := shopui.TelevisionPixels(g.shop.Noise[index], g.Bundle.ShopArt.Palette)
	g.graphics.shopNoise[index].WritePixels(pixels[:])
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(g.graphics.shopNoise[index], &op)
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func wrapLerp(a, b, t, period float64) float64 {
	delta := b - a
	if delta > period/2 {
		delta -= period
	} else if delta < -period/2 {
		delta += period
	}
	result := math.Mod(a+delta*t, period)
	if result < 0 {
		result += period
	}
	return result
}

func remapPalette(picture *image.NRGBA, from, to [16][4]uint8) *image.NRGBA {
	if from == to {
		return picture
	}
	colors := make(map[color.NRGBA]color.NRGBA, 16)
	for i, a := range from {
		b := to[i]
		colors[color.NRGBA{R: a[0], G: a[1], B: a[2], A: a[3]}] = color.NRGBA{R: b[0], G: b[1], B: b[2], A: b[3]}
	}
	out := image.NewNRGBA(picture.Bounds())
	for y := picture.Bounds().Min.Y; y < picture.Bounds().Max.Y; y++ {
		for x := picture.Bounds().Min.X; x < picture.Bounds().Max.X; x++ {
			c := picture.NRGBAAt(x, y)
			if mapped, ok := colors[c]; ok {
				c = mapped
			}
			out.SetNRGBA(x, y, c)
		}
	}
	return out
}

func (g *Game) drawAnchoredAtlasSprite(destination *ebiten.Image, atlas atlasGraphics, name string, x, y float64) {
	sprite, ok := atlas[name]
	if !ok {
		return
	}
	g.drawAtlasSprite(destination, atlas, name, x-float64(sprite.anchorX), y-float64(sprite.anchorY))
}

func (g *Game) drawFifthColumn(destination *ebiten.Image, view SpriteView, level int, alpha float64) {
	x, y := view.X, view.Y
	if view.Interpolate {
		x = lerp(view.PreviousX, x, alpha)
		y = lerp(view.PreviousY, y, alpha)
	}
	atlas := g.graphics.levels[level-1].guardianParts
	var top, bottom string
	for _, group := range g.Bundle.Levels[level-1].GuardianGroups {
		if group.ID != "middle-guardian" {
			continue
		}
		for _, clip := range group.Animations {
			if clip.ID == "fifth-laser-top" {
				top = clip.Animation.Frames[0].Sprite
			}
			if clip.ID == "fifth-laser-bottom" {
				bottom = clip.Animation.Frames[0].Sprite
			}
		}
	}
	if view.Tier != 0 || view.Length == 48 {
		g.drawAnchoredAtlasSprite(destination, atlas, top, x, y)
	}
	if view.Tier == 0 || view.Length == 48 {
		g.drawAnchoredAtlasSprite(destination, atlas, bottom, x, y+float64(view.Length)-1)
	}
	colors := [12]uint8{15, 7, 14, 14, 14, 14, 14, 14, 14, 14, 7, 15}
	for column, index := range colors {
		c := g.Bundle.Levels[level-1].Terrain.Palette[index]
		vector.FillRect(destination, float32(x)+float32(column), float32(y), 1, float32(view.Length), color.RGBA{c[0], c[1], c[2], c[3]}, false)
	}
}
