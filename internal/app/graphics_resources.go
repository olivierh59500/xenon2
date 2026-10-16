package app

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

func (g *graphics) newImage(width, height int) *ebiten.Image {
	texture := ebiten.NewImage(width, height)
	g.images = append(g.images, texture)
	return texture
}

func (g *graphics) newImageFromImage(source image.Image) *ebiten.Image {
	texture := ebiten.NewImageFromImage(source)
	g.images = append(g.images, texture)
	return texture
}

// Subimages share their parent's allocation, so retain and dispose the parent
// textures explicitly instead of relying on GC or disposing individual views.
func (g *graphics) dispose() {
	for _, texture := range g.images {
		texture.Dispose()
	}
	g.images = nil
	for _, shader := range []*ebiten.Shader{g.paletteShader, g.sparkShader, g.materialShader, g.backgroundStarShader, g.fadeShader} {
		if shader != nil {
			shader.Deallocate()
		}
	}
	g.paletteShader, g.sparkShader, g.materialShader = nil, nil, nil
	g.backgroundStarShader, g.fadeShader = nil, nil
}
