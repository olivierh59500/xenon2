package shopui

import (
	"strings"
	"testing"

	"xenon2/internal/engine"
	"xenon2/internal/visualassets"
)

func testShop(e *engine.Equipment, money *int) *State {
	catalogue := &visualassets.ShopCatalogue{Items: make([]visualassets.ShopItem, 25)}
	for i := range catalogue.Items {
		catalogue.Items[i] = visualassets.ShopItem{ID: "Item", Lines: []string{"ITEM"}}
	}
	scene := &visualassets.ShopScene{DialogueColumns: 20, DialogueX: 220, DialogueY: 118, DialogueLineStep: 7, Font: visualassets.Font{Width: 4}, Messages: map[string]string{"sell-question": "SELL", "buy-question": "BUY", "sale-price": "\rI WILL PAY ", "buy-price": "\rTHAT WILL COST ", "sale-complete": "SOLD", "another-item": "BOUGHT", "cannot-fit": "CANNOT FIT", "out-of-stock": "OUT OF STOCK"}}
	random := engine.NewRandomState()
	return New(e, money, engine.ShopRules{Level: 1, StockLimit: 10000}, catalogue, scene, func() uint16 { return uint16(random.Next()) })
}

func TestShopQuoteAndSeparateSaleConfirmation(t *testing.T) {
	e := engine.NewEquipment()
	e.ApplyItem(engine.ItemRearShot)
	money := 1234
	s := testShop(&e, &money)
	finishShopDialogue(s)
	if err := s.Confirm(); err != nil {
		t.Fatal(err)
	}
	if money != 1234 || e.Rear.Item != engine.ItemRearShot {
		t.Fatal("quoting sold equipment")
	}
	if !strings.Contains(shopDialogue(s), "IWILLPAY") {
		t.Fatal("original sale quote text missing")
	}
	finishShopDialogue(s)
	s.Row, s.Column = 4, 1
	if err := s.Confirm(); err != nil {
		t.Fatal(err)
	}
	if money != 1734 || e.Rear.Item != engine.ItemNone {
		t.Fatal("confirmed refund not applied to world state")
	}
}

func TestShopExitAndPurchasePreserveWorldEquipment(t *testing.T) {
	e := engine.NewEquipment()
	money := 5000
	s := testShop(&e, &money)
	finishShopDialogue(s)
	s.Row, s.Column = 4, 0
	s.Confirm()
	finishShopDialogue(s)
	if s.Selling || s.Done {
		t.Fatal("sell exit should enter buying")
	}
	s.Row, s.Column = 0, 1
	if err := s.Confirm(); err != nil {
		t.Fatal(err)
	}
	if money != 5000 || e.SpeedTier != 0 {
		t.Fatal("quote applied a purchase")
	}
	finishShopDialogue(s)
	s.Row, s.Column = 4, 1
	s.Confirm()
	if money != 4500 || e.SpeedTier != 1 {
		t.Fatal("purchase did not update world state")
	}
	if shopDialogue(s) != "BOUGHT" {
		t.Fatalf("purchase response %q", shopDialogue(s))
	}
	if s.DisplayMoney != 5000 {
		t.Fatal("purchase display skipped the original countdown")
	}
	for i := 0; i < 5; i++ {
		s.Advance()
	}
	finishShopDialogue(s)
	if s.DisplayMoney != 4500 {
		t.Fatal("cash display did not settle after five passes")
	}
	s.Row, s.Column = 4, 0
	s.Confirm()
	for i := 0; i < 20 && !s.Done; i++ {
		s.Advance()
	}
	if !s.Done || e.SpeedTier != 1 {
		t.Fatal("leaving shop resets equipment")
	}
}

func TestShopNavigationAndOriginalStockPage(t *testing.T) {
	e := engine.NewEquipment()
	money := 10000
	s := testShop(&e, &money)
	finishShopDialogue(s)
	s.Move(-1, 0)
	if s.Column != 4 {
		t.Fatal("column does not wrap")
	}
	s.Move(0, -1)
	if s.Row != 4 || s.Column != 1 {
		t.Fatal("two-button row not clamped")
	}
	s.Column = 0
	s.Confirm()
	finishShopDialogue(s)
	s.Row, s.Column = 3, 4
	if err := s.Confirm(); err != nil {
		t.Fatal(err)
	}
	if s.Offset != 19 || s.Entries[0].Item != engine.ItemBomb {
		t.Fatalf("more cell: offset=%d first=%d", s.Offset, s.Entries[0].Item)
	}
	s.Row, s.Column = 3, 4
	s.Confirm()
	if s.Offset != 0 {
		t.Fatal("last stock page does not wrap")
	}
}

func finishShopDialogue(s *State) {
	for i := 0; i < 1000 && (s.Entrance > 0 || s.Revealed < len(s.Dialogue) || s.HandRemaining > 0); i++ {
		s.Advance()
	}
}

func shopDialogue(s *State) string {
	var b strings.Builder
	for _, glyph := range s.Dialogue {
		b.WriteRune(glyph.Character)
	}
	return b.String()
}

func TestShopTransitionsKeepIndependentOriginalCounters(t *testing.T) {
	e := engine.NewEquipment()
	money := 0
	s := testShop(&e, &money)
	for pass := 0; pass < 12; pass++ {
		s.Advance()
	}
	if s.Headphones != 0 || s.LowerOverlay != 20 || s.Entrance != 5 {
		t.Fatalf("entrance counters %d/%d/%d", s.Headphones, s.LowerOverlay, s.Entrance)
	}
	for pass := 0; pass < 5; pass++ {
		s.Advance()
	}
	if s.LowerOverlay != 0 || s.Entrance != 0 {
		t.Fatal("entrance did not finish after seventeen passes")
	}
	finishShopDialogue(s)
	s.Row, s.Column = 4, 0
	s.Confirm()
	finishShopDialogue(s)
	s.Row, s.Column = 4, 0
	s.Confirm()
	for pass := 0; pass < 16; pass++ {
		s.Advance()
	}
	if s.Done || s.Headphones != 48 || s.LowerOverlay != 64 {
		t.Fatal("exit finished before lower overlay")
	}
	s.Advance()
	if !s.Done || s.LowerOverlay != 68 {
		t.Fatal("exit did not complete after seventeen passes")
	}
}
