package app

import (
	"image/color"
	"math"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"xenon2/internal/controls"
)

// MobileGame adds control panels around an unchanged original game canvas.
// Construction waits until Update, after the Android view has its context.
type MobileGame struct {
	config   Config
	game     *Game
	scene    *ebiten.Image
	width    int
	pad      controls.Pad
	touches  []ebiten.TouchID
	contacts []controls.Touch
	filtered []controls.Touch
	gestures controls.GestureGuard
}

type mobileGestureInsets struct{ left, top, right, bottom float64 }

var androidGestureInsets atomic.Pointer[mobileGestureInsets]

// SetMobileGestureInsets receives Android's system gesture areas without
// accessing game state from the platform UI thread.
func SetMobileGestureInsets(left, top, right, bottom, width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	androidGestureInsets.Store(&mobileGestureInsets{float64(left) / float64(width), float64(top) / float64(height), float64(right) / float64(width), float64(bottom) / float64(height)})
}

func (g *MobileGame) gestureInsets() controls.GestureInsets {
	if areas := androidGestureInsets.Load(); areas != nil {
		return controls.GestureInsets{Left: max(12, areas.left*float64(g.width)), Top: max(8, areas.top*controls.Height), Right: max(12, areas.right*float64(g.width)), Bottom: max(24, areas.bottom*controls.Height)}
	}
	// The minimum leaves every virtual control accessible and reserves the
	// original HUD's bottom strip even while immersive bars report zero insets.
	return controls.GestureInsets{Left: 12, Top: 8, Right: 12, Bottom: 24}
}

func NewMobileGame(config Config) *MobileGame {
	g := &MobileGame{config: config, width: 1344}
	g.pad.Place(controls.NewLayout(g.width))
	return g
}

func (g *MobileGame) Update() error {
	if g.game == nil {
		var err error
		g.game, err = NewGameFromConfig(g.config)
		if err != nil {
			return err
		}
		if err = g.game.EnableAudio(); err != nil {
			return err
		}
		g.scene = ebiten.NewImage(ScreenWidth, ScreenHeight)
	}
	g.contacts = g.contacts[:0]
	g.touches = ebiten.AppendTouchIDs(g.touches[:0])
	for _, id := range g.touches {
		x, y := ebiten.TouchPosition(id)
		g.contacts = append(g.contacts, controls.Touch{ID: int(id), X: float64(x), Y: float64(y), Pressed: inpututil.TouchPressDuration(id) == 1})
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		g.contacts = append(g.contacts, controls.Touch{ID: -1, X: float64(x), Y: float64(y), Pressed: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)})
	}
	i := sampleMobileInput()
	// The wide host's mouse coordinates must pass through the same transform
	// as touchscreen taps before reaching the original menus and shop.
	g.filtered = g.gestures.Filter(g.filtered[:0], g.contacts, float64(g.width), controls.Height, g.gestureInsets())
	i = mergeMobileInput(i, g.pad.Update(g.filtered))
	if i.escape && g.game.Screen == TitleScreen {
		// The desktop ESC exit is not an Android game-update error.
		i.escape = false
		g.game.BeginAttract()
	}
	return g.game.advanceWithInput(i)
}

func (g *MobileGame) Draw(dst *ebiten.Image) {
	dst.Fill(color.RGBA{7, 13, 22, 255})
	if g.game == nil {
		return
	}
	g.scene.Clear()
	g.game.Draw(g.scene)
	op := ebiten.DrawImageOptions{}
	op.GeoM.Scale(3, 3)
	op.GeoM.Translate(g.pad.Layout.SceneX, 0)
	dst.DrawImage(g.scene, &op)
	g.drawTouchControls(dst)
}

func (g *MobileGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.width = controls.LogicalWidth(outsideWidth, outsideHeight)
	g.pad.Place(controls.NewLayout(g.width))
	return g.width, controls.Height
}

func (g *MobileGame) drawTouchControls(dst *ebiten.Image) {
	j := &g.pad.Joystick
	x, y, r := float32(j.CenterX), float32(j.CenterY), float32(j.Radius)
	vector.FillCircle(dst, x, y, r, color.RGBA{22, 36, 54, 255}, true)
	vector.StrokeCircle(dst, x, y, r, 2, color.RGBA{100, 145, 173, 255}, true)
	vector.StrokeCircle(dst, x, y, float32(j.Travel), 1, color.RGBA{43, 69, 91, 255}, true)
	directions := [8][2]int{{1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}}
	for sector, d := range directions {
		a := float64(sector) * math.Pi / 4
		ux, uy := float32(math.Cos(a)), float32(math.Sin(a))
		px, py := x+ux*r*.82, y+uy*r*.82
		c := color.RGBA{89, 125, 151, 255}
		if j.X == d[0] && j.Y == d[1] {
			c = color.RGBA{154, 224, 250, 255}
		}
		vector.StrokeLine(dst, px-ux*5-uy*4, py-uy*5+ux*4, px, py, 2, c, true)
		vector.StrokeLine(dst, px, py, px-ux*5+uy*4, py-uy*5-ux*4, 2, c, true)
	}
	kx, ky := x+float32(j.OffsetX), y+float32(j.OffsetY)
	c := color.RGBA{47, 75, 99, 255}
	if j.Active() {
		c = color.RGBA{64, 117, 151, 255}
	}
	vector.FillCircle(dst, kx, ky, r*.32, c, true)
	vector.StrokeCircle(dst, kx, ky, r*.32, 2, color.RGBA{166, 209, 229, 255}, true)
	for button, b := range g.pad.Layout.Buttons {
		r := b.Bounds
		c := color.RGBA{24, 42, 61, 255}
		stroke := color.RGBA{87, 124, 154, 255}
		if g.pad.Frame.Held[button] {
			c = color.RGBA{40, 102, 138, 255}
			stroke = color.RGBA{155, 224, 250, 255}
		}
		if b.Round {
			radius := float32(r.Width / 2)
			vector.FillCircle(dst, float32(r.X)+radius, float32(r.Y)+radius, radius, c, true)
			vector.StrokeCircle(dst, float32(r.X)+radius, float32(r.Y)+radius, radius, 2, stroke, true)
		} else {
			vector.FillRect(dst, float32(r.X), float32(r.Y), float32(r.Width), float32(r.Height), c, true)
			vector.StrokeRect(dst, float32(r.X), float32(r.Y), float32(r.Width), float32(r.Height), 2, stroke, true)
		}
		g.drawMobileLabel(dst, b.Label, r.X+(r.Width-float64(len(b.Label))*12)/2, r.Y+(r.Height-12)/2)
	}
}

func (g *MobileGame) drawMobileLabel(dst *ebiten.Image, text string, x, y float64) {
	for _, character := range text {
		if glyph := g.game.graphics.presentationFont[character]; glyph != nil {
			op := ebiten.DrawImageOptions{}
			op.GeoM.Scale(.75, .75)
			op.GeoM.Translate(x, y)
			dst.DrawImage(glyph, &op)
		}
		x += 12
	}
}
