package app

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

func (g *Game) openCheatMenu() {
	if g.Screen == CheatScreen {
		return
	}
	g.cheatReturn = g.Screen
	g.Screen = CheatScreen
	g.Config.Demo = false
	g.demo = nil
}
func (g *Game) closeCheatMenu() {
	g.applyCheatOptions()
	g.Screen = g.cheatReturn
}
func (g *Game) applyCheatOptions() {
	if d, ok := g.Driver.(*worldDriver); ok {
		if d.session != nil {
			for _, w := range d.session.Players {
				if w != nil {
					w.SetCheats(g.Config.Cheats)
				}
			}
		} else {
			d.world.SetCheats(g.Config.Cheats)
		}
	}
}
func (g *Game) updateCheatMenu(i inputFrame) {
	if g.cheatHelp {
		if i.confirm || i.mousePressed {
			g.cheatHelp = false
		}
		return
	}
	if i.upPressed {
		g.cheatRow = (g.cheatRow + 7) % 8
	}
	if i.downPressed {
		g.cheatRow = (g.cheatRow + 1) % 8
	}
	if i.mousePressed && i.mouseX >= 12 && i.mouseX < 308 && i.mouseY >= 42 && i.mouseY < 182 {
		g.cheatRow = (i.mouseY - 42) / 18
		i.confirm = true
	}
	if !(i.confirm || i.leftPressed || i.rightPressed) {
		return
	}
	switch g.cheatRow {
	case 0:
		g.Config.Cheats.InfiniteLives = !g.Config.Cheats.InfiniteLives
	case 1:
		g.Config.Cheats.InfiniteCredits = !g.Config.Cheats.InfiniteCredits
	case 2:
		g.Config.Cheats.InfiniteMoney = !g.Config.Cheats.InfiniteMoney
	case 3:
		g.Config.Cheats.InfiniteEnergy = !g.Config.Cheats.InfiniteEnergy
	case 4:
		g.Config.Cheats.KeyFunctions = !g.Config.Cheats.KeyFunctions
	case 5:
		step := 1
		if i.leftPressed {
			step = -1
		}
		g.Config.Level = (max(1, g.Config.Level)-1+step+5)%5 + 1
	case 6:
		g.cheatHelp = !g.cheatHelp
	case 7:
		g.closeCheatMenu()
	}
	g.applyCheatOptions()
}
func (g *Game) drawCheatMenu(screen *ebiten.Image) {
	screen.Fill(color.Black)
	if g.cheatHelp {
		g.drawGlyphs(screen, g.graphics.presentationFont, "    CHEAT KEYS      ", 0, 4, 16)
		lines := []string{"F1 SPEED   F2 REPAIR  F4 AUTOFIRE", "F5 NASHWAN F6 HEALTH F7 REAR", "F8 MINE    F9 SIDE   F10 SHIP", "PAD 1 BALL   PAD 2 POWER UP", "PAD 3 MINE   PAD 4 DOUBLE SHOT", "PAD 5 CANNON PAD 6 DIVE", "PAD 7 MISSILE PAD 8 LASER", "PAD 9 DRONE   PAD 0 FLAMER", "DEL ENERGY ON  INS ENERGY OFF", "ENTER OR ESC TO RETURN"}
		for row, line := range lines {
			g.drawCheatText(screen, line, 16, 38+row*15, 8)
		}
		return
	}
	g.drawGlyphs(screen, g.graphics.presentationFont, "       CHEATS       ", 0, 4, 16)
	options := []struct {
		name    string
		enabled bool
	}{
		{"INFINITE LIVES", g.Config.Cheats.InfiniteLives},
		{"INFINITE CREDITS", g.Config.Cheats.InfiniteCredits},
		{"INFINITE MONEY", g.Config.Cheats.InfiniteMoney},
		{"INFINITE ENERGY", g.Config.Cheats.InfiniteEnergy},
		{"KEY FUNCTIONS", g.Config.Cheats.KeyFunctions},
	}
	for row, option := range options {
		status := "OFF"
		if option.enabled {
			status = "ON"
		}
		prefix := " "
		if g.cheatRow == row {
			prefix = ">"
		}
		g.drawCheatText(screen, fmt.Sprintf("%s%-19s %s", prefix, option.name, status), 16, 42+row*18, 8)
	}
	prefix := " "
	if g.cheatRow == 5 {
		prefix = ">"
	}
	g.drawCheatText(screen, fmt.Sprintf("%sSTARTING LEVEL %d", prefix, max(1, g.Config.Level)), 16, 132, 8)
	prefix = " "
	if g.cheatRow == 6 {
		prefix = ">"
	}
	g.drawCheatText(screen, prefix+"KEY HELP", 16, 150, 8)
	prefix = " "
	if g.cheatRow == 7 {
		prefix = ">"
	}
	g.drawCheatText(screen, prefix+"BACK", 16, 168, 8)
	g.drawCheatText(screen, "F3 OR ESC TO RETURN", 16, 188, 8)
}

// drawCheatText scales the native 16-pixel typeface for the additional settings.
// Original game scenes keep their own unscaled fonts and spacing.
func (g *Game) drawCheatText(screen *ebiten.Image, text string, x, y, advance int) {
	for _, character := range text {
		if glyph := g.graphics.presentationFont[character]; glyph != nil {
			op := ebiten.DrawImageOptions{}
			op.GeoM.Scale(.5, .5)
			op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(glyph, &op)
		}
		x += advance
	}
}
