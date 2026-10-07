package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestFourthWorldCrawlerNestDeathAndReuseOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 1100, 1600, 1600
	record := visualassets.FixedEncounter{EnemyKind: 3, State1: 3, X: 24, Y: 1200, Variant: 0}
	w.spawnFixed(record)
	var actor *WorldActor
	for _, candidate := range w.Actors {
		if candidate.fourthCrawler != nil {
			actor = candidate
			break
		}
	}
	if actor == nil || len(w.UnimplementedFixedEncounters) != 0 || len(actor.TileOverlays) != 1 {
		t.Fatal("source crawler selector did not create its sprite and tiled nest cover")
	}
	slot, id := actor.Binding.Slot, actor.ID
	actor.fourthCrawler.Phase = 2
	actor.fourthCrawler.X, actor.fourthCrawler.Y = 160, 1260
	actor.X, actor.Y = 160, 160
	w.updateFourthCrawlerCover(actor)
	if len(actor.TileOverlays) != 0 {
		t.Fatal("escaping crawler retained its tiled cover")
	}
	w.damageActor(actor, 3)
	if actor.Health != 7 || !actor.Flash {
		t.Fatal("crawler nonlethal flash/health differs")
	}
	w.damageActor(actor, 7)
	if !actor.Active || actor.Health != 10 || actor.ID != id || actor.Binding.Slot != slot || actor.fourthCrawler.Phase != 0 || actor.fourthCrawler.X != 32 || actor.fourthCrawler.Y != 1220 || w.Score != 200 || len(w.Collectibles) != 1 {
		t.Fatal("crawler destruction must reset the same actor and emit its large coin")
	}
	coin := w.Collectibles[0]
	if coin.Cash != 100 || coin.Motion.X != 160 || coin.Motion.Y != 160 || w.PendingExitDrops != 0 || w.Pool.Slot(coin.Binding.Slot).ResourceTag != 24 {
		t.Fatal("crawler coin must retain source location/tag without becoming an exit reward")
	}
	if len(actor.TileOverlays) != 1 || actor.TileOverlays[0].Patch.Columns != 1 || actor.TileOverlays[0].Patch.Rows != 3 {
		t.Fatal("crawler nest cover did not return after reset")
	}
}
