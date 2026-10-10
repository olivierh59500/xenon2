package app

import (
	"testing"

	"xenon2/internal/engine"
)

func TestDemoShopChoosesCheapestSufficientOriginalRepair(t *testing.T) {
	for _, fixture := range []struct {
		shield, money, stock int
		want                 engine.Item
		remaining            int
	}{
		{19, 3000, 3000, engine.ItemHealth1, 2500},
		{25, 3000, 3000, engine.ItemHealth1, 2500},
		{18, 3000, 3000, engine.ItemHealth2, 2000},
		{7, 500, 600, engine.ItemHealth1, 0},
	} {
		e := engine.NewEquipment()
		e.Shield = fixture.shield
		rules := engine.ShopRules{Level: 1, StockLimit: fixture.stock}
		item := demoShopPurchase(e, fixture.money, rules)
		if item != fixture.want {
			t.Fatalf("shield%d wallet%d: item%d, want%d", fixture.shield, fixture.money, item, fixture.want)
		}
		money := fixture.money
		if _, err := rules.Buy(&e, &money, item); err != nil {
			t.Fatal(err)
		}
		if money != fixture.remaining {
			t.Fatalf("repair changed original price: wallet%d want%d", money, fixture.remaining)
		}
		wantShield := min(39, fixture.shield+20)
		if item == engine.ItemHealth2 {
			wantShield = min(39, fixture.shield+40)
		}
		if e.Shield != wantShield {
			t.Fatalf("repair changed original shield amount: got%d want%d", e.Shield, wantShield)
		}
	}
}

func TestDemoShopShipReserveRespectsActualStockAndWallet(t *testing.T) {
	e := engine.NewEquipment()
	e.Lives, e.FireAdvance, e.SpeedTier = 2, 3, 1
	for _, fixture := range []struct {
		level, stock, money int
		want                engine.Item
	}{
		{1, 3000, 10000, engine.ItemRearShot},
		{2, 1200, 10000, engine.ItemRearShot},
		{4, 6000, 6000, engine.ItemExtraLife},
		{4, 6000, 5500, engine.ItemNone},
		{5, 6000, 3000, engine.ItemExtraLife},
	} {
		rules := engine.ShopRules{Level: fixture.level, StockLimit: fixture.stock}
		before := e
		if item := demoShopPurchase(e, fixture.money, rules); item != fixture.want {
			t.Fatalf("level%d stock%d wallet%d: item%d want%d", fixture.level, fixture.stock, fixture.money, item, fixture.want)
		}
		if e != before {
			t.Fatal("policy bypassed the original merchant transaction")
		}
	}
}

func TestDemoShopCapsRepeatedTiersAndKeepsWeaponVariety(t *testing.T) {
	e, money := engine.NewEquipment(), 10000
	rules := engine.ShopRules{Level: 2, StockLimit: 4000}
	want := []engine.Item{engine.ItemAutofire, engine.ItemAutofire, engine.ItemSpeedup, engine.ItemRearShot, engine.ItemCannon, engine.ItemPowerup}
	for purchase, expected := range want {
		item := demoShopPurchase(e, money, rules)
		if item != expected {
			t.Fatalf("purchase%d chose%d, want%d", purchase, item, expected)
		}
		if _, err := rules.Buy(&e, &money, item); err != nil {
			t.Fatal(err)
		}
	}
	if next := demoShopPurchase(e, money, rules); next != engine.ItemNone {
		t.Fatalf("policy continues upgrading after its diverse loadout: %d", next)
	}
	if money != 1500 || e.FireAdvance != 3 || e.SpeedTier != 1 || e.Rear.Item != engine.ItemRearShot || e.Rear.Tier != 0 || e.Primary.Tier != 1 || e.Mounts[0].Item != engine.ItemCannon || e.Mounts[1].Item != engine.ItemNone {
		t.Fatalf("original transactions did not retain expected budget/loadout: cash%d equipment%+v", money, e)
	}
}

func TestDemoShopPreservesInstalledSideShotAndPickupAutofire(t *testing.T) {
	e := engine.NewEquipment()
	e.ApplyItem(engine.ItemSideShot)
	e.FireAdvance, e.SpeedTier, e.Primary.Tier = 4, 1, 1
	if item := demoShopPurchase(e, 3000, engine.ShopRules{Level: 1, StockLimit: 3000}); item != engine.ItemNone {
		t.Fatalf("policy would replace a side weapon or rebuy an already stronger firing upgrade: %d", item)
	}
}

