package engine

import (
	"strconv"
	"testing"

	"xenon2/internal/visualassets"
)

// The whole-pixel crawler at 0x556f6/0x562b0 preserves fractional coordinates
// but publishes its direction word at 0x2a and emitter byte at 0x5e.
func TestThirdCrawlerRetainsFractionsAndPublishesPhysicalStateOptional(t *testing.T) {
	for direction := range 8 {
		t.Run(strconv.Itoa(direction), func(t *testing.T) {
			w, err := NewWorld(originalWorldData(t, 3))
			if err != nil {
				t.Fatal(err)
			}
			w.ScrollY = 2700
			slot := w.Pool.FreeFirst()
			w.Pool.Slot(slot).Residue = secondResidueFixture()
			w.spawnThirdFixed(visualassets.FixedEncounter{EnemyKind: 6, State2: direction, X: 128, Y: 2850})
			actor := w.Actors[0]
			if actor.Binding.Slot != slot || actor.thirdCrawler == nil {
				t.Fatal("original crawler constructor did not reclaim its physical slot")
			}
			want := secondResidueFixture()
			want.X, want.Y = int16(actor.X), int16(actor.Y)
			want.Counter, want.Direction, want.EmitterClock = 0, int16(direction), 0
			want.Health, want.PowerOrScore, want.WaveBonusToken = uint16(actor.Health), 100, 0
			want.StrongHealth = false
			if got := actor.Binding.Residue; got != want {
				t.Fatalf("crawler birth physical state: got %+v want %+v", got, want)
			}
			for range 32 {
				w.advanceThirdCrawler(actor)
				w.finishActorUpdate(actor)
				want.X, want.Y = int16(actor.X), int16(actor.Y)
				want.Direction = int16(actor.thirdCrawler.Direction)
				want.SetFireState(actor.thirdCrawler.FireAccumulator, 0)
				if got := w.Pool.Slot(slot).Residue; got != want {
					t.Fatalf("crawler live physical state: got %+v want %+v", got, want)
				}
			}
			actor.thirdCrawler.Y = 1000
			w.MaximumScrollY = w.ScrollY
			w.advanceThirdCrawler(actor)
			w.finishActorUpdate(actor)
			if actor.Active || w.Pool.Slot(slot).ResourceTag != 4 {
				t.Fatal("crawler did not expire through the source off-screen boundary")
			}
			w.releaseDeadPoolEntries(ActorPoolMoving)
			if w.Pool.Slot(slot).allocated || w.Pool.FreeFirst() != slot {
				t.Fatal("expired crawler did not release its physical slot")
			}
			if residue := w.Pool.Slot(slot).Residue; residue.XFraction != want.XFraction || residue.YFraction != want.YFraction {
				t.Fatal("whole-pixel crawler erased inherited fractions before reuse")
			}
		})
	}
}
