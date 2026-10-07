package app

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	ebitenaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"xenon2/internal/audio"
	"xenon2/internal/engine"
	"xenon2/internal/presentation"
	"xenon2/internal/shopui"
	"xenon2/internal/visualassets"
)

const (
	ScreenWidth     = 320
	ScreenHeight    = 200
	PlayfieldHeight = 192
)

type Screen uint8

const (
	TitleScreen Screen = iota
	LevelScreen
	ShopScreen
	PresentationScreen
)

// Input is sampled at the display rate and consumed by the simulation driver.
type Input struct {
	Motion      engine.MotionInput
	Fire        bool
	DivePressed bool
}

// SpriteView identifies exported artwork and screen-space anchor coordinates.
type SpriteView struct {
	Materializing        bool
	Order                int
	Tier, Length         int
	Kind                 string
	Patch                visualassets.TilePatch
	Flash                bool
	Layer                string
	ID                   int
	Atlas                string
	Sprite               string
	X, Y                 float64
	PreviousX, PreviousY float64
	Interpolate          bool
}

// SceneFrame is a presentation snapshot, independent of original memory.
type SceneFrame struct {
	BackgroundStars                          *engine.BackgroundStarfield
	PlayerNumber, PlayerCount                int
	PlayerScores, PlayerLives, PlayerShields [2]int
	ContinueCredits                          int
	DivePhase                                int
	PlayerSprite                             string
	Ready, GameOver                          bool
	FreezeInterpolation                      bool
	PaletteMask                              uint16
	Shades                                   bool
	Level                                    int
	CameraY, BackgroundX, BackgroundY        float64
	Player                                   engine.PlayerMotionState
	Sprites                                  []SpriteView
	HUD                                      []SpriteView
	Score, Money, Shield, Lives              int
	Diagnostic                               bool
	PlayerAlive                              bool
	TerrainMap                               []uint16
}

// Driver is the boundary between the pure Go game rules and their renderer.
type Driver interface {
	Advance(Input) error
	Frame() SceneFrame
}

type Config struct {
	Level       int
	Frames      int
	Screenshot  string
	Mute        bool
	StartScreen Screen
}

type Game struct {
	gameOverRunning          bool
	presentationStarPhase    presentation.Phase
	continueAfterScores      bool
	presentationInput        presentation.Input
	director                 *presentation.Director
	onContinue               func() error
	readyRunning             bool
	starfield                *presentation.Starfield
	menuClock                engine.FrameClock
	shop                     *shopui.State
	palClock                 engine.FrameClock
	Bundle                   *Bundle
	Config                   Config
	Screen                   Screen
	View                     SceneFrame
	previous                 SceneFrame
	Driver                   Driver
	clock                    engine.FrameClock
	graphics                 graphics
	stream                   *audio.Stream
	player                   *ebitenaudio.Player
	music                    bool
	soundtrack               string
	menu                     int
	updates                  int
	pendingFire, pendingDive bool
	done                     bool
	capturePending           bool
	err                      error
}

// New creates a resource viewer until a verified simulation driver is attached.
// It does not introduce substitute enemies, weapon rules or guardian behavior.
func New(bundle *Bundle) (*Game, error) {
	if err := bundle.Validate(); err != nil {
		return nil, err
	}
	stream, err := audio.NewStream(bundle.AudioBank, bundle.Waveforms, 44100)
	if err != nil {
		return nil, err
	}
	g := &Game{Bundle: bundle, Screen: TitleScreen, clock: engine.NewFrameClock(25, 60), stream: stream, music: true}
	g.graphics = prepareGraphics(bundle)
	g.palClock = engine.NewFrameClock(50, 60)
	random := engine.NewRandomState()
	g.starfield = presentation.NewStarfield(&random, bundle.Presentation.StarColors)
	g.director = presentation.NewDirector(&bundle.Presentation)
	g.menuClock = engine.NewFrameClock(25, 60)
	shader, err := ebiten.NewShader([]byte(paletteShaderSource))
	if err != nil {
		return nil, err
	}
	g.graphics.paletteShader = shader
	sparkShader, err := ebiten.NewShader([]byte(sparkShaderSource))
	if err != nil {
		return nil, err
	}
	g.graphics.sparkShader = sparkShader
	g.graphics.sparkScratch = ebiten.NewImage(3, 3)
	materialShader, err := ebiten.NewShader([]byte(terrainClippedSpriteShaderSource))
	if err != nil {
		return nil, err
	}
	g.graphics.materialShader = materialShader
	backgroundStarShader, err := ebiten.NewShader([]byte(backgroundStarShaderSource))
	if err != nil {
		return nil, err
	}
	g.graphics.backgroundStarShader = backgroundStarShader
	g.ResetDiagnosticLevel(1)
	if err = g.StartLevel(1); err != nil {
		return nil, err
	}
	return g, nil
}

