package app

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"strings"
	"xenon2/internal/engine"
	"xenon2/internal/presentation"
)

// BeginAttract starts the source presentation loop without changing game rules.
func (g *Game) BeginAttract() {
	g.gameMusicRunning, g.gameMusicAfterFade = false, false
	g.fade, g.whenFadeEnds = nil, nil
	if g.updates > 0 {
		g.resetPresentationStars(presentation.LogoDelay)
	}
	scores := g.director.Scores
	g.director = presentation.NewDirector(&g.Bundle.Presentation)
	g.creditClock = engine.NewRationalFrameClock(31, 2, 60)
	g.director.Scores = scores
	g.readyRunning, g.gameOverRunning = false, false
	g.presentationStarPhase = presentation.LogoDelay
	g.Screen = PresentationScreen
	g.selectMusic()
}

// beginPendingWorldPresentation admits the source director before the first
// draw of a new READY or final-loss state. No fallback logo frame is inserted.
func (g *Game) beginPendingWorldPresentation() bool {
	if g.View.Ready && !g.readyRunning {
		player := max(1, g.View.PlayerNumber)
		g.resetPresentationStars(presentation.ReadyMessage)
		g.director.BeginReady(player)
		g.stream.QueueStopEffects()
		g.Screen, g.readyRunning = PresentationScreen, true
		g.selectMusic()
		return true
	}
	if g.View.GameOver && !g.gameOverRunning {
		g.gameOverRunning = true
		g.stream.QueueStopEffects()
		if g.director.InsertScore(g.View.Score) {
			g.continueAfterScores = true
		} else if g.View.ContinueCredits > 0 || g.Config.Cheats.InfiniteCredits {
			g.director.BeginContinue()
		} else {
			g.director.BeginGameOver()
		}
		g.resetPresentationStars(g.director.Phase)
		g.Screen = PresentationScreen
		g.selectMusic()
		return true
	}
	return false
}

func (g *Game) SetContinueHandler(handler func() error) { g.onContinue = handler }

func (g *Game) updatePresentation(controls inputFrame) error {
	input := presentation.Input{Confirm: controls.confirm || controls.mousePressed}
	if controls.left {
		input.Horizontal = -1
	}
	if controls.right {
		input.Horizontal = 1
	}

	g.presentationInput.Confirm = g.presentationInput.Confirm || input.Confirm
	g.presentationInput.Horizontal = input.Horizontal
	clock := &g.menuClock
	if g.director.Phase == presentation.Credits {
		// The recorded A500 spends about six seconds on each 93-pass pair.
		// Use that measured average for the expensive credit zoom sequence.
		clock = &g.creditClock
	}
	for ticks := clock.Advance(); ticks > 0; ticks-- {
		switch result := g.director.Advance(g.presentationInput); result {
		case presentation.StartGame:
			g.restoreGameplayStars()
			g.beginLoading(max(1, g.Config.Level), false, headerStart)
		case presentation.HeaderShown:
			g.director.BeginHeaderExit()
		case presentation.HeaderFinished:
			g.restoreGameplayStars()
			if err := g.completeHeader(); err != nil {
				return err
			}
		case presentation.ShowMenu:
			g.Screen = TitleScreen
			g.selectMusic()
		case presentation.ResumeGame:
			g.restoreGameplayStars()
			if driver, ok := g.Driver.(*worldDriver); ok {
				g.deliverEffectActivity()
				if err := driver.Advance(Input{Fire: true}); err != nil {
					return err
				}
				if err := g.consumeDriverAudio(); err != nil {
					return err
				}
				driver.world.PrimeBackgroundStars(2)
				g.View = driver.Frame()
				g.rememberFrameHistory()
			}
			g.Screen = LevelScreen
			g.readyRunning = false
			if driver, ok := g.Driver.(*worldDriver); ok && driver.session != nil {
				switch driver.session.AfterReady() {
				case engine.ResumeMerchantEnding:
					if err := g.requestShop(); err != nil {
						return err
					}
				case engine.ResumeNextStage:
					g.beginLoading(driver.world.Level.Number%5+1, false, headerNextStage)
				}
				if g.Screen != LevelScreen {
					break
				}
			}
			if !g.gameMusicAfterFade {
				g.beginGameplayMusic()
			}
			g.startLevelFade()
			g.selectMusic()
		case presentation.ContinueAccepted:
			g.restoreGameplayStars()
			if g.onContinue != nil {
				if err := g.onContinue(); err != nil {
					return err
				}
				g.Screen = LevelScreen
				g.gameOverRunning = false
				g.beginPendingWorldPresentation()
			} else {
				g.finishPlayerGame()
			}
		case presentation.ContinueDeclined:
			g.restoreGameplayStars()
			g.resetPresentationStars(presentation.GameOverMessage)
			g.director.BeginGameOver()
		case presentation.PlayerGameFinished:
			g.restoreGameplayStars()
			g.finishPlayerGame()
		}
		if g.continueAfterScores && g.director.Phase == presentation.LogoDelay {
			g.continueAfterScores = false
			g.restoreGameplayStars()
			if g.View.ContinueCredits > 0 || g.Config.Cheats.InfiniteCredits {
				g.resetPresentationStars(presentation.ContinueIn)
				g.director.BeginContinue()
			} else {
				g.resetPresentationStars(presentation.GameOverMessage)
				g.director.BeginGameOver()
			}
		}
		g.presentationInput.Confirm = false
		if g.Screen == LevelScreen {
			g.selectMusic()
			continue
		}
		if g.director.DisplayPhase == presentation.MenuIn && g.presentationStarPhase != presentation.MenuIn {
			g.resetPresentationStars(presentation.MenuIn)
		}
		g.starfield.Advance()
		if driver, ok := g.Driver.(*worldDriver); ok {
			driver.world.SetRandomState(*g.starfield.Random)
		}
		g.selectMusic()
		for i, star := range g.starfield.Stars {
			if g.presentationPixelOccupied(star.ScreenX, star.ScreenY) {
				g.starfield.Covered(i)
			}
		}
	}
	return nil
}

