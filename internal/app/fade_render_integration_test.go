package app

import (
	"image/color"
	"strings"
	"testing"
	"xenon2/internal/engine"
	"xenon2/internal/presentation"
	"xenon2/internal/shopui"
)

func finishRenderFade(t *testing.T, g *Game) {
	t.Helper()
	for pass := 0; pass < 120; pass++ {
		if g.fade == nil || g.fade.Done {
			return
		}
		advanceFrontend(t, g, inputFrame{})
	}
	t.Fatal("bounded palette fade did not finish")
}

func TestCompletedFadeUncoversHeaderAndEndingDotOptional(t *testing.T) {
	g, b := renderIntegrationGame(t)
	g.Config.Mute = true
	g.paused = false
	g.startFade(presentation.NewPaletteFadeOut(0), func() error { g.beginHeader(b.Presentation.EnteringShopHeading, headerNone); return nil })
	finishRenderFade(t, g)
	if g.fade != nil {
		t.Fatal("completed outgoing palette still covers the new header")
	}
	for pass := 0; pass < 120; pass++ {
		advanceFrontend(t, g, inputFrame{})
		if len(g.director.Captions) > 0 && g.director.Captions[0].Scale == 16 && g.director.Captions[0].CenterY == 12 {
			break
		}
	}
	header := renderIntegrationPixels(t, g, "completed-fade-entering-shop-header")
	visible := 0
	for y := 0; y < 26; y++ {
		for x := 0; x < ScreenWidth; x++ {
			c := header.NRGBAAt(x, y)
			if c.R != 0 || c.G != 0 || c.B != 0 {
				visible++
			}
		}
	}
	if visible < 100 {
		t.Fatalf("entering-shop caption remained black: %d nonblack pixels", visible)
	}

	w := g.Driver.(*worldDriver).world
	g.shop = shopui.New(&w.Equipment, &w.Money, engine.ShopRules{Level: 5, StockLimit: 6000}, &b.Shop, &b.ShopScene, w.NextUIRandom)
	g.Screen = ShopScreen
	g.shop.Phase = shopui.EndingFade
	g.startFade(presentation.NewPaletteFadeOut(0), func() error { g.shop.BeginEndingDot(); return nil })
	finishRenderFade(t, g)
	dot := renderIntegrationPixels(t, g, "completed-fade-ending-dot-visible")
	pattern := [3][4]uint8{{5, 7, 7, 5}, {7, 7, 7, 7}, {5, 7, 7, 5}}
	for y, row := range pattern {
		for x, index := range row {
			c := b.ShopScene.EndingPalette[index]
			want := color.NRGBA{R: c[0], G: c[1], B: c[2], A: c[3]}
			if got := dot.NRGBAAt(172+x, 100+y); got != want {
				t.Fatalf("ending dot pixel%d,%d got%v want%v", x, y, got, want)
			}
		}
	}
	if c := dot.NRGBAAt(171, 100); c.R != 0 || c.G != 0 || c.B != 0 {
		t.Fatal("ending dot leaked beyond its original four-column pattern")
	}
}

func TestCompletedBlackFadeRemainsUntilExplicitScreenChangeOptional(t *testing.T) {
	g, _ := renderIntegrationGame(t)
	g.Config.Mute = true
	g.paused = false
	g.startFade(presentation.NewPaletteFadeOut(0), nil)
	finishRenderFade(t, g)
	if g.fade == nil || !g.fade.Done || g.fade.Deduction != 8 {
		t.Fatal("terminal fade-out lost its intentionally black palette")
	}
	black := renderIntegrationPixels(t, g, "completed-terminal-fade-remains-black")
	for y := 0; y < ScreenHeight; y++ {
		for x := 0; x < ScreenWidth; x++ {
			c := black.NRGBAAt(x, y)
			if c.R != 0 || c.G != 0 || c.B != 0 {
				t.Fatal("finished terminal fade uncovered its old scene")
			}
		}
	}
	g.BeginAttract()
	if g.fade != nil || g.whenFadeEnds != nil {
		t.Fatal("explicit attract admission retained its old palette fade")
	}
	for range 100 {
		advanceFrontend(t, g, inputFrame{})
	}
	visible := renderIntegrationPixels(t, g, "attract-after-terminal-black-fade")
	found := false
	for y := 0; y < ScreenHeight; y++ {
		for x := 0; x < ScreenWidth; x++ {
			c := visible.NRGBAAt(x, y)
			found = found || c.R != 0 || c.G != 0 || c.B != 0
		}
	}
	if !found {
		t.Fatal("attract stayed black after explicit scene admission")
	}
}

func TestFadeCallbackMayInstallItsNextFadeOptional(t *testing.T) {
	g, _ := renderIntegrationGame(t)
	g.Config.Mute = true
	called := false
	g.startFade(presentation.NewPaletteFadeOut(0), func() error { called = true; g.startFade(presentation.NewPaletteFadeIn(), nil); return nil })
	for pass := 0; pass < 30; pass++ {
		advanceFrontend(t, g, inputFrame{})
		if called && g.fade != nil && !g.fade.Done && g.fade.Deduction == 7 {
			return
		}
	}
	t.Fatal("outgoing cleanup erased the fade installed by its callback")
}

type firstDrawPresentationDriver struct {
	frame SceneFrame
	next  SceneFrame
}

func (d *firstDrawPresentationDriver) Advance(Input) error { d.frame = d.next; return nil }
func (d *firstDrawPresentationDriver) Frame() SceneFrame   { return d.frame }

func assertNoFallbackLogo(t *testing.T, g *Game, name string) {
	t.Helper()
	pixels := renderIntegrationPixels(t, g, name)
	logo := g.Bundle.Title.Image
	matches := 0
	for y := 0; y < logo.Bounds().Dy(); y++ {
		for x := 0; x < logo.Bounds().Dx(); x++ {
			want := logo.NRGBAAt(x, y)
			if want.A != 0 && (want.R != 0 || want.G != 0 || want.B != 0) && pixels.NRGBAAt(g.Bundle.Title.X+x, g.Bundle.Title.Y+y) == want {
				matches++
			}
		}
	}
	if matches > 16 {
		t.Fatalf("first director draw retained%d pixels from the unrelated fallback logo", matches)
	}
}

func TestFirstDrawAdmitsPlayerTwoReadyAndFinalLossOptional(t *testing.T) {
	for _, ready := range []bool{true, false} {
		g, _ := renderIntegrationGame(t)
		g.Config.Mute = true
		g.paused = false
		g.clock = engine.NewFrameClock(60, 60)
		frame := g.View
		frame.Ready, frame.GameOver = false, false
		frame.PlayerNumber = 2
		next := frame
		next.Ready = ready
		next.GameOver = !ready
		next.Score = 0
		next.ContinueCredits = 0
		d := &firstDrawPresentationDriver{frame: frame, next: next}
		g.SetDriver(d)
		advanceFrontend(t, g, inputFrame{})
		if g.Screen != PresentationScreen {
			t.Fatal("updated world state waited another display update before its director")
		}
		if ready {
			if g.director.Phase != presentation.ReadyMessage || !strings.Contains(g.director.Captions[0].Text, "2") {
				t.Fatal("player2 turn flashed the player1 READY fallback")
			}
			assertNoFallbackLogo(t, g, "first-draw-player-two-ready")
		} else {
			if g.director.Phase != presentation.GameOverMessage {
				t.Fatal("final loss flashed the fallback before GAME OVER")
			}
			assertNoFallbackLogo(t, g, "first-draw-final-loss-director")
		}
	}
}
