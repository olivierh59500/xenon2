package shopui

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/engine"
	"xenon2/internal/visualassets"
)

func TestPrivateOriginalShopTransitions(t *testing.T) {
	dir := os.Getenv("XENON2_SHOP_TEST_DIR")
	if dir == "" {
		t.Skip("local shop transition references not supplied")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "05c400f8.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	catalogue, err := visualassets.DecodeShopCatalogue(raw)
	if err != nil {
		t.Fatal(err)
	}
	common, err := os.ReadFile(filepath.Join(dir, "XenonII.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	art, err := visualassets.DecodeShopArt(raw, common, catalogue)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := visualassets.DecodeShopScene(raw, art.Palette)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(filepath.Dir(dir), "analysis", "shop-sequence-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	modes := map[string]Phase{"portrait-in": PortraitEntrance, "televisions-on": TelevisionsOn, "hand": HeadphoneHand, "televisions-off": TelevisionsOff, "portrait-out": PortraitExit}
	var current string
	var state *State
	for _, row := range rows[1:] {
		if row[0] != current {
			current = row[0]
			e := engine.NewEquipment()
			money := 0
			r := engine.NewRandomState()
			state = New(&e, &money, engine.ShopRules{Level: 1}, catalogue, scene, func() uint16 { return uint16(r.Next()) })
			state.Phase = modes[current]
			state.PhasePass = 0
			if state.Phase != PortraitEntrance {
				state.Headphones, state.LowerOverlay = 0, 0
			}
			if state.Phase == TelevisionsOn {
				state.afterTelevisions = HeadphoneHand
			}
			if state.Phase == TelevisionsOff {
				state.afterTelevisions = PortraitExit
			}
		}
		state.Advance()
		number := func(at int) int {
			v, err := strconv.Atoi(row[at])
			if err != nil {
				t.Fatal(err)
			}
			return v
		}
		switch current {
		case "portrait-in", "portrait-out":
			if state.Headphones != number(2) || state.LowerOverlay != number(3) {
				t.Fatalf("%s pass %s counters Go %d/%d native %s/%s", current, row[1], state.Headphones, state.LowerOverlay, row[2], row[3])
			}
		case "televisions-on", "televisions-off":
			if state.Television[0] != number(4) {
				t.Fatalf("%s pass %s TV Go %d native %s", current, row[1], state.Television[0], row[4])
			}
		case "hand":
			frame := scene.IntroHand[state.IntroHandFrame]
			if frame.X != number(5) || frame.Y != number(6) {
				t.Fatalf("hand pass %s Go %d/%d native %s/%s", row[1], frame.X, frame.Y, row[5], row[6])
			}
			if row[7] == "5cabc" && frame.Sprite != scene.IntroHand[12].Sprite || row[7] == "5c82c" && frame.Sprite != scene.IntroHand[0].Sprite {
				t.Fatalf("hand pass %s changed its source pose", row[1])
			}
		}
	}
	if len(scene.AdviceTips) != 5 {
		t.Fatal("stage advice table incomplete")
	}
	for _, tips := range scene.AdviceTips {
		if len(tips) != 6 {
			t.Fatal("visit advice table incomplete")
		}
	}
	t.Logf("Compared %d original shop transition passes.", len(rows)-1)
}
