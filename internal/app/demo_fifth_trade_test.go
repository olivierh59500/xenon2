package app

import (
	"testing"

	"xenon2/internal/engine"
)

func fifthTradeEquipment() engine.Equipment {
	e := engine.NewEquipment()
	e.ApplyItem(engine.ItemCannon)
	e.Lives, e.Shield = 1, 27
	return e
}

func TestFourthFinalLaserTradeUsesActualSaleRepairAndPurchaseBudget(t *testing.T) {
	e := fifthTradeEquipment()
	money := 5050
	rules := engine.ShopRules{Level: 4, StockLimit: 6000}
	before := e
	if !demoFourthFinalLaserSale(e, money, rules) || e != before || money != 5050 {
		t.Fatal("affordable native trade was rejected or mutated live values")
	}
	refund, err := rules.Sell(&e, &money, engine.SaleMount0)
	if err != nil || refund != 2000 {
		t.Fatalf("native cannon sale: refund%d %v", refund, err)
	}
	if _, err := rules.Buy(&e, &money, engine.ItemHealth1); err != nil {
		t.Fatal(err)
	}
	if _, err := rules.Buy(&e, &money, engine.ItemLaser); err != nil {
		t.Fatal(err)
	}
	if money != 2550 || e.Shield != 39 || e.Lives != 1 || e.Mounts[0].Item != engine.ItemLaser || e.Mounts[1].Item != engine.ItemNone {
		t.Fatalf("native trade changed its earned budget or mount: cash%d equipment%+v", money, e)
	}
}

func TestFourthFinalLaserTradeDeclinesIncompleteBudgets(t *testing.T) {
	for _, q := range []struct {
		shield, money, stock, level int
		want                        bool
	}{
		{27, 2499, 6000, 4, false}, {27, 2500, 6000, 4, true},
		{18, 2999, 6000, 4, false}, {18, 3000, 6000, 4, true},
		{39, 1999, 6000, 4, false}, {39, 2000, 6000, 4, true},
		{27, 5050, 3500, 4, false}, {27, 5050, 6000, 3, false},
	} {
		e := fifthTradeEquipment()
		e.Shield = q.shield
		if got := demoFourthFinalLaserSale(e, q.money, engine.ShopRules{Level: q.level, StockLimit: q.stock}); got != q.want {
			t.Fatalf("HP%d cash%d stock%d level%d: trade%v want%v", q.shield, q.money, q.stock, q.level, got, q.want)
		}
	}
}
