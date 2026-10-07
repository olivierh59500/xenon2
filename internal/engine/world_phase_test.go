package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"xenon2/internal/visualassets"
)

func TestSourceGameplayPhaseOrderNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare local original traces")
	}
	file, err := os.Open(filepath.Join(root, "gameplay-phase-order.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"count/-1", "stage/-1", "update/0", "update/1", "update/2", "update/3", "update/4", "draw/0", "draw/2", "draw/1", "draw/3", "draw/4", "timers/-1", "sparks/-1", "encounters/-1"}
	if len(rows) != len(expected)+1 {
		t.Fatal("source phase trace is incomplete")
	}
	for i, row := range rows[1:] {
		if row[1]+"/"+row[2] != expected[i] {
			t.Fatalf("source phase%d=%v", i, row)
		}
	}
}

func TestSourceMovingCountNativeGroupsAndTombstonesOptional(t *testing.T) {
	nativeCombatRows(t, "gameplay-phase-count.csv", func(v []int64) {
		w := testWorld(t)
		var actors [4]*WorldActor
		for i := range actors {
			tag := int(v[2])
			if i == 0 {
				tag = int(v[1])
			}
			if i == 3 {
				tag = 184
			}
			actor := &WorldActor{Active: true, ActorList: "moving", part: &visualassets.ActorPart{ResourceTag: tag}}
			binding, err := w.reserveWorldActor(int16(tag), ActorPoolMoving, true)
			if err != nil {
				t.Fatal(err)
			}
			actor.ID, actor.Binding = binding.EntityID, binding
			w.poolActors[binding.Slot] = actor
			w.Actors = append(w.Actors, actor)
			actors[i] = actor
		}
		if v[3] != 0 {
			for i := 0; i < 3; i++ {
				slot := w.Pool.Slot(actors[i].Binding.Slot)
				slot.Linked = true
				slot.Residue.OwnerSlot = actors[0].Binding.Slot
			}
		}
		w.countSourceMovingActors()
		if w.MovingEnemyCount != int(v[4]) {
			t.Fatalf("native count case%d: got%d want%d", v[0], w.MovingEnemyCount, v[4])
		}
	})
}

func TestStageBornStreamsAdvanceBeforeLateEncounterBirthsOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 160, 0, 192, 192
	w.cursor = RestartEncounterCursor(w.ScrollY)
	w.InvulnerableFrames = 10000
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	var head *WorldActor
	for _, actor := range w.Actors {
		if actor.thirdFinalMember != nil && actor.thirdPart.Index == 0 {
			head = actor
		}
	}
	if head == nil || !head.Visible || head.thirdFinalMember.Motion.ProgramCounter <= 1 {
		t.Fatal("pre-actor stage birth must advance during its construction pass")
	}
}
