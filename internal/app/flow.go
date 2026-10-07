package app

import (
	"fmt"
	"strings"
	"xenon2/internal/engine"
	"xenon2/internal/presentation"
)

type headerAction uint8

const (
	headerNone headerAction = iota
	headerStart
	headerShop
	headerReload
	headerNextStage
)

func (g *Game) levelData(number int) engine.LevelData {
	l := g.Bundle.Levels[number-1]
	return engine.LevelData{Number: number, Terrain: &l.Terrain, Paths: &l.Paths, Encounters: &l.Encounters, Actors: &l.Actors, FixedSprites: &l.FixedSprites, FixedTiles: &l.FixedTiles, PlayerStencil: &g.Bundle.Stencil, Rules: &l.Rules, Ships: &g.Bundle.Ships, Common: &g.Bundle.Common, Guardians: l.Guardians, GuardianGroups: l.GuardianGroups, GuardianParts: l.GuardianParts}
}

func (g *Game) beginHeader(text string, action headerAction) {
	g.fade, g.whenFadeEnds = nil, nil
	g.headerAction = action
	g.resetPresentationStars(presentation.HeaderIn)
	g.director.BeginHeader(text)
	g.Screen = PresentationScreen
	g.selectMusic()
}

func (g *Game) beginLoading(number int, reload bool, action headerAction) {
	text := strings.Replace(g.Bundle.Presentation.LoadingHeading, "1", fmt.Sprint(number), 1)
	if reload && len(text) >= 3 {
		text = text[:1] + "RE" + text[3:]
	}
	g.beginHeader(text, action)
}

func (g *Game) completeHeader() error {
	action := g.headerAction
	g.headerAction = headerNone
	switch action {
	case headerStart:
		if err := g.StartSession(max(1, g.Config.Level), g.pendingPlayers); err != nil {
			return err
		}
		g.Screen = LevelScreen
		g.beginPendingWorldPresentation()
	case headerShop:
		return g.EnterShop(g.shopFinal)
	case headerReload:
		driver := g.Driver.(*worldDriver)
		driver.world.ResumeShop()
		driver.world.PrimeBackgroundStars(1)
		g.View = driver.Frame()
		g.rememberFrameHistory()
		g.Screen = LevelScreen
		g.startLevelFade()
		g.selectMusic()
	case headerNextStage:
		return g.advanceCompletedStage()
	}
	return nil
}

func (g *Game) requestShop() error {
	driver, ok := g.Driver.(*worldDriver)
	if !ok {
		return fmt.Errorf("shop needs an active world")
	}
	world := driver.world
	g.shopFinal = world.LevelFinished
	g.stopGameplayMusic()
	// In the last stage, the first surviving player waits without seeing the
	// merchant ending. The final surviving completion shows it once.
	if g.shopFinal && world.Level.Number == 5 && driver.session != nil && driver.session.PlayerCount == 2 {
		other := driver.session.Current ^ 1
		if next := driver.session.Players[other]; next.Equipment.Lives > 0 && !next.GameOver && !driver.session.Completed[other] {
			return g.advanceCompletedStage()
		}
	}
	g.startFade(presentation.NewPaletteFadeOut(2), func() error { g.beginHeader(g.Bundle.Presentation.EnteringShopHeading, headerShop); return nil })
	return nil
}

func (g *Game) leaveShop() error {
	driver, ok := g.Driver.(*worldDriver)
	if !ok {
		return fmt.Errorf("shop return needs an active world")
	}
	if g.shopFinal {
		next := driver.world.Level.Number%5 + 1
		// A second surviving player may still be on the same stage, so admission
		// happens before deciding whether a next-stage loading caption is needed.
		if driver.session != nil {
			other := driver.session.Current ^ 1
			if driver.session.PlayerCount == 2 {
				if player := driver.session.Players[other]; player.Equipment.Lives > 0 && !player.GameOver && !driver.session.Completed[other] {
					return g.advanceCompletedStage()
				}
			}
		}
		g.beginLoading(next, false, headerNextStage)
	} else {
		g.beginLoading(driver.world.Level.Number, true, headerReload)
	}
	return nil
}

func (g *Game) advanceCompletedStage() error {
	driver, ok := g.Driver.(*worldDriver)
	if !ok {
		return fmt.Errorf("stage transition needs an active world")
	}
	if driver.session == nil {
		// Diagnostic level views still use the same original stage transition.
		driver.session = &engine.Session{Players: [2]*engine.World{driver.world, nil}, PlayerCount: 1, Difficulty: 1}
	}
	next := driver.world.Level.Number%5 + 1
	transition, err := driver.session.CompleteStage(g.levelData(next))
	if err != nil {
		return err
	}
	driver.world = driver.session.ActiveWorld()
	g.applyCheatOptions()
	g.gameMusicAfterFade = transition == engine.LoadedNextStage
	driver.turnChanged = true
	g.View = driver.Frame()
	g.rememberFrameHistory()
	g.readyRunning, g.gameOverRunning = false, false
	g.Screen = LevelScreen
	g.clock = g.newLogicClock()
	g.palClock = engine.NewFrameClock(50, 60)
	g.beginPendingWorldPresentation()
	return nil
}

func (g *Game) startFade(fade presentation.PaletteFade, after func() error) {
	g.fade = &fade
	g.whenFadeEnds = after
	g.palClock = engine.NewFrameClock(50, 60)
}

func (g *Game) startLevelFade() {
	g.backdropOnly = true
	g.startFade(presentation.NewPaletteFadeIn(), func() error {
		g.backdropOnly = false
		g.clock = g.newLogicClock()
		if g.gameMusicAfterFade {
			g.beginGameplayMusic()
		}
		return nil
	})
}
