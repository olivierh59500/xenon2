package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"xenon2/internal/visualassets"
)

func assertSweeperPredictionReadOnly(t *testing.T, w *World, actor *WorldActor) {
	t.Helper()
	before, pool, random := *actor, *w.Pool, w.RandomState()
	player, equipment, frame, camera, nextID := w.Player, w.Equipment, w.Frame, w.ScrollY, w.nextActorID
	art, err := json.Marshal(actor.fixedKind)
	if err != nil {
		t.Fatal(err)
	}
	for future := 1; future <= 18; future++ {
		if _, supported := demoActorPrediction(w, actor, future, w.ScrollY-(future-1)*w.ScrollDelta); !supported {
			t.Fatal("supported constant-camera history was rejected")
		}
	}
	afterArt, err := json.Marshal(actor.fixedKind)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*actor, before) || *w.Pool != pool || w.RandomState() != random || w.Player != player || w.Equipment != equipment || w.Frame != frame || w.ScrollY != camera || w.nextActorID != nextID || string(art) != string(afterArt) {
		t.Fatal("sweeper prediction changed live state or source art")
	}
}

func originalSweeperPredictionWorld(t *testing.T, right bool) (*World, *WorldActor) {
	t.Helper()
	w, err := NewWorld(originalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	var record visualassets.FixedEncounter
	wantX := 56
	if right {
		wantX = 248
	}
	for _, candidate := range w.Level.Encounters.Fixed {
		if candidate.EnemyKind == 2 && candidate.X == wantX {
			record = candidate
			break
		}
	}
	if record.EnemyKind != 2 {
		t.Fatal("original sweeper encounter is missing")
	}
	// Compare the actor controller and steady camera without unrelated births
	// or terrain motion. The player retains ordinary collision and damage.
	w.Level.Encounters = &visualassets.Encounters{}
	w.Ready, w.MaterializationFrames = false, 0
	w.Player.X, w.Player.Y = 300, 16
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = record.TriggerY, record.TriggerY+16, record.TriggerY+16
	w.MinimumScrollY = 0
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.spawnFixed(record)
	actor := w.Actors[0]
	if actor.fixedKind == nil || actor.fixedKind.Behavior != "horizontal-sweeper" {
		t.Fatal("original encounter did not select its sweeper callback")
	}
	return w, actor
}

func TestDemoSweeperPredictionMatchesOriginalCallbacksOptional(t *testing.T) {
	for _, fixture := range []struct {
		name  string
		right bool
		ready func(*World, *WorldActor) bool
	}{
		{"left attack", false, func(_ *World, a *WorldActor) bool { return a.fixedState.Phase == 0 && a.fixedState.X == 134 }},
		{"right attack", true, func(_ *World, a *WorldActor) bool { return a.fixedState.Phase == 0 && a.fixedState.X == 186 }},
		{"right edge reversal", true, func(_ *World, a *WorldActor) bool {
			return a.fixedState.Phase == 0 && a.fixedState.VelocityX > 0 && a.fixedState.X == 346
		}},
		{"retirement", false, func(w *World, a *WorldActor) bool { return a.fixedState.Y == w.MaximumScrollY+208-w.ScrollY }},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			w, actor := originalSweeperPredictionWorld(t, fixture.right)
			for pass := 0; pass < 400 && actor.Active && !fixture.ready(w, actor); pass++ {
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
				if !w.PlayerAlive || w.ScrollDelta != 1 {
					t.Fatal("ordinary callback fixture lost its steady camera")
				}
			}
			if !actor.Active || !fixture.ready(w, actor) {
				t.Fatal("actual sweeper never reached the source forecast boundary")
			}
			assertSweeperPredictionReadOnly(t, w, actor)
			if _, supported := demoActorPrediction(w, actor, 2, w.ScrollY+1); supported {
				t.Fatal("changing camera history received a constant-delta guarantee")
			}
			var predictions [6]demoActorView
			for future := range predictions {
				var supported bool
				predictions[future], supported = demoActorPrediction(w, actor, future+1, w.ScrollY-future*w.ScrollDelta)
				if !supported {
					t.Fatal("original sweeper forecast was rejected")
				}
			}
			if fixture.name == "left attack" && predictions[0].X != 136 || fixture.name == "right attack" && predictions[0].X != 184 {
				t.Fatal("forecast missed the exact native attack stop")
			}
			for future, predicted := range predictions {
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
				if predicted.X != int(actor.X) || predicted.Y != int(actor.Y) || predicted.Sprite != actor.Sprite || predicted.Bounds != actor.Collision || predicted.Active != actor.Active || predicted.Visible != actor.Visible {
					t.Fatalf("future%d predicted%+v actual xy%v,%v sprite%s bounds%+v active%v visible%v", future+1, predicted, actor.X, actor.Y, actor.Sprite, actor.Collision, actor.Active, actor.Visible)
				}
			}
		})
	}
}

func TestDemoSweeperPredictionRejectsChangingCameraHistoryOptional(t *testing.T) {
	w, actor := originalSweeperPredictionWorld(t, false)
	if _, supported := demoActorPrediction(w, actor, 3, w.ScrollY+1); supported {
		t.Fatal("a changing camera received a constant-delta sweeper forecast")
	}
	view, supported := demoActorPrediction(w, actor, 3, w.ScrollY-2*w.ScrollDelta)
	if !supported || !view.Active || view.Bounds.Empty() {
		t.Fatal("the actual constant-delta actor-phase camera was rejected")
	}
	assertSweeperPredictionReadOnly(t, w, actor)
}
