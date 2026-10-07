package engine

// CheatOptions keeps optional trainer behavior separate from normal game rules.
// The zero value preserves the original untrained game.
type CheatOptions struct {
	InfiniteLives, InfiniteCredits bool
	InfiniteMoney, InfiniteEnergy  bool
	KeyFunctions                   bool
}

func (c CheatOptions) Enabled() bool {
	return c.InfiniteLives || c.InfiniteCredits || c.InfiniteMoney || c.InfiniteEnergy || c.KeyFunctions
}

func (w *World) SetCheats(options CheatOptions) { w.Cheats = options }

// SetCheats applies the same selected trainer options to both saved players.
func (s *Session) SetCheats(options CheatOptions) {
	for _, world := range s.Players {
		if world != nil {
			world.SetCheats(options)
		}
	}
}

// PrepareShop preserves the trainer's entry grant rather than making trades
// free. Purchases still subtract their normal prices and obey compatibility.
func (w *World) PrepareShop(rules ShopRules) ShopRules {
	if w.Cheats.InfiniteMoney {
		w.Money = 5000000
		rules.StockLimit = 30000
	}
	return rules
}

// ApplyCheatItem admits a keyboard equipment initializer only while the
// optional key functions are enabled. It never bypasses a guardian or shop.
func (w *World) ApplyCheatItem(item Item) bool {
	if !w.Cheats.KeyFunctions || item <= ItemAdvice || item > ItemBitmapShades {
		return false
	}
	changed := w.Equipment.ApplyItem(item)
	if changed && item == ItemSuperNashwan {
		w.Equipment.BeginSuperLoadout()
	}
	return changed
}
