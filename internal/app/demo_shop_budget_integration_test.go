package app

import (
	"testing"

	"xenon2/internal/engine"
	"xenon2/internal/shopui"
)

func finishDemoMerchantFixture(t *testing.T, g *Game) []engine.Item {
	t.Helper()
	g.Config.Demo = true
	var quoted []engine.Item
	last := engine.ItemNone
	for update := 0; update < 12000; update++ {
		advanceFrontend(t, g, inputFrame{})
		if g.shop != nil && g.shop.Phase == shopui.Buying && g.shop.QuoteValid {
			item := g.shop.Entries[g.shop.Quoted].Item
			if item != last {
				quoted = append(quoted, item)
				last = item
			}
		}
		if g.Screen == LevelScreen && !g.backdropOnly && (g.fade == nil || g.fade.Done) {
			return quoted
		}
	}
	t.Fatal("bounded merchant controls did not return to gameplay")
	return nil
}

func TestDemoShopPolicyKeepsSmallRepairSavingsAndBaseRearTier(t *testing.T) {
	g := menuAdmittedGame(t, 1, 1)
	w := g.Driver.(*worldDriver).world
	// A merchant fixture arranges the inventory and actual wallet only. The
	// automated player must buy through the original quote/confirmation UI.
	w.Money = 3000
	w.Equipment.ApplyItem(engine.ItemRearShot)
	w.Equipment.Shield, w.Equipment.FireAdvance, w.Equipment.SpeedTier = 19, 3, 1
	// Use end-merchant stock as an explicit shop fixture without claiming a
	// guardian victory; return through the ordinary same-stage reload.
	if err := g.EnterShop(false); err != nil {
		t.Fatal(err)
	}
	g.shop.Rules.StockLimit = w.Level.Terrain.EndShopStockLimit
	quoted := finishDemoMerchantFixture(t, g)
	current := g.Driver.(*worldDriver).world
	if len(quoted) != 2 || quoted[0] != engine.ItemHealth1 || quoted[1] != engine.ItemPowerup {
		t.Fatalf("merchant quotes did not preserve repair/budget priorities: %v", quoted)
	}
	if current.Money != 500 || current.Equipment.Shield != 39 || current.Equipment.Rear.Tier != 0 || current.Equipment.Primary.Tier != 1 {
		t.Fatalf("real merchant trades lost repair savings or selected repeated rear upgrades: cash%d equipment%+v", current.Money, current.Equipment)
	}
}

func TestDemoShopPolicyVisitsShipPageBeforeAffordableLuxury(t *testing.T) {
	g := menuAdmittedGame(t, 4, 1)
	w := g.Driver.(*worldDriver).world
	w.Money, w.Equipment.Lives = 6000, 2
	// Use end-merchant stock as an explicit shop fixture without claiming a
	// guardian victory; return through the ordinary same-stage reload.
	if err := g.EnterShop(false); err != nil {
		t.Fatal(err)
	}
	g.shop.Rules.StockLimit = w.Level.Terrain.EndShopStockLimit
	quoted := finishDemoMerchantFixture(t, g)
	current := g.Driver.(*worldDriver).world
	if len(quoted) != 1 || quoted[0] != engine.ItemExtraLife {
		t.Fatalf("cheap first-page items displaced the affordable ship: %v", quoted)
	}
	if current.Money != 0 || current.Equipment.Lives != 3 || current.Equipment.FireAdvance != 1 || current.Equipment.Rear.Item != engine.ItemNone {
		t.Fatalf("real ship purchase changed its original price or granted unrelated upgrades: cash%d equipment%+v", current.Money, current.Equipment)
	}
}