// SetDriver replaces diagnostics with a snapshot from the verified game engine.
func (g *Game) SetDriver(driver Driver) {
	g.Driver = driver
	if driver != nil {
		g.View = driver.Frame()
		g.rememberFrameHistory()
	}
}

func (g *Game) ResetDiagnosticLevel(level int) {
	if level < 1 || level > len(g.Bundle.Levels) {
		level = 1
	}
	l := g.Bundle.Levels[level-1]
	g.View = SceneFrame{Level: level, CameraY: float64(l.Terrain.Rows*l.Terrain.TileSize - PlayfieldHeight), Player: engine.PlayerMotionState{X: 160, Y: 176}, Shield: 39, Lives: 3, Diagnostic: true, PlayerAlive: true}
	g.rememberFrameHistory()
	g.clock = engine.NewFrameClock(25, 60)
}

func (g *Game) Update() error {
	if g.err != nil {
		return g.err
	}
	if g.done {
		return ebiten.Termination
	}
	g.updates++
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.Screen == TitleScreen {
			return ebiten.Termination
		}
		g.Screen = TitleScreen
		g.selectMusic()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.music = !g.music
		g.selectMusic()
	}
	switch g.Screen {
	case PresentationScreen:
		if err := g.updatePresentation(); err != nil {
			return err
		}
	case TitleScreen:
		for ticks := g.menuClock.Advance(); ticks > 0; ticks-- {
			g.starfield.Advance()
			for i, star := range g.starfield.Stars {
				if g.menuPixelOccupied(star.ScreenX, star.ScreenY) {
					g.starfield.Covered(i)
				}
			}
		}
		g.updateTitle()
	case ShopScreen:
		if err := g.updateShop(); err != nil {
			return err
		}
	case LevelScreen:
		if g.View.Ready && !g.readyRunning {
			player := g.View.PlayerNumber
			if player == 0 {
				player = 1
			}
			g.resetPresentationStars(presentation.ReadyMessage)
			g.director.BeginReady(player)
			g.stream.StopEffects()
			g.Screen = PresentationScreen
			g.readyRunning = true
			g.selectMusic()
			break
		}
		if g.View.GameOver && !g.gameOverRunning {
			g.gameOverRunning = true
			g.stream.StopEffects()
			if g.director.InsertScore(g.View.Score) {
				g.continueAfterScores = true
			} else if g.View.ContinueCredits > 0 {
				g.director.BeginContinue()
			} else {
				g.director.BeginGameOver()
			}
			g.resetPresentationStars(g.director.Phase)
			g.Screen = PresentationScreen
			g.selectMusic()
			break
		}
		for ticks := g.palClock.Advance(); ticks > 0; ticks-- {
			if source, ok := g.Driver.(interface{ AdvancePALTick() }); ok {
				source.AdvancePALTick()
				if err := g.consumeDriverAudio(); err != nil {
					return err
				}
				g.View = g.Driver.Frame()
				if source, ok := g.Driver.(interface{ ConsumeTurnChange() bool }); ok && source.ConsumeTurnChange() {
					g.rememberFrameHistory()
					g.readyRunning = false
					g.gameOverRunning = false
				}
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
			if err := g.EnterShop(false); err != nil {
				return err
			}
			break
		}
		if g.View.Diagnostic {
			for n, key := range []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5} {
				if inpututil.IsKeyJustPressed(key) {
					if err := g.StartLevel(n + 1); err != nil {
						return err
					}
				}
			}
		}
		g.pendingFire = g.pendingFire || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyControl)
		g.pendingDive = g.pendingDive || inpututil.IsKeyJustPressed(ebiten.KeyAlt)
		input := Input{Motion: engine.MotionInput{Left: ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA), Right: ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD), Up: ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW), Down: ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS)}, Fire: ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyControl) || g.pendingFire, DivePressed: g.pendingDive}
		for steps := g.clock.Advance(); steps > 0; steps-- {
			g.rememberFrameHistory()
			if g.Driver != nil {
				g.deliverEffectActivity()
				if err := g.Driver.Advance(input); err != nil {
					return err
				}
				if err := g.consumeDriverAudio(); err != nil {
					return err
				}
				g.View = g.Driver.Frame()
				if source, ok := g.Driver.(interface{ ConsumeTurnChange() bool }); ok && source.ConsumeTurnChange() {
					g.rememberFrameHistory()
					g.readyRunning = false
					g.gameOverRunning = false
				}
			}
			g.pendingFire, g.pendingDive = false, false
		}
	}
	if g.Config.Frames > 0 && g.updates >= g.Config.Frames {
		g.capturePending = true
	}
	return nil
}

