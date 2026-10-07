package engine

import "fmt"

// Item identifies an equipment catalogue entry, independently of its artwork.
type Item uint8

const (
	ItemNone Item = iota
	ItemForwardShot
	ItemAdvice
	ItemSpeedup
	ItemHealth1
	ItemAutofire
	ItemSuperNashwan
	ItemHealth2
	ItemRearShot
	ItemMineSmall
	ItemSideShot
	ItemElectroBall
	ItemPowerup
	ItemMineLarge
	ItemDoubleShot
	ItemCannon
	ItemDive
	ItemMissileLauncher
	ItemLaser
	ItemDrone
	ItemFlamer
	ItemBomb
	ItemExtraLife
	ItemHomingMissile
	ItemProtection
	ItemBitmapShades
)

var itemPrices = [...]int{0, 0, 200, 500, 500, 500, 600, 1000, 1000, 1000, 1000, 1200, 2000, 3000, 3000, 4000, 4000, 4000, 4000, 4500, 5000, 5500, 6000, 6000, 6000, 6000}

// WeaponSlot retains power independently of the shared firing clock.
type WeaponSlot struct {
	Item    Item
	Tier    int
	MaxTier int
	Serial  uint32
}

// WeaponLoadout matches the seven physical equipment positions. Four additional
// mounts can carry separate cannons, lasers or missile launchers.
type WeaponLoadout struct {
	Primary WeaponSlot
	Mounts  [4]WeaponSlot
	Rear    WeaponSlot
	Side    WeaponSlot
}

// Equipment contains persistent upgrades and the temporary Nashwan loadout.
// FirePeriod is history-dependent: removing a weapon does not recalculate it.
type Equipment struct {
	WeaponLoadout
	Shield, Lives, SpeedTier int
	FirePeriod, FireAdvance  int
	DiveCharges, SuperFrames int
	Protection               bool
	ShadesFrames             int
	NextWeaponSerial         uint32
	SuperLoadoutActive       bool
	SavedLoadout             WeaponLoadout
}

// NewEquipment starts a game with three ships, a full shield and the basic gun.
func NewEquipment() Equipment {
	e := Equipment{Shield: 39, Lives: 3, FirePeriod: 8, FireAdvance: 1}
	e.install(&e.Primary, ItemForwardShot, 0, 2)
	return e
}

func (e *Equipment) slots() [7]*WeaponSlot {
	return [7]*WeaponSlot{&e.Primary, &e.Mounts[0], &e.Mounts[1], &e.Mounts[2], &e.Mounts[3], &e.Rear, &e.Side}
}

func (e *Equipment) install(slot *WeaponSlot, item Item, tier, maximum int) {
	e.NextWeaponSerial++
	*slot = WeaponSlot{Item: item, Tier: tier, MaxTier: maximum, Serial: e.NextWeaponSerial}
}

func upgrade(slot *WeaponSlot) bool {
	if slot.Tier < slot.MaxTier {
		slot.Tier++
		return true
	}
	return false
}

// CanInstall reproduces shop compatibility independently of price and stock.
// Pickup installation is separate because some pickups can replace equipment
// that the shop refuses to replace before it has been sold.
func (e Equipment) CanInstall(item Item) bool {
	switch item {
	case ItemAdvice:
		return true
	case ItemSpeedup:
		return e.SpeedTier != 2
	case ItemHealth1, ItemHealth2:
		return e.Shield != 39
	case ItemAutofire:
		return e.FireAdvance != 3
	case ItemSuperNashwan:
		return e.SuperFrames == 0
	case ItemPowerup:
		copy := e
		for _, slot := range copy.slots() {
			if slot.Item != ItemNone && slot.Tier != slot.MaxTier {
				return true
			}
		}
		return false
	case ItemDive:
		return e.DiveCharges == 0
	case ItemExtraLife:
		return e.Lives != 9
	case ItemProtection:
		return !e.Protection
	case ItemBitmapShades:
		return e.ShadesFrames == 0
	case ItemForwardShot, ItemDoubleShot, ItemFlamer:
		if e.Primary.Item == item {
			return e.Primary.Tier != e.Primary.MaxTier
		}
		return e.Primary.Item == ItemNone || e.Primary.Item == ItemForwardShot
	case ItemCannon, ItemLaser, ItemMissileLauncher:
		for _, slot := range e.Mounts {
			if slot.Item == ItemNone {
				return true
			}
		}
		return false
	case ItemRearShot, ItemMineSmall, ItemMineLarge, ItemElectroBall, ItemDrone:
		return e.Rear.Item == ItemNone || e.Rear.Item == item && e.Rear.Tier != e.Rear.MaxTier
	case ItemSideShot, ItemBomb:
		return e.Side.Item == ItemNone || e.Side.Item == item && e.Side.Tier != e.Side.MaxTier
	case ItemHomingMissile:
		return e.Rear.Item == ItemNone || e.Side.Item == ItemNone
	default:
		return false
	}
}

