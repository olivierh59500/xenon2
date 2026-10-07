package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func fourthCrawlerTestArt(t *testing.T) (visualassets.FixedSpriteKind, visualassets.SpriteAtlas) {
	t.Helper()
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR for crawler source comparisons")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "031F0159.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(data)
	if err != nil {
		t.Fatal(err)
	}
	bank, err := visualassets.DecodeFixedSprites(4, data, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range bank.Kinds {
		if kind.Kind == 3 {
			return kind, bank.Atlas
		}
	}
	t.Fatal("crawler kind was not exported")
	return visualassets.FixedSpriteKind{}, visualassets.SpriteAtlas{}
}

func TestFourthCrawlerNativeTraceOptional(t *testing.T) {
	art, atlas := fourthCrawlerTestArt(t)
	boxes := map[string]visualassets.CollisionBox{}
	for _, region := range atlas.Sprites {
		boxes[region.Name] = *region.Collision
	}
	var state FourthCrawlerState
	var random RandomState
	cases := 0
	nativeCombatRows(t, "fourth-crawler-trace.csv", func(v []int64) {
		if v[1] == 0 {
			var err error
			state, err = NewFourthCrawler(visualassets.FixedEncounter{X: int(v[2]), Y: int(v[3]), State1: int(v[4]), Variant: int(v[0])}, art)
			if err != nil {
				t.Fatal(err)
			}
			random = NewRandomState()
		}
		_, err := state.Advance(art, FourthCrawlerInput{ScrollY: int(v[5]), MaximumScrollY: int(v[6]), PlayerY: int(v[7])}, func(name string) visualassets.CollisionBox { return boxes[name] }, random.Next)
		if err != nil {
			t.Fatal(err)
		}
		expected := atlas.SourceSpriteNames[int(v[13])]
		if state.X != int(v[8]) || state.Y != int(v[9]) || state.Phase != int(v[10]) || state.VerticalChoice != int(v[11]) || state.Health != uint16(v[12]) || state.Sprite(art) != expected || v[14] >= 0 && state.Animation.Remaining != int(v[14]) || state.Collision.Left != int(v[15]) || state.Collision.Top != int(v[16]) || state.Collision.Right != int(v[17]) || state.Collision.Bottom != int(v[18]) || state.Removed != (v[19] != 0) || random.A != uint32(v[20]) || random.B != uint32(v[21]) {
			t.Fatalf("crawler %v: state=%+v sprite=%s expected=%s random=%+v", v, state, state.Sprite(art), expected, random)
		}
		cases++
	})
	if cases != 1600 {
		t.Fatalf("crawler proof has %d updates", cases)
	}
}

func TestFourthCrawlerDamageNativeTraceOptional(t *testing.T) {
	art, atlas := fourthCrawlerTestArt(t)
	regions := map[string]visualassets.SpriteRegion{}
	for _, region := range atlas.Sprites {
		regions[region.Name] = region
	}
	var state FourthCrawlerState
	var random RandomState
	score := 0
	nativeCombatRows(t, "fourth-crawler-damage-trace.csv", func(v []int64) {
		if v[1] == 0 {
			var err error
			state, err = NewFourthCrawler(visualassets.FixedEncounter{X: 24, Y: 1200, State1: 3, Variant: int(v[0])}, art)
			if v[0] == 1 {
				state, err = NewFourthCrawler(visualassets.FixedEncounter{X: 296, Y: 1200, State1: 3, Variant: 1}, art)
			}
			if err != nil {
				t.Fatal(err)
			}
			random = NewRandomState()
			score = 0
		}
		state.X, state.Y, state.Phase = int(v[3]), int(v[4]), 2
		image := regions[atlas.SourceSpriteNames[int(v[5])]]
		event := state.Strike(art, uint16(v[2]), 1100, image, random.Next)
		score += event.Score
		if state.Health != uint16(v[6]) || state.Phase != int(v[7]) || state.X != int(v[8]) || state.Y != int(v[9]) || score != int(v[10]) || boolCount(event.Destroyed) != int(v[11]) || event.Destroyed && (event.ExplosionX != int(v[12]) || event.ExplosionY != int(v[13]) || event.CashX != int(v[15]) || event.CashY != int(v[16]) || event.CashDirection != uint8(v[17])) || random.A != uint32(v[18]) || random.B != uint32(v[19]) {
			t.Fatalf("crawler damage %v: state=%+v event=%+v score=%d random=%+v", v, state, event, score, random)
		}
	})
}
