package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestShopTransactionsCompatibilityAndRefund(t *testing.T) {
	e := NewEquipment()
	money := 20000
	shop := ShopRules{Level: 1, StockLimit: 6000}
	if price, err := shop.Buy(&e, &money, ItemDoubleShot); err != nil || price != 3000 || e.Primary.Item != ItemDoubleShot {
		t.Fatalf("double-shot purchase: price=%d error=%v equipment=%+v", price, err, e)
	}
	before, oldMoney := e, money
	if _, err := shop.Buy(&e, &money, ItemFlamer); err == nil || e != before || money != oldMoney {
		t.Fatal("incompatible purchases must leave equipment and money unchanged")
	}
	if _, err := shop.Buy(&e, &money, ItemPowerup); err != nil || e.Primary.Tier != 1 {
		t.Fatalf("first power tier: %+v error=%v", e, err)
	}
	refund, err := shop.Sell(&e, &money, SalePrimary)
	if err != nil || refund != 2500 || e.Primary.Item != ItemNone {
		t.Fatalf("sale: refund=%d error=%v primary=%+v", refund, err, e.Primary)
	}
	shop.Leave(&e)
	if e.Primary.Item != ItemForwardShot || e.Primary.Tier != 0 {
		t.Fatalf("shop exit must restore the basic gun: %+v", e.Primary)
	}
	final := ShopRules{Level: 5, StockLimit: 6000}
	if price, err := final.Buy(&e, &money, ItemDoubleShot); err != nil || price != 1500 {
		t.Fatalf("final-level discount: price=%d error=%v", price, err)
	}
	if refund, err := final.Sell(&e, &money, SalePrimary); err != nil || refund != 1500 {
		t.Fatalf("refund uses the undiscounted catalogue: refund=%d error=%v", refund, err)
	}
}

func TestShopPowerSelectionMountCapacityAndStock(t *testing.T) {
	e := NewEquipment()
	e.ApplyItem(ItemDrone)
	e.ApplyItem(ItemLaser)
	e.ApplyItem(ItemPowerup)
	if e.Primary.Tier != 1 || e.Rear.Tier != 0 || e.Mounts[0].Tier != 0 {
		t.Fatalf("lowest-tier ties must choose the primary first: %+v", e)
	}
	e.ApplyItem(ItemPowerup)
	if e.Mounts[0].Tier != 1 || e.Rear.Tier != 0 {
		t.Fatalf("mounts precede the rear slot on the next tie: %+v", e)
	}
	for range 3 {
		e.ApplyItem(ItemCannon)
	}
	if e.CanInstall(ItemLaser) || e.ApplyItem(ItemCannon) {
		t.Fatal("four occupied mounts cannot accept another weapon")
	}
	money := 20000
	before := e
	if _, err := (ShopRules{Level: 1, StockLimit: 600}).Buy(&e, &money, ItemPowerup); err == nil || e != before || money != 20000 {
		t.Fatal("shop stock is limited independently of the wallet")
	}
}

func TestEquipmentTemporaryLoadoutAndSharedClock(t *testing.T) {
	e := NewEquipment()
	e.ApplyItem(ItemHomingMissile)
	if e.FirePeriod != 30 || e.Rear.Item != ItemHomingMissile {
		t.Fatalf("homing installation: %+v", e)
	}
	original := e.WeaponLoadout
	e.ApplyItem(ItemSuperNashwan)
	if e.SuperLoadoutActive || e.WeaponLoadout != original || e.SuperFrames != 170 {
		t.Fatal("Nashwan purchase must preserve shop equipment until departure")
	}
	e.BeginSuperLoadout()
	if e.Primary.Item != ItemDoubleShot || e.Primary.Tier != 2 || e.Mounts[0].Item != ItemCannon || e.Mounts[1].Item != ItemCannon || e.Mounts[2].Item != ItemLaser || e.Mounts[2].Tier != 2 || e.Mounts[3].Item != ItemLaser || e.Rear.Item != ItemDrone || e.Side.Item != ItemSideShot {
		t.Fatalf("temporary suite: %+v", e)
	}
	for range 170 {
		e.AdvanceTimers()
	}
	if e.SuperLoadoutActive || e.WeaponLoadout != original || e.FirePeriod != 8 {
		t.Fatalf("restoration must preserve the shared clock's history: %+v", e)
	}
}

