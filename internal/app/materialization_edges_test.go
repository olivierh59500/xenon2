package app

import (
	"fmt"
	"testing"
)

func TestMaterializationClipsOriginalSpriteAtPlayfieldEdgesGPUOptional(t *testing.T) {
	g, b := renderIntegrationGame(t)
	d := renderFixtureWorld(t, b, 1, -1, 0)
	g.View = d.Frame()
	g.View.PlayerAlive, g.View.BackgroundStars = false, nil
	g.View.PlayerSprite, g.View.Sprites = "", nil
	g.View.TerrainMap, g.View.ActorTerrainMap = make([]uint16, 6000), make([]uint16, 6000)
	g.View.CameraY, g.View.ActorCameraY = 0, 0
	g.rememberFrameHistory()
	base := renderIntegrationPixels(t, g, "materialization-edge-backdrop")
	sprite := SpriteView{Atlas: "ships", Sprite: b.Ships.SteeringFrames[6], Layer: "moving", Materializing: true}
	source, region := integrationSpritePixels(b, sprite, 1)
	for index, position := range [][2]int{{-8, 45}, {308, 45}, {145, -8}, {145, 182}} {
		sprite.X, sprite.Y = float64(position[0]+region.AnchorX), float64(position[1]+region.AnchorY)
		g.View.Sprites = []SpriteView{sprite}
		actual := renderIntegrationPixels(t, g, fmt.Sprintf("materialization-edge-%d", index))
		for y := 0; y < ScreenHeight; y++ {
			for x := 0; x < ScreenWidth; x++ {
				want := base.NRGBAAt(x, y)
				sx, sy := x-position[0], y-position[1]
				if y < PlayfieldHeight && sx >= 0 && sx < region.Width && sy >= 0 && sy < region.Height {
					pixel := source.NRGBAAt(region.X+sx, region.Y+sy)
					if pixel.A != 0 {
						want = pixel
					}
				}
				if actual.NRGBAAt(x, y) != want {
					t.Fatalf("edge%d pixel%d,%d: got%v want%v", index, x, y, actual.NRGBAAt(x, y), want)
				}
			}
		}
	}
}