func TestFifthShopUsesRealRepairAndSideGunBudget(t *testing.T) {
	e := engine.NewEquipment()
	for _, item := range []engine.Item{engine.ItemPowerup, engine.ItemLaser, engine.ItemRearShot, engine.ItemRearShot, engine.ItemRearShot, engine.ItemAutofire, engine.ItemAutofire, engine.ItemSpeedup, engine.ItemSpeedup} {
		if !e.ApplyItem(item) {
			t.Fatal("cannot initialize explicit fifth loadout")
		}
	}
	e.Shield = 3
	money := 1050
	rules := engine.ShopRules{Level: 5, StockLimit: 6000}
	if item := demoShopPurchase(e, money, rules); item != engine.ItemHealth2 {
		t.Fatal("fifth policy omitted necessary full repair")
	}
	if price, err := rules.Buy(&e, &money, engine.ItemHealth2); err != nil || price != 500 || e.Shield != 39 || money != 550 {
		t.Fatal("native fifth repair transaction changed")
	}
	before := e
	if item := demoShopPurchase(e, money, rules); item != engine.ItemSideShot || e != before || money != 550 {
		t.Fatal("affordable fifth flank gun was rejected or policy changed live equipment")
	}
	if price, err := rules.Buy(&e, &money, engine.ItemSideShot); err != nil || price != 500 || money != 50 || e.Side.Item != engine.ItemSideShot || e.Side.Tier != 0 || e.Rear.Item != engine.ItemNone || e.Mounts[0].Item != engine.ItemLaser {
		t.Fatal("native side transaction lost price or replacement semantics")
	}
	if item := demoShopPurchase(e, money, rules); item != engine.ItemNone {
		t.Fatalf("fifth policy spent an unavailable budget on%d", item)
	}
}

func TestFifthShopTradesCollectedHomingAndRearThroughOriginalTransactions(t *testing.T) {
	e := engine.NewEquipment()
	for _, item := range []engine.Item{engine.ItemLaser, engine.ItemRearShot, engine.ItemHomingMissile} {
		if !e.ApplyItem(item) {
			t.Fatal("cannot initialize collected fifth equipment")
		}
	}
	e.Primary.Tier, e.Mounts[0].Tier, e.Rear.Tier = 2, 1, 2
	e.Shield, e.FireAdvance, e.SpeedTier = 3, 3, 2
	money := 1050
	rules := engine.ShopRules{Level: 5, StockLimit: 6000}
	for _, fixture := range []struct {
		position engine.SalePosition
		refund   int
	}{
		{engine.SaleRear, 2500},
		{engine.SaleSide, 3000},
	} {
		before, beforeMoney := e, money
		position, ok := demoFifthSideSale(e, money, rules)
		if !ok || position != fixture.position || e != before || money != beforeMoney {
			t.Fatal("fifth sale plan changed live inventory or chose the wrong occupied slot")
		}
		if refund, err := rules.Sell(&e, &money, position); err != nil || refund != fixture.refund {
			t.Fatalf("original fifth refund differs: %d %v", refund, err)
		}
	}
	if _, ok := demoFifthSideSale(e, money, rules); ok || money != 6550 {
		t.Fatal("sale plan repeats a sold weapon or loses its refund")
	}
	for _, item := range []engine.Item{engine.ItemHealth2, engine.ItemSideShot, engine.ItemProtection, engine.ItemPowerup, engine.ItemPowerup} {
		before, beforeMoney := e, money
		if next := demoShopPurchase(e, money, rules); next != item || e != before || money != beforeMoney {
			t.Fatalf("fifth purchase plan chose%d, want%d, or changed live state", next, item)
		}
		if _, err := rules.Buy(&e, &money, item); err != nil {
			t.Fatal(err)
		}
	}
	if money != 550 || e.Shield != 39 || e.Primary.Tier != 2 || e.Mounts[0].Item != engine.ItemLaser || e.Mounts[0].Tier != 2 || e.Side.Item != engine.ItemSideShot || e.Side.Tier != 1 || e.Rear.Item != engine.ItemNone || !e.Protection || e.FirePeriod != 8 || e.Lives != 3 || demoShopPurchase(e, money, rules) != engine.ItemNone {
		t.Fatalf("earned fifth transaction budget/loadout differs: cash%d equipment%+v", money, e)
	}
}

func TestFifthSideSaleRejectsForeignOrUnfittableInventories(t *testing.T) {
	e := engine.NewEquipment()
	e.ApplyItem(engine.ItemLaser)
	e.ApplyItem(engine.ItemRearShot)
	e.ApplyItem(engine.ItemHomingMissile)
	for _, fixture := range []struct {
		name  string
		edit  func(*engine.Equipment)
		rules engine.ShopRules
	}{
		{"earlier shop", func(*engine.Equipment) {}, engine.ShopRules{Level: 4, StockLimit: 6000}},
		{"insufficient stock", func(*engine.Equipment) {}, engine.ShopRules{Level: 5, StockLimit: 100}},
		{"temporary suite", func(e *engine.Equipment) { e.SuperLoadoutActive = true }, engine.ShopRules{Level: 5, StockLimit: 6000}},
		{"different mount", func(e *engine.Equipment) { e.Mounts[0].Item = engine.ItemCannon }, engine.ShopRules{Level: 5, StockLimit: 6000}},
		{"existing side gun", func(e *engine.Equipment) { e.Side.Item = engine.ItemSideShot }, engine.ShopRules{Level: 5, StockLimit: 6000}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			probe := e
			fixture.edit(&probe)
			before := probe
			if _, ok := demoFifthSideSale(probe, 1050, fixture.rules); ok || probe != before {
				t.Fatal("fifth sale accepted an unrelated inventory or altered its input")
			}
		})
	}
}
