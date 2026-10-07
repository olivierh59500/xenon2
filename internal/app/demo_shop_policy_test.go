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