func (g *Game) menuPixelOccupied(x, y int) bool {
	p := &g.Bundle.Presentation
	if fontPixelOccupied(p.Font, p.MenuHeading, 0, 4, p.Font.Width, x, y) {
		return true
	}
	for i, line := range p.Menu {
		text := line.Text
		if i == g.menu {
			text = line.SelectedText
		}
		if line.ID == "music" {
			if g.music {
				text = p.MusicOn
				if i == g.menu {
					text = p.MusicOnSelected
				}
			} else {
				text = p.MusicOff
				if i == g.menu {
					text = p.MusicOffSelected
				}
			}
		}
		if fontPixelOccupied(p.Font, text, 0, line.CenterY-8, p.Font.Width, x, y) {
			return true
		}
	}
	return fontPixelOccupied(g.Bundle.Font, g.creditCaption(3), 176, 184, 8, x, y)
}

func fontPixelOccupied(font visualassets.Font, text string, left, top, advance, x, y int) bool {
	if x < left || y < top || y >= top+font.Height {
		return false
	}
	characterIndex := (x - left) / advance
	if characterIndex < 0 || characterIndex >= len(text) {
		return false
	}
	character := rune(text[characterIndex])
	glyph := strings.IndexRune(font.Characters, character)
	if glyph < 0 {
		return false
	}
	px := (glyph%font.Columns)*font.Width + (x - left - characterIndex*advance)
	py := (glyph/font.Columns)*font.Height + y - top
	c := font.Image.NRGBAAt(px, py)
	return c.R != 0 || c.G != 0 || c.B != 0
}

func (g *Game) rememberFrameHistory() {
	g.previous = g.View
	// Actor interpolation uses the world's own previous coordinates. History
	// keeps only scalar presentation fields, never reused actor/map slices.
	g.previous.Sprites = nil
	g.previous.HUD = nil
	g.previous.TerrainMap = nil
}

func (g *Game) deliverEffectActivity() {
	if target, ok := g.Driver.(interface{ SetEffectActivity([4]bool) }); ok {
		var active [4]bool
		for channel := range active {
			active[channel] = g.stream.EffectActive(channel)
		}
		target.SetEffectActivity(active)
	}
}