func TestEquipmentPickupAndShopLimitsRemainDistinct(t *testing.T) {
	e := NewEquipment()
	e.ApplyItem(ItemAutofire)
	e.ApplyItem(ItemAutofire)
	if e.FireAdvance != 3 || e.CanInstall(ItemAutofire) {
		t.Fatal("shop autofire cap is three")
	}
	if !e.ApplyItem(ItemAutofire) || e.FireAdvance != 4 || !e.CanInstall(ItemAutofire) {
		t.Fatal("the original pickup cap and shop equality check must remain distinct")
	}
	e.ApplyItem(ItemRearShot)
	e.ApplyItem(ItemSideShot)
	if e.Rear.Item != ItemNone || e.Side.Item != ItemSideShot {
		t.Fatalf("side-shot initializer replaces a rear shot: %+v", e)
	}
	e.ApplyItem(ItemRearShot)
	if e.Rear.Item != ItemRearShot || e.Side.Item != ItemNone {
		t.Fatalf("rear-shot initializer replaces a side shot: %+v", e)
	}
}

func TestShopEligibilityNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	file, err := os.Open(filepath.Join(root, "shop-eligibility-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows[1:] {
		values := make([]int, len(row))
		for i, field := range row {
			values[i], err = strconv.Atoi(field)
			if err != nil {
				t.Fatal(err)
			}
		}
		e := Equipment{}
		for index, slot := range e.slots() {
			start := 3 + index*3
			*slot = WeaponSlot{Item: Item(values[start]), Tier: values[start+1], MaxTier: values[start+2]}
		}
		e.Shield, e.Lives, e.SpeedTier, e.FireAdvance = values[24], values[25], values[26], values[27]
		e.DiveCharges, e.SuperFrames, e.Protection, e.ShadesFrames = values[28], values[29], values[30] != 0, values[31]
		if actual := e.CanInstall(Item(values[1])); actual != (values[2] != 0) {
			t.Fatalf("case %d item %d: allowed=%v, expected %v, equipment=%+v", values[0], values[1], actual, row, e)
		}
	}
	if len(rows)-1 != 1325 {
		t.Fatalf("incomplete shop comparison: %d cases", len(rows)-1)
	}
}

func TestEquipmentInitializersNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	file, err := os.Open(filepath.Join(root, "equipment-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	e := Equipment{}
	for _, row := range rows[1:] {
		values := make([]int, len(row))
		for i, field := range row {
			values[i], err = strconv.Atoi(field)
			if err != nil {
				t.Fatal(err)
			}
		}
		if values[1] == 0 {
			e = NewEquipment()
		}
		switch values[2] {
		case 0:
			e.BeginSuperLoadout()
		case 26:
			e.RestoreSuperLoadout()
		case 27:
			e.AdvanceTimers()
		default:
			e.ApplyItem(Item(values[2]))
		}
		for index, slot := range e.slots() {
			start := 3 + index*3
			if int(slot.Item) != values[start] || slot.Tier != values[start+1] || slot.MaxTier != values[start+2] {
				t.Fatalf("case %d step %d item %d slot %d: got %+v, expected %v", values[0], values[1], values[2], index, slot, row)
			}
		}
		if e.Shield != values[24] || e.Lives != values[25] || e.SpeedTier != values[26] || e.FireAdvance != values[27] || e.DiveCharges != values[28] || e.SuperFrames != values[29] || e.Protection != (values[30] != 0) || e.ShadesFrames != values[31] || e.FirePeriod != values[32] {
			t.Fatalf("case %d step %d item %d: got %+v, expected %v", values[0], values[1], values[2], e, row)
		}
	}
	if len(rows)-1 != 234 {
		t.Fatalf("incomplete initializer comparison: %d steps", len(rows)-1)
	}
}

func TestShopQuotesNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	file, err := os.Open(filepath.Join(root, "shop-quote-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	weapons := map[Item]bool{ItemDoubleShot: true, ItemFlamer: true, ItemCannon: true, ItemLaser: true, ItemMissileLauncher: true, ItemRearShot: true, ItemMineSmall: true, ItemMineLarge: true, ItemElectroBall: true, ItemDrone: true, ItemSideShot: true, ItemBomb: true, ItemHomingMissile: true}
	for _, row := range rows[1:] {
		var values [5]int
		for i := range values {
			values[i], err = strconv.Atoi(row[i])
			if err != nil {
				t.Fatal(err)
			}
		}
		shop, item := ShopRules{Level: values[0]}, Item(values[1])
		if price, err := shop.Price(item); err != nil || price != values[3] {
			t.Fatalf("buy quote %v: price=%d error=%v", row, price, err)
		}
		if weapons[item] {
			e := Equipment{WeaponLoadout: WeaponLoadout{Primary: WeaponSlot{Item: item, Tier: values[2], MaxTier: 2}}}
			if refund, err := shop.QuoteSale(e, SalePrimary); err != nil || refund != values[4] {
				t.Fatalf("sale quote %v: refund=%d error=%v", row, refund, err)
			}
		}
	}
	if len(rows)-1 != 150 {
		t.Fatalf("incomplete quote comparison: %d cases", len(rows)-1)
	}
}
