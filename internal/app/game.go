package app

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	ebitenaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"xenon2/internal/audio"
	"xenon2/internal/engine"
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
)

// Input is sampled at the display rate and consumed by the simulation driver.
type Input struct {
	Motion      engine.MotionInput
	Fire        bool
	DivePressed bool
}

// SpriteView identifies exported artwork and screen-space anchor coordinates.
type SpriteView struct {
	ID                   int
	Atlas                string
	Sprite               string
	X, Y                 float64
	PreviousX, PreviousY float64
	Interpolate          bool
}

// SceneFrame is a presentation snapshot, independent of original memory.
type SceneFrame struct {
	Level                             int
	CameraY, BackgroundX, BackgroundY float64
	Player                            engine.PlayerMotionState
	Sprites                           []SpriteView
	HUD                               []SpriteView
	Score, Money, Shield, Lives       int
	Diagnostic                        bool
	PlayerAlive                       bool
	TerrainMap                        []uint16
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
	menu                     int
	updates                  int
	shopPage                 int
	shopFrame                int
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
	case TitleScreen:
		g.updateTitle()
	case ShopScreen:
		g.shopFrame += g.clock.Advance()
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			g.shopPage = (g.shopPage + 3) % 4
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			g.shopPage = (g.shopPage + 1) % 4
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.Screen = LevelScreen
			g.selectMusic()
		}
	case LevelScreen:
		if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
			g.Screen = ShopScreen
			g.stream.StopMusic()
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
				if err := g.Driver.Advance(input); err != nil {
					return err
				}
				g.View = g.Driver.Frame()
			}
			g.pendingFire, g.pendingDive = false, false
		}
	}
	if g.Config.Frames > 0 && g.updates >= g.Config.Frames {
		g.capturePending = true
	}
	return nil
}

func (g *Game) rememberFrameHistory() {
	g.previous = g.View
	// Actor interpolation uses the world's own previous coordinates. History
	// keeps only scalar presentation fields, never reused actor/map slices.
	g.previous.Sprites = nil
	g.previous.HUD = nil
	g.previous.TerrainMap = nil
}

func (g *Game) updateTitle() {
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.menu = (g.menu + 3) % 4
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.menu = (g.menu + 1) % 4
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if x >= 32 && x < 288 && y >= 88 && y < 168 {
			g.menu = (y - 88) / 20
			g.activateMenu()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.activateMenu()
	}
}

func (g *Game) activateMenu() {
	if g.menu >= 2 {
		g.music = g.menu == 2
		g.selectMusic()
		return
	}
	if err := g.StartLevel(g.View.Level); err != nil {
		g.err = err
		return
	}
	g.Screen = LevelScreen
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
	if !g.music || g.Config.Mute || g.Screen == ShopScreen {
		g.stream.StopMusic()
		return
	}
	id := "megablast-main"
	if g.Screen == TitleScreen {
		id = "megablast-menu"
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
