package app

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Exercise positive and negative native anchors, nonzero atlas origins,
// transparency and clipping without needing original game resources.
func TestAtlasSpriteAnchorGPUPixels(t *testing.T) {
	if logicOnlyTests {
		t.Skip("sprite pixels require the graphics test runner")
	}
	source := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	red := color.NRGBA{R: 255, A: 255}
	green := color.NRGBA{G: 255, A: 255}
	blue := color.NRGBA{B: 255, A: 255}
	for x := 2; x < 5; x++ {
		source.SetNRGBA(x, 3, red)
		source.SetNRGBA(x, 4, green)
		source.SetNRGBA(x, 5, blue)
	}
	source.SetNRGBA(3, 4, color.NRGBA{})
	texture := ebiten.NewImageFromImage(source)
	defer texture.Dispose()
	region := texture.SubImage(image.Rect(2, 3, 5, 6)).(*ebiten.Image)
	actual := ebiten.NewImage(16, 16)
	defer actual.Dispose()
	pixels := make([]byte, 16*16*4)
	var g Game
	for _, pose := range []struct{ x, y, anchorX, anchorY int }{
		{8, 8, 2, 3}, {8, 8, -2, -1}, {1, 0, 2, 1}, {15, 15, 0, -1},
	} {
		actual.Clear()
		atlas := atlasGraphics{"sprite": {image: region, anchorX: pose.anchorX, anchorY: pose.anchorY, width: 3, height: 3}}
		g.drawAtlasSprite(actual, atlas, "sprite", float64(pose.x), float64(pose.y))
		actual.ReadPixels(pixels)
		for y := range 16 {
			for x := range 16 {
				want := color.NRGBA{}
				sx, sy := x-(pose.x-pose.anchorX), y-(pose.y-pose.anchorY)
				if sx >= 0 && sx < 3 && sy >= 0 && sy < 3 {
					want = source.NRGBAAt(sx+2, sy+3)
				}
				at := (y*16 + x) * 4
				got := color.NRGBA{R: pixels[at], G: pixels[at+1], B: pixels[at+2], A: pixels[at+3]}
				if got != want {
					t.Fatalf("pose %+v at (%d,%d): got %v want %v", pose, x, y, got, want)
				}
			}
		}
	}
}
