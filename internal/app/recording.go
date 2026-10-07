package app

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
)

// RecordingGame retains ordinary simulation and the demo pilot while ignoring
// unrelated desktop input. The DCK recorder consumes the game's own PCM stream.
type RecordingGame struct{ *Game }

func NewRecordingGame() (*RecordingGame, error) {
	g, err := NewGameFromConfig(Config{Level: 1, StartScreen: PresentationScreen, Demo: true})
	if err != nil {
		return nil, err
	}
	if err := g.EnableAudio(); err != nil {
		g.Close()
		return nil, err
	}
	return &RecordingGame{Game: g}, nil
}

func (g *RecordingGame) Update() error { return g.advanceWithInput(inputFrame{}) }

func (g *RecordingGame) Draw(screen *ebiten.Image) { g.Game.Draw(screen) }

func (g *RecordingGame) RecordingChapter() string {
	switch g.Screen {
	case PresentationScreen:
		switch g.headerAction {
		case headerStart:
			return fmt.Sprintf("Loading level %d", max(1, g.Config.Level))
		case headerShop:
			return "Entering the merchant"
		case headerReload:
			return fmt.Sprintf("Reloading level %d", g.View.Level)
		case headerNextStage:
			return "Loading the next stage"
		}
		if g.readyRunning {
			return fmt.Sprintf("Level %d: get ready", g.View.Level)
		}
		if g.gameOverRunning {
			return "Score and continue"
		}
		return "Original intro and credits"
	case TitleScreen:
		return "Main menu"
	case LevelScreen:
		return fmt.Sprintf("Level %d: automatic gameplay", g.View.Level)
	case ShopScreen:
		return "Merchant: repairs and equipment"
	default:
		return "Xenon 2 Go"
	}
}
