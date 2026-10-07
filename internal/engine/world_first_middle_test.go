package engine

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestFirstMiddleWorldSpawnsFiveStreamsAndOpensShopOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 3000, 3344, 3344
	w.Player.ScrollStep = 1
	w.cursor = RestartEncounterCursor(w.ScrollY)
	if err := w.advanceFirstMiddleStage(); err != nil {
		t.Fatal(err)
	}
	followers := 0
	for _, actor := range w.Actors {
		if actor.firstMiddleFollower != nil {
			followers++
		}
	}
	if followers != 55 || w.ShopReady {
		t.Fatal("arena did not construct all five original eleven-part streams")
	}
	for pass := range 50 {
		if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
			t.Fatalf("pass%d:%v", pass, err)
		}
		if err := w.advanceFirstMiddleStage(); err != nil {
			t.Fatal(err)
		}
		w.compactActors()
	}
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
	w.ScrollY = 2500
	w.Player.Y = 176
	if err := w.advanceFirstMiddleStage(); err != nil {
		t.Fatal(err)
	}
	if !w.ShopReady || w.ScrollY != 2496 || w.MaximumScrollY != 2496 || w.LevelFinished || w.ExitReady {
		t.Fatal("middle crossing must request its shop without completing the stage")
	}
	if w.Level.Terrain.Map == nil || len(w.Level.GuardianParts.Sprites) == 0 {
		t.Fatal("arena lacks mutable map or source images")
	}
}

func TestFirstMiddleStationaryArenaDoesNotLoseEveryStreamEachPassOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 3000, 3344, 3344
	w.cursor = RestartEncounterCursor(3000)
	w.InvulnerableFrames = 10000
	total := 0
	for pass := range 10000 {
		w.Player.X, w.Player.Y = 160, 176
		w.ScrollY, w.MaximumScrollY = 3000, 3344
		before := w.nextActorID
		if err := w.Step(Input{Fire: true}); err != nil {
			t.Fatal(err)
		}
		total += w.nextActorID - before
		if pass > 1 && w.nextActorID-before > 60 {
			counts := map[int]int{}
			for _, slot := range w.Pool.slots {
				if slot.allocated {
					counts[int(slot.ResourceTag)]++
				}
			}
			t.Fatalf("arena replaced almost every chain at once on pass%d: slots=%v actors=%d", pass, counts, len(w.Actors))
		}
	}
	t.Logf("Total constructions: %d live actors: %d", total, len(w.Actors))
}

func TestFirstMiddleDestroyedFollowerBecomesFragmentWithoutBreakingChainOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY = 3000, 3344
	if err := w.spawnFirstMiddleStream(0, 0); err != nil {
		t.Fatal(err)
	}
	var target *WorldActor
	for _, actor := range w.Actors {
		if actor.firstMiddleFollower != nil && actor.Score == 250 {
			target = actor
			break
		}
	}
	if target == nil {
		t.Fatal("source stream lacks its head")
	}
	w.damageActor(target, 1)
	if target.Active || w.Score != 250 || w.Actors[0].firstMiddleFragment == nil || w.Actors[0].Sprite == "" {
		t.Fatal("head death did not preserve its named fragment and source score")
	}
	if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
	remaining := 0
	for _, actor := range w.Actors {
		if actor.Active && actor.firstMiddleFollower != nil {
			remaining++
		}
	}
	if remaining != 10 {
		t.Fatal("removing the head incorrectly removed its ten surviving chain members")
	}
}

func TestFirstMiddleWorldNativeChainOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to local source comparisons")
	}
	data := originalWorldData(t, 1)
	file, err := os.Open(filepath.Join(root, "first-middle-parts-long-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	var w *World
	var chain []*WorldActor
	previousLaunch, previousPass := -1, -1
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		v := make([]int64, len(row))
		for i := range v {
			v[i], err = strconv.ParseInt(row[i], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		launch, pass, part := int(v[0]), int(v[1]), int(v[2])
		if launch != previousLaunch {
			w, err = NewWorld(data)
			if err != nil {
				t.Fatal(err)
			}
			w.Actors = nil
			w.ScrollY, w.MaximumScrollY, w.Player.ScrollStep, w.ScrollDelta = 3000, 3344, 1, 1
			if err := w.spawnFirstMiddleStream(0, launch); err != nil {
				t.Fatal(err)
			}
			chain = append(chain[:0], w.Actors...)
			previousLaunch, previousPass = launch, -1
		}
		if pass != previousPass {
			if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
				t.Fatal(err)
			}
			previousPass = pass
		}
		if part < 0 || part > 13 {
			continue
		}
		actor := chain[part]
		tag := 0
		if actor.Binding.EntityID != 0 {
			if slot := w.Pool.Slot(actor.Binding.Slot); slot.allocated && slot.EntityID == actor.ID {
				tag = int(slot.ResourceTag)
			}
		}
		if tag != int(v[3]) {
			t.Fatalf("World chain removal differs launch%d pass%d part%d: tag=%d want%d", launch, pass, part, tag, v[3])
		}
		if v[3] == 0 || v[3] == 4 {
			continue
		}
		if part == 0 || part == 13 {
			continue
		}
		if int(actor.X) != int(int32(v[4])>>16) || int(actor.Y) != int(int32(v[5])>>16) || actor.Visible != (v[9] != 0) {
			t.Fatalf("World chain differs %v: x/y=%v/%v visible=%t", v[:8], actor.X, actor.Y, actor.Visible)
		}
	}
}