func (g *Game) consumeDriverAudio() error {
	if source, ok := g.Driver.(interface{ ConsumeEffectStop() bool }); ok {
		if source.ConsumeEffectStop() && !g.Config.Mute {
			g.stream.QueueStopEffects()
		}
	}

	if source, ok := g.Driver.(interface{ ConsumeImmediateSounds() [4]string }); ok {
		for channel, id := range source.ConsumeImmediateSounds() {
			if id != "" && !g.Config.Mute {
				if err := g.stream.PlayEffect(id, channel); err != nil {
					return err
				}
			}
		}
	}
	if source, ok := g.Driver.(interface{ ConsumeSoundRequests() [4]string }); ok {
		for channel, id := range source.ConsumeSoundRequests() {
			if id != "" && !g.Config.Mute {
				if err := g.stream.QueueEffect(id, channel); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *Game) updateTitle() {
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.menu = (g.menu + 2) % 3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.menu = (g.menu + 1) % 3
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if x >= 0 && x < 320 {
			for i, line := range g.Bundle.Presentation.Menu {
				if y >= line.CenterY-8 && y < line.CenterY+14 {
					g.menu = i
					g.activateMenu()
					break
				}
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.activateMenu()
	}
}

func (g *Game) activateMenu() {
	if g.menu >= 2 {
		g.music = !g.music
		g.selectMusic()
		return
	}
	if err := g.StartSession(max(1, g.Config.Level), g.menu+1); err != nil {
		g.err = err
		return
	}
	g.director.BeginStart()
	g.Screen = PresentationScreen
	g.selectMusic()
}

var audioContext struct {
	sync.Once
	context *ebitenaudio.Context
}

func (g *Game) EnableAudio() error {
	if g.Config.Mute {
		return nil
	}
	audioContext.Do(func() { audioContext.context = ebitenaudio.NewContext(44100) })
	player, err := audioContext.context.NewPlayer(g.stream)
	if err != nil {
		return err
	}
	g.player = player
	player.Play()
	g.selectMusic()
	return nil
}

func (g *Game) selectMusic() {
	id := ""
	if !g.Config.Mute {
		if g.music && g.Screen == LevelScreen && !g.View.Ready && !g.View.GameOver {
			id = "megablast-main"
		}
		if g.Screen == PresentationScreen && !g.readyRunning && !g.gameOverRunning {
			if g.director.AttractMusic() {
				id = "megablast-menu"
			}
		}
	}
	if id == g.soundtrack {
		return
	}
	g.soundtrack = id
	if id == "" {
		g.stream.StopMusic()
		return
	}
	if err := g.stream.PlayMusic(id); err != nil {
		g.err = err
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.drawScreen(screen)
	if g.capturePending {
		if g.Config.Screenshot != "" {
			g.err = saveScreenshot(screen, g.Config.Screenshot)
		}
		g.done = true
		g.capturePending = false
	}
}

func (g *Game) Layout(int, int) (int, int) { return ScreenWidth, ScreenHeight }

func saveScreenshot(screen *ebiten.Image, path string) error {
	picture := image.NewNRGBA(image.Rect(0, 0, ScreenWidth, ScreenHeight))
	screen.ReadPixels(picture.Pix)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err = png.Encode(file, picture); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// Run keeps display updates at sixty per second; driver steps use their own clock.
func Run(bundle *Bundle, config Config) error {
	if config.Level == 0 {
		config.Level = 1
	}
	if config.Level < 1 || config.Level > 5 || config.Frames < 0 {
		return fmt.Errorf("invalid level or frame limit")
	}
	g, err := New(bundle)
	if err != nil {
		return err
	}
	g.Config = config
	g.ResetDiagnosticLevel(config.Level)
	if err = g.StartLevel(config.Level); err != nil {
		return err
	}
	g.Screen = config.StartScreen
	if config.StartScreen == PresentationScreen {
		g.BeginAttract()
	}
	if config.StartScreen == ShopScreen {
		if err = g.EnterShop(false); err != nil {
			return err
		}
	}
	if err = g.EnableAudio(); err != nil {
		return err
	}
	if g.player != nil {
		defer g.player.Close()
	}
	ebiten.SetTPS(60)
	ebiten.SetWindowSize(ScreenWidth*3, ScreenHeight*3)
	ebiten.SetWindowTitle("Xenon 2 Go")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return ebiten.RunGame(g)
}