func (g *Game) finishPlayerGame() {
	if driver, ok := g.Driver.(*worldDriver); ok && driver.session != nil && driver.session.DeclineContinue() {
		driver.world = driver.session.ActiveWorld()
		g.View = driver.Frame()
		g.rememberFrameHistory()
		g.readyRunning, g.gameOverRunning = false, false
		g.Screen = LevelScreen
		g.beginPendingWorldPresentation()
		return
	}
	g.BeginAttract()
	g.director.Advance(presentation.Input{})
}

func (g *Game) presentationPixelOccupied(x, y int) bool {
	d, p := g.director, &g.Bundle.Presentation
	if d.LogoScale > 0 {
		i := min(15, d.LogoScale-1)
		region := p.LogoZoom.Sprites[i]
		px, py := x-p.LogoZoomX[i], y-p.LogoZoomY[i]
		if px >= 0 && px < region.Width && py >= 0 && py < region.Height {
			c := p.LogoZoom.Image.NRGBAAt(region.X+px, region.Y+py)
			if c.R != 0 || c.G != 0 || c.B != 0 {
				return true
			}
		}
	}
	if g.creditOverlapVisible() {
		pixels := g.graphics.creditOverlapPixels[d.Captions[0].Text]
		if x >= 0 && x < 320 && y >= 112 && y < 134 {
			c := pixels.NRGBAAt(x, y-112)
			if c.R != 0 || c.G != 0 || c.B != 0 {
				return true
			}
		}
		return false
	}
	byName := g.graphics.textZoom
	for _, caption := range d.Captions {
		step := min(16, caption.Scale)
		if step < 1 {
			continue
		}
		left, top := 160-step*10, (caption.CenterY*2-step)/2
		if step == 16 {
			top = caption.CenterY - 8
		}
		letter := (x - left) / step
		if x < left || letter < 0 || letter >= len(caption.Text) {
			continue
		}
		glyph := strings.IndexRune(p.Font.Characters, rune(caption.Text[letter]))
		if glyph < 0 {
			continue
		}
		name := p.TextZoomRegions[step-1][glyph]
		meta := byName[name]
		px, py := x-left-letter*step, y-top
		if px >= 0 && px < meta.width && py >= 0 && py < meta.height {
			region := g.graphics.textZoomRegions[name]
			c := p.TextZoom.Image.NRGBAAt(region.X+px, region.Y+py)
			if c.R != 0 || c.G != 0 || c.B != 0 {
				return true
			}
		}
	}
	if d.ShowScores {
		for index, score := range d.Scores {
			line := fmt.Sprintf("%2d. %07d %s", index+1, score.Points, score.Initials)
			if fontPixelOccupied(g.Bundle.Font, line, 32, 38+index*16, 16, x, y) {
				return true
			}
		}
	}
	if d.ShowCredits && fontPixelOccupied(g.Bundle.Font, g.creditCaption(g.View.ContinueCredits), 176, 184, 8, x, y) {
		return true
	}
	return false
}