// ApplyItem applies the original pickup or purchase initializer. It returns
// whether anything changed; a valid initializer may intentionally do nothing.
func (e *Equipment) ApplyItem(item Item) bool {
	switch item {
	case ItemForwardShot:
		e.install(&e.Primary, item, 0, 2)
		e.FirePeriod = 8
	case ItemSpeedup:
		if e.SpeedTier == 2 {
			return false
		}
		e.SpeedTier++
	case ItemHealth1, ItemHealth2:
		amount := 20
		if item == ItemHealth2 {
			amount = 40
		}
		e.Shield += amount
		if e.Shield > 39 {
			e.Shield = 39
		}
	case ItemAutofire:
		if e.FireAdvance >= 4 {
			return false
		}
		e.FireAdvance++
	case ItemSuperNashwan:
		e.SuperFrames = 170
	case ItemPowerup:
		var selected *WeaponSlot
		for _, slot := range e.slots() {
			if slot.Item == ItemNone || slot.Tier == slot.MaxTier || slot.Tier >= 127 {
				continue
			}
			if selected == nil || slot.Tier < selected.Tier {
				selected = slot
			}
		}
		if selected == nil {
			return false
		}
		selected.Tier++
	case ItemDoubleShot, ItemFlamer:
		if e.Primary.Item == item {
			return upgrade(&e.Primary)
		}
		e.install(&e.Primary, item, 0, 2)
		e.FirePeriod = 8
	case ItemRearShot:
		if e.Rear.Item == item {
			return upgrade(&e.Rear)
		}
		if e.Side.Item == ItemSideShot {
			e.Side = WeaponSlot{}
		}
		e.install(&e.Rear, item, 0, 2)
	case ItemSideShot:
		if e.Side.Item == item {
			return upgrade(&e.Side)
		}
		if e.Side.Item != ItemNone {
			return false
		}
		if e.Rear.Item == ItemRearShot {
			e.Rear = WeaponSlot{}
		}
		e.install(&e.Side, item, 0, 2)
		e.FirePeriod = 8
	case ItemMineSmall, ItemMineLarge:
		if e.Rear.Item == ItemMineSmall || e.Rear.Item == ItemMineLarge {
			return upgrade(&e.Rear)
		}
		tier := 0
		if item == ItemMineLarge {
			tier = 1
		}
		e.install(&e.Rear, item, tier, 1)
	case ItemDrone:
		if e.Rear.Item == item {
			return upgrade(&e.Rear)
		}
		e.install(&e.Rear, item, 0, 2)
	case ItemElectroBall:
		if e.Rear.Item == item {
			return upgrade(&e.Rear)
		}
		e.install(&e.Rear, item, 0, 0)
	case ItemBomb:
		if e.Side.Item == item {
			return upgrade(&e.Side)
		}
		e.install(&e.Side, item, 0, 0)
		e.FirePeriod = 12
	case ItemHomingMissile:
		if e.Rear.Item == item {
			return upgrade(&e.Rear)
		}
		if e.Side.Item == item {
			return upgrade(&e.Side)
		}
		slot := &e.Rear
		if e.Rear.Item != ItemNone {
			slot = &e.Side
		}
		e.install(slot, item, 0, 0)
		e.FirePeriod = 30
	case ItemCannon, ItemLaser, ItemMissileLauncher:
		for i := range e.Mounts {
			if e.Mounts[i].Item == ItemNone {
				maximum := 0
				if item == ItemLaser {
					maximum = 2
				}
				e.install(&e.Mounts[i], item, 0, maximum)
				e.EnsureBasicWeapon()
				return true
			}
		}
		return false
	case ItemDive:
		e.DiveCharges += 3
	case ItemExtraLife:
		e.Lives++
	case ItemProtection:
		e.Protection = true
	case ItemBitmapShades:
		e.ShadesFrames = 220
	default:
		return false
	}
	return true
}

// EnsureBasicWeapon is called when leaving the shop, not while selling guns.
func (e *Equipment) EnsureBasicWeapon() {
	if e.Primary.Item == ItemNone {
		e.ApplyItem(ItemForwardShot)
	}
}

