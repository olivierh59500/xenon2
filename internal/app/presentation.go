package app

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"strings"
	"xenon2/internal/presentation"
)

// BeginAttract starts the source presentation loop without changing game rules.
func (g *Game) BeginAttract() {
	g.director = presentation.NewDirector(&g.Bundle.Presentation)
	g.Screen = PresentationScreen
}

func (g *Game) SetContinueHandler(handler func() error) { g.onContinue = handler }

func (g *Game) updatePresentation() error {
	input := presentation.Input{Confirm: inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyControl) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		input.Horizontal = -1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		input.Horizontal = 1
	}
	g.presentationInput.Confirm = g.presentationInput.Confirm || input.Confirm
	if input.Horizontal != 0 {
		g.presentationInput.Horizontal = input.Horizontal
	}
	for ticks := g.menuClock.Advance(); ticks > 0; ticks-- {
		g.starfield.Advance()
		switch result := g.director.Advance(g.presentationInput); result {
		case presentation.ShowMenu:
			g.Screen = TitleScreen
			g.selectMusic()
		case presentation.ResumeGame:
			if driver, ok := g.Driver.(*worldDriver); ok {
				if err := driver.Advance(Input{Fire: true}); err != nil {
					return err
				}
				g.View = driver.Frame()
				g.rememberFrameHistory()
			}
			g.Screen = LevelScreen
			g.readyRunning = false
		case presentation.ContinueAccepted:
			if g.onContinue != nil {
				if err := g.onContinue(); err != nil {
					return err
				}
				g.Screen = LevelScreen
				g.gameOverRunning = false
			} else {
				g.finishPlayerGame()
			}
		case presentation.ContinueDeclined:
			g.finishPlayerGame()
		}
		if g.continueAfterScores && g.director.Phase == presentation.LogoDelay {
			g.continueAfterScores = false
			if g.View.ContinueCredits > 0 {
				g.director.BeginContinue()
			} else {
				g.finishPlayerGame()
			}
		}
		g.presentationInput = presentation.Input{}
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
		return
	}
	g.Screen = TitleScreen
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
	return false
}

func (g *Game) drawPresentation(screen *ebiten.Image) {
	d := g.director
	p := &g.Bundle.Presentation
	if d.LogoScale > 0 {
		index := min(15, d.LogoScale-1)
		g.drawAtlasSprite(screen, g.graphics.logoZoom, p.LogoZoom.Sprites[index].Name, float64(p.LogoZoomX[index]), float64(p.LogoZoomY[index]))
	}
	for _, caption := range d.Captions {
		g.drawCaptionZoom(screen, caption)
	}
	if d.ShowScores {
		for index, score := range d.Scores {
			line := fmt.Sprintf("%2d. %07d %s", index+1, score.Points, score.Initials)
			g.drawGlyphs(screen, g.graphics.font, line, 32, 38+index*16, 16)
		}
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
