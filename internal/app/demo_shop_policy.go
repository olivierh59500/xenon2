package app

import "xenon2/internal/engine"

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
	if affordable(engine.ItemProtection) {
		return engine.ItemProtection
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
