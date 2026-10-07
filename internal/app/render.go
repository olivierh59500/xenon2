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

	"xenon2/internal/visualassets"
)

type spriteGraphic struct {
	image                           *ebiten.Image
	anchorX, anchorY, width, height int
}
type atlasGraphics map[string]spriteGraphic
type levelGraphics struct {
	background                          *ebiten.Image
	tiles                               map[uint16]*ebiten.Image
	moving, fixed, ships, common, shots atlasGraphics
}
type graphics struct {
	title               *ebiten.Image
	font                map[rune]*ebiten.Image
	ships, common, shop atlasGraphics
	levels              [5]levelGraphics
	playfield           *ebiten.Image
}

func prepareGraphics(b *Bundle) graphics {
	g := graphics{title: ebiten.NewImageFromImage(b.Title.Image), font: make(map[rune]*ebiten.Image), ships: prepareAtlas(b.Ships.Atlas), common: prepareAtlas(b.Common), shop: prepareAtlas(b.ShopArt.Atlas), playfield: ebiten.NewImage(ScreenWidth, PlayfieldHeight)}
	font := ebiten.NewImageFromImage(remapPalette(b.Font.Image, b.Levels[0].Terrain.Palette, b.Title.Palette))
	for index, char := range b.Font.Characters {
		x, y := (index%b.Font.Columns)*b.Font.Width, (index/b.Font.Columns)*b.Font.Height
		g.font[char] = font.SubImage(image.Rect(x, y, x+b.Font.Width, y+b.Font.Height)).(*ebiten.Image)
	}
	for i, l := range b.Levels {
		atlas := ebiten.NewImageFromImage(l.Terrain.Atlas)
		v := levelGraphics{background: ebiten.NewImageFromImage(l.Terrain.Background), tiles: make(map[uint16]*ebiten.Image, len(l.Terrain.Tiles)), moving: prepareAtlas(l.Actors.Atlas), fixed: prepareAtlas(l.FixedSprites.Atlas)}
		v.shots = prepareAtlas(l.Rules.EnemyShots)
		ships, common := b.Ships.Atlas, b.Common
		ships.Image = remapPalette(ships.Image, b.Levels[0].Terrain.Palette, l.Terrain.Palette)
		common.Image = remapPalette(common.Image, b.Levels[0].Terrain.Palette, l.Terrain.Palette)
		v.ships, v.common = prepareAtlas(ships), prepareAtlas(common)
		for _, tile := range l.Terrain.Tiles {
			v.tiles[tile.ID] = atlas.SubImage(image.Rect(tile.X, tile.Y, tile.X+16, tile.Y+16)).(*ebiten.Image)
		}
		g.levels[i] = v
	}
	return g
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
	}
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(g.Bundle.Title.X), float64(g.Bundle.Title.Y))
	screen.DrawImage(g.graphics.title, &op)
	labels := []string{"1 PLAYER GAME", "2 PLAYER GAME", "MUSIC ON", "MUSIC OFF"}
	for i, label := range labels {
		y := 88 + i*20
		if i == g.menu {
			vector.DrawFilledRect(screen, 32, float32(y), 4, 16, color.RGBA{238, 170, 102, 255}, false)
		}
		g.drawText(screen, label, 48, float64(y))
	}
	if g.Driver == nil || g.View.Diagnostic {
		ebitenutil.DebugPrintAt(screen, "REFERENCE BUILD - GAME RULES IN PROGRESS", 8, 184)
	}
}

func (g *Game) drawLevel(screen *ebiten.Image) {
	level := g.View.Level
	if level < 1 || level > 5 {
		return
	}
	l := g.Bundle.Levels[level-1]
	gpu := &g.graphics.levels[level-1]
	alpha := g.clock.Fraction()
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
	playerX, playerY := lerp(float64(g.previous.Player.X), float64(g.View.Player.X), alpha), lerp(float64(g.previous.Player.Y), float64(g.View.Player.Y), alpha)
	frame := max(0, min(12, g.View.Player.Inertia+6))
	name := g.Bundle.Ships.SteeringFrames[frame]
	if g.View.PlayerAlive {
		g.drawSprite(field, SpriteView{Atlas: "ships", Sprite: name, X: playerX, Y: playerY}, level, alpha)
	}
	for _, source := range []string{"moving", "level", "shots", "enemy-shots", "common", "fixed"} {
		for _, sprite := range g.View.Sprites {
			if sprite.Atlas == source {
				g.drawSprite(field, sprite, level, alpha)
			}
		}
	}
	screen.DrawImage(field, nil)
	for _, sprite := range g.View.HUD {
		g.drawSprite(screen, sprite, level, alpha)
	}
	if g.View.Diagnostic {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("REFERENCE VIEW  LEVEL %d  ARROWS: MOVE  F2: SHOP", level), 0, 192)
	}
}

func (g *Game) drawSprite(destination *ebiten.Image, sprite SpriteView, level int, alpha float64) {
	var atlas atlasGraphics
	switch sprite.Atlas {
	case "ships":
		atlas = g.graphics.levels[level-1].ships
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
	destination.DrawImage(image.image, &op)
}

func (g *Game) drawShop(screen *ebiten.Image) {
	g.drawText(screen, "SHOP ART VIEWER", 40, 4)
	for cell := 0; cell < 8; cell++ {
		itemIndex := g.shopPage*8 + cell
		if itemIndex >= len(g.Bundle.Shop.Items) {
			break
		}
		item := g.Bundle.Shop.Items[itemIndex]
		x, y := float64(12+(cell%4)*76), float64(38+(cell/4)*68)
		g.drawSprite(screen, SpriteView{Atlas: "shop", Sprite: "shop-background-1", X: x, Y: y}, 1, 0)
		g.drawSprite(screen, SpriteView{Atlas: "shop", Sprite: "shop-background-2", X: x, Y: y}, 1, 0)
		for _, animation := range g.Bundle.ShopArt.Animations {
			if animation.ID != item.ID || len(animation.Frames) == 0 {
				continue
			}
			index := g.shopFrame
			if index >= len(animation.Frames) {
				index = animation.LoopFrom + (index-animation.LoopFrom)%(len(animation.Frames)-animation.LoopFrom)
			}
			name := animation.Frames[index]
			sprite := g.graphics.shop[name]
			g.drawSprite(screen, SpriteView{Atlas: "shop", Sprite: name, X: x + 20 - float64(sprite.width)/2 + float64(sprite.anchorX), Y: y + 18 - float64(sprite.height)/2 + float64(sprite.anchorY)}, 1, 0)
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%02d  %d", item.Order, item.Price), int(x), int(y)+42)
	}
	ebitenutil.DebugPrintAt(screen, "ORIGINAL ITEMS AND PRICES - DIAGNOSTIC GALLERY", 4, 174)
	ebitenutil.DebugPrintAt(screen, "LEFT/RIGHT: PAGE  ENTER: RETURN  ESC: MENU", 4, 186)
}

func (g *Game) drawText(destination *ebiten.Image, text string, x, y float64) {
	for _, char := range strings.ToUpper(text) {
		if glyph := g.graphics.font[char]; glyph != nil {
			op := ebiten.DrawImageOptions{}
			op.GeoM.Translate(x, y)
			destination.DrawImage(glyph, &op)
		}
		x += float64(g.Bundle.Font.Width)
	}
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