// BeginSuperLoadout installs the temporary suite after leaving the shop. The
// earlier purchase only starts its timer, so intervening shop trades affect
// the loadout that will eventually be restored.
func (e *Equipment) BeginSuperLoadout() {
	if e.SuperFrames == 0 || e.SuperLoadoutActive {
		return
	}
	e.SavedLoadout = e.WeaponLoadout
	e.WeaponLoadout = WeaponLoadout{}
	e.ApplyItem(ItemForwardShot)
	e.ApplyItem(ItemCannon)
	e.ApplyItem(ItemCannon)
	e.ApplyItem(ItemLaser)
	e.Mounts[2].Tier = 2
	e.ApplyItem(ItemLaser)
	e.Mounts[3].Tier = 2
	e.ApplyItem(ItemDrone)
	e.Rear.Tier = 2
	e.ApplyItem(ItemSideShot)
	e.Side.Tier = 2
	e.ApplyItem(ItemDoubleShot)
	e.Primary.Tier = 2
	e.SuperLoadoutActive = true
}

// AdvanceTimers applies one original gameplay pass to equipment durations.
// The shared firing period remains unchanged when the prior guns are restored.
func (e *Equipment) AdvanceTimers() {
	if e.ShadesFrames > 0 {
		e.ShadesFrames--
	}
	if e.SuperFrames > 0 {
		e.SuperFrames--
		if e.SuperFrames == 0 && e.SuperLoadoutActive {
			e.RestoreSuperLoadout()
		}
	}
}

// RestoreSuperLoadout restores the prior guns without changing the remaining
// duration. Level transitions can then install the suite again after restart.
func (e *Equipment) RestoreSuperLoadout() {
	if e.SuperLoadoutActive {
		e.WeaponLoadout = e.SavedLoadout
		e.SavedLoadout = WeaponLoadout{}
		e.SuperLoadoutActive = false
	}
}

// ShopRules supplies the level and the maximum catalogue price for this visit.
// StockLimit is separate from the player's available money.
type ShopRules struct {
	Level      int
	StockLimit int
}

func (s ShopRules) Price(item Item) (int, error) {
	if item < ItemForwardShot || int(item) >= len(itemPrices) {
		return 0, fmt.Errorf("unknown equipment item %d", item)
	}
	price := itemPrices[item]
	if s.Level == 5 {
		price /= 2
	}
	return price, nil
}

func (s ShopRules) CanBuy(e Equipment, money int, item Item) error {
	price, err := s.Price(item)
	if err != nil {
		return err
	}
	if price > s.StockLimit {
		return fmt.Errorf("that item is out of stock")
	}
	if price > money {
		return fmt.Errorf("not enough money")
	}
	if !e.CanInstall(item) {
		return fmt.Errorf("that item cannot be fitted")
	}
	return nil
}

// Buy validates the complete transaction before changing either state.
func (s ShopRules) Buy(e *Equipment, money *int, item Item) (int, error) {
	if e == nil || money == nil {
		return 0, fmt.Errorf("purchase needs equipment and money")
	}
	if err := s.CanBuy(*e, *money, item); err != nil {
		return 0, err
	}
	price, _ := s.Price(item)
	if item != ItemForwardShot {
		e.ApplyItem(item)
	}
	*money -= price
	return price, nil
}

// SalePosition identifies the seven physical equipment positions.
type SalePosition uint8

const (
	SalePrimary SalePosition = iota
	SaleMount0
	SaleMount1
	SaleMount2
	SaleMount3
	SaleRear
	SaleSide
)

// QuoteSale returns the unchanged equipment's sale price. Only installed
// weapons appear in the original sale menu; the basic gun is omitted.
func (s ShopRules) QuoteSale(e Equipment, position SalePosition) (int, error) {
	if position > SaleSide {
		return 0, fmt.Errorf("invalid sale position")
	}
	slot := e.slots()[position]
	switch slot.Item {
	case ItemDoubleShot, ItemFlamer, ItemCannon, ItemLaser, ItemMissileLauncher,
		ItemRearShot, ItemMineSmall, ItemMineLarge, ItemElectroBall, ItemDrone,
		ItemSideShot, ItemBomb, ItemHomingMissile:
	default:
		return 0, fmt.Errorf("that weapon cannot be sold")
	}
	if slot.Tier < 0 || slot.Tier > slot.MaxTier {
		return 0, fmt.Errorf("invalid weapon power tier")
	}
	return (itemPrices[slot.Item] + slot.Tier*2000) / 2, nil
}

// Sell returns half the base price plus one thousand per purchased power tier.
// The final level's buying discount does not change the original refund rule.
func (s ShopRules) Sell(e *Equipment, money *int, position SalePosition) (int, error) {
	if e == nil || money == nil {
		return 0, fmt.Errorf("invalid sale position")
	}
	refund, err := s.QuoteSale(*e, position)
	if err != nil {
		return 0, err
	}
	*e.slots()[position] = WeaponSlot{}
	*money += refund
	return refund, nil
}

// Leave applies the equipment initialization that follows a shop visit.
func (s ShopRules) Leave(e *Equipment) {
	e.EnsureBasicWeapon()
	e.BeginSuperLoadout()
}