func (g *Game) drawPresentation(screen *ebiten.Image) {
	d := g.director
	p := &g.Bundle.Presentation
	if d.LogoScale > 0 {
		index := min(15, d.LogoScale-1)
		g.drawAtlasSprite(screen, g.graphics.logoZoom, p.LogoZoom.Sprites[index].Name, float64(p.LogoZoomX[index]), float64(p.LogoZoomY[index]))
	}
	if g.creditOverlapVisible() {
		op := ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, 112)
		screen.DrawImage(g.graphics.creditOverlaps[d.Captions[0].Text], &op)
	} else {
		for _, caption := range d.Captions {
			g.drawCaptionZoom(screen, caption)
		}
	}
	if d.ShowScores {
		for index, score := range d.Scores {
			line := fmt.Sprintf("%2d. %07d %s", index+1, score.Points, score.Initials)
			g.drawGlyphs(screen, g.graphics.font, line, 32, 38+index*16, 16)
		}
	}
	if d.ShowCredits {
		g.drawGlyphs(screen, g.graphics.font, g.creditCaption(g.View.ContinueCredits), 176, 184, 8)
	}
	g.drawStarfield(screen)
}

func (g *Game) drawCaptionZoom(screen *ebiten.Image, caption presentation.Caption) {
	step := caption.Scale
	if step < 1 {
		return
	}
	step = min(16, step)
	p := &g.Bundle.Presentation
	x, y := 160-step*10, (caption.CenterY*2-step)/2
	if step == 16 {
		y = caption.CenterY - 8
	}
	for _, character := range caption.Text {
		glyph := strings.IndexRune(p.Font.Characters, character)
		if glyph >= 0 {
			name := p.TextZoomRegions[step-1][glyph]
			g.drawAtlasSprite(screen, g.graphics.textZoom, name, float64(x), float64(y))
		}
		x += step
	}
}

func (g *Game) resetPresentationStars(phase presentation.Phase) {
	g.presentationStarPhase = phase
	random := *g.starfield.Random
	if driver, ok := g.Driver.(*worldDriver); ok {
		random = driver.world.RandomState()
	}
	g.starfield = presentation.NewStarfield(&random, g.Bundle.Presentation.StarColors)
	if driver, ok := g.Driver.(*worldDriver); ok {
		driver.world.SetRandomState(random)
	}
	g.menuClock = engine.NewFrameClock(25, 60)
}

func (g *Game) creditCaption(credits int) string {
	text := g.Bundle.Presentation.CreditsCaption
	if len(text) > 0 {
		text = text[:len(text)-1] + fmt.Sprint(credits)
	}
	return text
}

func (g *Game) creditOverlapVisible() bool {
	captions := g.director.Captions
	return len(captions) == 2 && captions[0].Text == captions[1].Text && captions[0].Scale == 16 && captions[1].Scale == 15 && captions[0].CenterY == 120 && captions[1].CenterY == 120
}

func (g *Game) restoreGameplayStars() {
	if driver, ok := g.Driver.(*worldDriver); ok {
		driver.world.SetRandomState(*g.starfield.Random)
		driver.world.ResetBackgroundStars()
		*g.starfield.Random = driver.world.RandomState()
	}
}
