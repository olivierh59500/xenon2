package app

import "xenon2/internal/engine"

// demoFifthSideSale plans the original sale transactions before fitting flank
// guns. A collected homing missile occupies their slot, while fitting Side Shot
// would discard a Rear Shot without refunding its remaining sale value.
func demoFifthSideSale(equipment engine.Equipment, money int, rules engine.ShopRules) (engine.SalePosition, bool) {
	if rules.Level != 5 || equipment.SuperLoadoutActive || equipment.Mounts[0].Item != engine.ItemLaser || equipment.Side.Item != engine.ItemHomingMissile {
		return 0, false
	}
	position := engine.SaleSide
	probe, wallet := equipment, money
	if equipment.Rear.Item == engine.ItemRearShot {
		position = engine.SaleRear
		if _, err := rules.Sell(&probe, &wallet, engine.SaleRear); err != nil {
			return 0, false
		}
	}
	if _, err := rules.Sell(&probe, &wallet, engine.SaleSide); err != nil {
		return 0, false
	}
	if probe.Shield < 39 {
		repair := demoShopPurchase(probe, wallet, rules)
		if repair != engine.ItemHealth1 && repair != engine.ItemHealth2 {
			return 0, false
		}
		if _, err := rules.Buy(&probe, &wallet, repair); err != nil || probe.Shield != 39 {
			return 0, false
		}
	}
	return position, rules.CanBuy(probe, wallet, engine.ItemSideShot) == nil
}

// The fifth launcher corridor benefits from a piercing left mount. Trade only
// after the fourth guardian is defeated and a normal repair and laser fit in
// the actual sale budget. The shop director executes the quote and purchase.
func demoFourthFinalLaserSale(equipment engine.Equipment, money int, rules engine.ShopRules) bool {
	if rules.Level != 4 || equipment.Mounts[0].Item != engine.ItemCannon || equipment.SuperLoadoutActive {
		return false
	}
	refund, err := rules.QuoteSale(equipment, engine.SaleMount0)
	if err != nil {
		return false
	}
	budget := money + refund
	if equipment.Shield < 39 {
		repair := demoShopPurchase(equipment, budget, rules)
		if repair != engine.ItemHealth1 && repair != engine.ItemHealth2 {
			return false
		}
		price, err := rules.Price(repair)
		if err != nil {
			return false
		}
		budget -= price
	}
	price, err := rules.Price(engine.ItemLaser)
	if err != nil || price > rules.StockLimit || budget < price {
		return false
	}
	probe, wallet := equipment, money
	if _, err := rules.Sell(&probe, &wallet, engine.SaleMount0); err != nil {
		return false
	}
	return rules.CanBuy(probe, budget, engine.ItemLaser) == nil
}

// demoShopPurchase chooses one item without changing equipment or cash. The
// director still has to find it on a shop page, request its quote and confirm.
func demoShopPurchase(equipment engine.Equipment, money int, rules engine.ShopRules) engine.Item {
	affordable := func(item engine.Item) bool { return rules.CanBuy(equipment, money, item) == nil }
	if equipment.Shield < 39 {
		// The original small repair adds 20 and the large repair adds 40. A small
		// repair already reaches the 39-point cap when at least 19 remain.
		if equipment.Shield < 19 && affordable(engine.ItemHealth2) {
			return engine.ItemHealth2
		}
		if affordable(engine.ItemHealth1) {
			return engine.ItemHealth1
		}
	}
	if equipment.Lives < 3 {
		if affordable(engine.ItemExtraLife) {
			return engine.ItemExtraLife
		}
		price, err := rules.Price(engine.ItemExtraLife)
		if err == nil && price <= rules.StockLimit {
			// Keep a low ship count's existing savings for a ship sold by this
			// merchant instead of repeatedly spending them on weapon tiers.
			return engine.ItemNone
		}
	}
	// The fifth final guardian's horizontal armor requires flank firing lanes.
	// Fit the affordable native Side Shot after repairs; the real shop handles
	// its price and replacement of an installed Rear Shot.
	if rules.Level == 5 && equipment.Mounts[0].Item == engine.ItemLaser && equipment.Side.Item == engine.ItemNone && affordable(engine.ItemSideShot) {
		return engine.ItemSideShot
	}
	if affordable(engine.ItemProtection) {
		return engine.ItemProtection
	}
	if rules.Level == 5 && equipment.Side.Item == engine.ItemSideShot && equipment.Mounts[0].Item == engine.ItemLaser && affordable(engine.ItemPowerup) {
		return engine.ItemPowerup
	}
	if equipment.FireAdvance < 3 && affordable(engine.ItemAutofire) {
		return engine.ItemAutofire
	}
	if equipment.SpeedTier < 1 && affordable(engine.ItemSpeedup) {
		return engine.ItemSpeedup
	}
	if equipment.Rear.Item == engine.ItemNone && equipment.Side.Item != engine.ItemSideShot && affordable(engine.ItemRearShot) {
		return engine.ItemRearShot
	}
	hasCannon := false
	for _, slot := range equipment.Mounts {
		hasCannon = hasCannon || slot.Item == engine.ItemCannon
	}
	if !hasCannon && affordable(engine.ItemCannon) {
		return engine.ItemCannon
	}
	// The first power-up selects the basic primary gun before equally powered
	// rear equipment. Later rear tiers are deliberately left for the player.
	if equipment.Primary.Item != engine.ItemNone && equipment.Primary.Tier == 0 && equipment.Primary.MaxTier > 0 && affordable(engine.ItemPowerup) {
		return engine.ItemPowerup
	}
	return engine.ItemNone
}
