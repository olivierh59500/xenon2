package app

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
)

// RecordingGame retains ordinary simulation and the demo pilot while ignoring
// unrelated desktop input. The DCK recorder consumes the game's own PCM stream.
type RecordingGame struct {
	*Game
	completeLevel int
}

func NewRecordingGame() (*RecordingGame, error) {
	g, err := NewGameFromConfig(Config{Level: 1, StartScreen: PresentationScreen, Demo: true, HumanDemo: true})
	if err != nil {
		return nil, err
	}
	if err := g.EnableAudio(); err != nil {
		g.Close()
		return nil, err
	}
	return &RecordingGame{Game: g}, nil
}

func NewFirstLevelRecordingGame() (*RecordingGame, error) {
	g, err := NewRecordingGame()
	if err != nil {
		return nil, err
	}
	g.completeLevel = 1
	return g, nil
}

func (g *RecordingGame) Update() error {
	if err := g.advanceWithInput(inputFrame{}); err != nil {
		return err
	}
	if g.completeLevel != 0 {
		if driver, ok := g.Driver.(*worldDriver); ok && driver.session != nil {
			if driver.world.Level.Number > g.completeLevel {
				return ebiten.Termination
			}
			if driver.world.GameOver && driver.world.ContinueCredits == 0 {
				return fmt.Errorf("presentation exhausted ordinary recovery before completing level %d", g.completeLevel)
			}
		}
		if g.updates >= 60*1200 {
			return fmt.Errorf("presentation did not complete level %d within twenty minutes", g.completeLevel)
		}
	}
	return nil
}

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
