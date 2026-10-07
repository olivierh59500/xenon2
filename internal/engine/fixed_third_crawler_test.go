package engine

import (
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestThirdCrawlerNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "02020113.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(data)
	if err != nil {
		t.Fatal(err)
	}
	bank, err := visualassets.DecodeFixedSprites(3, data, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	art := bank.Third.Crawler
	var state ThirdCrawlerState
	var random RandomState
	passes := 0
	nativeCombatRows(t, "fixed-third-crawler-trace.csv", func(v []int64) {
		if v[1] == 0 {
			state = NewThirdCrawler(visualassets.FixedEncounter{X: 128, Y: 2850, State2: int(v[0])}, 2700, art)
			random = NewRandomState()
		}
		event := state.Advance(ThirdCrawlerInput{ScrollY: 2700, MaximumScrollY: 4607, PlayerX: 160, PlayerY: 120, Columns: terrain.Columns, Map: terrain.Map}, art, &random)
		if state.X != int(v[2]) || state.Y != int(v[3]) || state.Direction != int(v[4]) || state.FireAccumulator != uint8(v[5]) || state.Animation.Remaining != int(v[6]) || state.Animation.Sprite(art.Animations[state.Direction]) != bank.Atlas.SourceSpriteNames[int(v[7])] || random.A != uint32(v[8]) || random.B != uint32(v[9]) || event.Shot != (v[10] != 0) {
			t.Fatalf("crawler dir%d frame%d: %+v event%+v random%+v native%v", v[0], v[1], state, event, random, v)
		}
		if event.Shot && (event.X != int(v[11]) || event.Y != int(v[12]) || event.Speed != int(v[13]) || event.Direction != uint8(v[14])) {
			t.Fatalf("crawler shot differs: %+v native%v", event, v)
		}
		passes++
	})
	if passes != 3200 {
		t.Fatalf("incomplete crawler proof: %d passes", passes)
	}
	t.Log("Compared all8 initial directions across3200 original crawler passes")
}
