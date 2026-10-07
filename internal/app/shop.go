package app

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"xenon2/internal/engine"
	"xenon2/internal/shopui"
)

// EnterShop retains the world's current weapons, upgrades and accumulated cash.
func (g *Game) EnterShop(endOfLevel bool) error {
	driver, ok := g.Driver.(*worldDriver)
	if !ok {
		return fmt.Errorf("shop needs an active game world")
	}
	w := driver.world
	stock := w.Level.Terrain.MidShopStockLimit
	if endOfLevel {
		stock = w.Level.Terrain.EndShopStockLimit
	}
	if endOfLevel && w.AdviceIndex < 12 {
		w.AdviceIndex = 12
	}
	g.shop = shopui.New(&w.Equipment, &w.Money, engine.ShopRules{Level: w.Level.Number, StockLimit: stock}, &g.Bundle.Shop, &g.Bundle.ShopScene, w.NextUIRandom)
	g.shop.AdviceIndex = &w.AdviceIndex
	g.Screen = ShopScreen
	g.clock = engine.NewFrameClock(25, 60)
	g.stream.StopMusic()
	g.soundtrack = ""
	g.stream.StopEffects()
	if !g.Config.Mute {
		return g.stream.QueueEffect("shop-synthesized-effect-08", 2)
	}
	return nil
}

func (g *Game) updateShop() error {
	if g.shop == nil {
		return fmt.Errorf("shop state unavailable")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		g.shop.Move(-1, 0)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.shop.Move(1, 0)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.shop.Move(0, -1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.shop.Move(0, 1)
	}
	confirm := inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyControl)
	if !g.shop.Busy() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		matched := false
		for _, cell := range g.Bundle.ShopScene.Cells {
			if x >= cell.X-4 && x < cell.X+36 && y >= cell.Y-4 && y < cell.Y+36 {
				g.shop.Column, g.shop.Row = cell.Column, cell.Row
				matched = true
				break
			}
		}
		if !matched && y >= 168 && y < 196 {
			if x >= 4 && x < 36 {
				g.shop.Column, g.shop.Row = 0, 4
				matched = true
			} else if x >= 45 && x < 77 {
				g.shop.Column, g.shop.Row = 1, 4
				matched = true
			}
		}
		confirm = confirm || matched
	}
	if confirm {
		if err := g.shop.Confirm(); err != nil {
			return err
		}
	}
	for ticks := g.clock.Advance(); ticks > 0; ticks-- {
		g.shop.Advance()
	}
	if g.shop.StopEffectsRequested {
		g.shop.StopEffectsRequested = false
		g.stream.StopEffects()
	}
	for _, id := range g.shop.TakeCues() {
		channel := 2
		if len(id) >= len("shop-sampled") && id[:len("shop-sampled")] == "shop-sampled" {
			channel = 0
		}
		if !g.Config.Mute {
			if err := g.stream.QueueEffect(id, channel); err != nil {
				return err
			}
		}
	}
	if g.shop.Done {
		if driver, ok := g.Driver.(*worldDriver); ok {
			driver.world.AdviceIndex = 12
		}
		g.View = g.Driver.Frame()
		g.rememberFrameHistory()
		g.Screen = LevelScreen
		g.clock = engine.NewFrameClock(25, 60)
		g.selectMusic()
	}
	return nil
}
