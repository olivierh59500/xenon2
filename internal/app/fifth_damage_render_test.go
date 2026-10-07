package app

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"xenon2/internal/engine"
)

// This arranges an exposed final weak point to isolate damage drawing. The hit
// still comes from the ordinary basic gun and the production world traversal.
func TestFifthCoreShotPublishesOwnerFlashGPUPixelsOptional(t *testing.T) {
	g, b := renderIntegrationGame(t)
	d := renderFixtureWorld(t, b, 5, 6, -1)
	w := d.world
	if err := w.Step(engine.Input{}); err != nil {
		t.Fatal(err)
	}
	if w.FifthFinal == nil {
		t.Fatal("original final selector did not admit its guardian")
	}
	w.ScrollY, w.ScrollDelta = 216, 0
	w.FifthFinal.Parts[0].Y = -120
	w.FifthFinal.OuterRemaining = 0
	for index := 3; index < 21; index++ {
		w.FifthFinal.Parts[index].Destroyed = true
	}
	w.Player = engine.PlayerMotionState{X: 163, Y: 169}
	w.PreviousPlayer = w.Player
	w.MaterializationFrames = 0
	if err := w.Step(engine.Input{Fire: true}); err != nil {
		t.Fatal(err)
	}
	if w.FifthFinal.CoreHealth != 19 {
		t.Fatalf("ordinary shot missed the exposed core: health %d", w.FifthFinal.CoreHealth)
	}
	frame := d.Frame()
	var body SpriteView
	for _, sprite := range frame.Sprites {
		if sprite.Kind == "tiles" && sprite.Flash {
			body = sprite
			break
		}
	}
	if body.Patch.Rows == 0 || body.X != 48 || body.Y != -120 {
		t.Fatal("core hit did not publish its normally terrain-only owner")
	}
	actual := ebiten.NewImage(ScreenWidth, PlayfieldHeight)
	defer actual.Dispose()
	g.drawSprite(actual, body, 5, 1)
	pixels := make([]byte, ScreenWidth*PlayfieldHeight*4)
	actual.ReadPixels(pixels)
	palette := b.Levels[4].Terrain.Palette[15]
	want := color.NRGBA{R: palette[0], G: palette[1], B: palette[2], A: 255}
	covered := 0
	for row := 0; row < body.Patch.Rows; row++ {
		for column := 0; column < body.Patch.Columns; column++ {
			id := body.Patch.Tiles[row*body.Patch.Columns+column]
			var sourceX, sourceY int
			found := false
			for _, tile := range b.Levels[4].Terrain.Tiles {
				if tile.ID == id {
					sourceX, sourceY, found = tile.X, tile.Y, true
					break
				}
			}
			if !found {
				continue
			}
			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					px, py := int(body.X)+column*16+x, int(body.Y)+row*16+y
					if py < 0 || py >= PlayfieldHeight || px < 0 || px >= ScreenWidth || b.Levels[4].Terrain.Atlas.NRGBAAt(sourceX+x, sourceY+y).A == 0 {
						continue
					}
					index := (py*ScreenWidth + px) * 4
					got := color.NRGBA{pixels[index], pixels[index+1], pixels[index+2], pixels[index+3]}
					if got != want {
						t.Fatalf("owner flash pixel (%d,%d): got %v want %v", px, py, got, want)
					}
					covered++
				}
			}
		}
	}
	if covered < 1000 {
		t.Fatalf("owner flash comparison covered only %d pixels", covered)
	}
	if err := w.Step(engine.Input{}); err != nil {
		t.Fatal(err)
	}
	for _, sprite := range d.Frame().Sprites {
		if sprite.ID == body.ID && sprite.Kind == "tiles" && sprite.Flash {
			t.Fatal("one-pass core flash remained on the following frame")
		}
	}
}
