package app

import "github.com/hajimehoshi/ebiten/v2"

func (g *Game) drawBackgroundStars(destination *ebiten.Image, level int, alpha float64) {
	if g.View.BackgroundStars == nil {
		return
	}
	pixels := g.graphics.backgroundStarPixels
	clear(pixels)
	palette := g.Bundle.Levels[level-1].Terrain.Palette
	for _, star := range g.View.BackgroundStars.Stars {
		y := int(wrapLerp(float64(star.PreviousY), float64(star.Y), alpha, PlayfieldHeight)) % PlayfieldHeight
		at := (y*ScreenWidth + star.X) * 4
		// Earlier source stars win coincident pixels, just like the four bands.
		if pixels[at+3] != 0 {
			continue
		}
		color := palette[star.Color]
		copy(pixels[at:at+4], color[:])
	}
	g.graphics.backgroundStarPoints.WritePixels(pixels)
	g.graphics.backgroundStarBackdrop.Clear()
	g.graphics.backgroundStarBackdrop.DrawImage(destination, nil)
	op := ebiten.DrawRectShaderOptions{}
	op.Images[0], op.Images[1] = g.graphics.backgroundStarBackdrop, g.graphics.backgroundStarPoints
	destination.DrawRectShader(ScreenWidth, PlayfieldHeight, g.graphics.backgroundStarShader, &op)
}
