package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestFirstGuardianWorldCompletionWaitsForExitCash(t *testing.T) {
	w := testWorld(t)
	w.Level.Guardians = &visualassets.Guardians{Visuals: []visualassets.GuardianVisual{{InitialHealth: 30, BodyX: 112, Body: visualassets.TilePatch{Columns: 1, Rows: 1, Tiles: []uint16{0}}}}}
	w.firstGuardianArt = &w.Level.Guardians.Visuals[0]
	state := NewFirstGuardianState(30)
	w.FirstGuardian = &state
	w.firstGuardianActor = &WorldActor{ID: 10, Active: true, ActorList: "moving", firstGuardian: true, part: &visualassets.ActorPart{ResourceTag: 80}}
	w.Actors = append(w.Actors, w.firstGuardianActor)
	for _, name := range []string{"cash-small", "cash-large"} {
		w.commonAnimations[name] = visualassets.NamedActorAnimation{ID: name, Ending: "hold", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: name}}}}
	}
	w.ScrollY = 48
	w.advanceFirstGuardian()
	if !w.FirstGuardian.Active || len(w.Actors) != 9 {
		t.Fatal("guardian activation must create eight articulated pieces")
	}
	w.strikeFirstGuardian(w.FirstGuardian.WeakPoint(w.ScrollY), 30)
	if !w.FirstGuardian.Defeated || w.PendingExitDrops != 18 || w.ExitReady || len(w.Collectibles) != 18 {
		t.Fatal("guardian death must await its eighteen cash rewards")
	}
	for _, coin := range w.Collectibles {
		if coin.Cash == 50 && coin.Order < 0 || coin.Cash == 100 && coin.Order >= 0 {
			t.Fatal("small and large exit cash must retain head/tail insertion")
		}
		coin.Motion.Mode, coin.Motion.Y = 0, 199
		w.advanceCollectible(coin)
	}
	if w.PendingExitDrops != 0 || !w.ExitReady {
		t.Fatal("expiring all exit coins must release the stage exit")
	}
}
