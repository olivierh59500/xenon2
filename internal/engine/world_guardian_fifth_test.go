package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func fifthResourceWorld(t *testing.T) *World {
	t.Helper()
	world, err := NewWorld(originalWorldData(t, 5))
	if err != nil {
		t.Fatal(err)
	}
	return world
}

func TestFifthWorldGuardiansUseOriginalPartsAndArena(t *testing.T) {
	w := fifthResourceWorld(t)
	if err := w.activateFifthGuardian(visualassets.FixedEncounter{Y: 2336}, false); err != nil {
		t.Fatal(err)
	}
	for frame := 0; frame < 90; frame++ {
		w.Frame++
		w.advanceFifthGuardian(false)
	}
	if w.FifthMiddle == nil || len(w.fifthMiddleActors) != 10 || w.MaximumScrollY < 2336 {
		t.Fatal("middle guardian did not retain ten source parts")
	}
	foundColumn := false
	for _, actor := range w.Actors {
		if actor.fifthColumn != nil {
			w.advanceFifthColumn(actor)
			foundColumn = actor.DrawKind == "fifth-column" && actor.DrawLength > 0
		}
	}
	if !foundColumn {
		t.Fatal("source growing laser has no visible render descriptor")
	}
	w.damageFifthGuardian(w.fifthMiddleActors[1], 40)
	if !w.FifthMiddle.Parts[1].Destroyed || !w.fifthMiddleActors[1].Active {
		t.Fatal("mount did not retain its noncollidable wreck")
	}
	oldY := w.FifthMiddle.Parts[0].Y
	w.Checkpoint.ScrollY = 4600
	w.RestartCheckpoint()
	if w.FifthMiddle.Parts[0].Y != oldY+(4608-4600) || w.FifthMiddle.Parts[1].Health != 0 {
		t.Fatal("checkpoint restore reset guardian damage or camera-relative position")
	}
	if err := w.activateFifthGuardian(visualassets.FixedEncounter{}, true); err != nil {
		t.Fatal(err)
	}
	if w.ScrollY != 416 || w.Checkpoint.ScrollY != 416 || w.FifthFinal == nil {
		t.Fatal("final arena omitted its original camera/restart boundary")
	}
	for index := 3; index < 21; index++ {
		w.damageFifthGuardian(w.fifthFinalActors[index], 255)
	}
	w.advanceFifthGuardian(true)
	if w.FifthFinal.OuterRemaining != 0 || w.fifthFinalActors[21].Collision.Empty() {
		t.Fatal("core did not open after eighteen defenses fell")
	}
	before := len(w.Collectibles)
	w.damageFifthGuardian(w.fifthFinalActors[21], 20)
	if len(w.Collectibles)-before != 20 || w.PendingExitDrops != 20 {
		t.Fatal("final source reward factory did not produce ten pairs")
	}
	if !w.LevelFinished || !w.FifthFinal.Defeated || w.ExitReady || w.ShopReady {
		t.Fatal("final core did not dispatch stage completion")
	}
	for row := 0; row < 39; row++ {
		for column := 0; column < 20; column++ {
			if w.Level.Terrain.Map[row*20+column] != 0 {
				t.Fatal("final death did not remove its arena tiles")
			}
		}
	}
}
