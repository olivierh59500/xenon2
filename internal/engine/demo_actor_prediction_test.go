package engine

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"xenon2/internal/visualassets"
)

func assertDemoActorPredictionReadOnly(t *testing.T, w *World, actor *WorldActor, cameraY int) {
	t.Helper()
	before, pool, random := *actor, *w.Pool, w.RandomState()
	player, equipment, frame, camera, nextID := w.Player, w.Equipment, w.Frame, w.ScrollY, w.nextActorID
	part, _ := json.Marshal(actor.part)
	path, _ := json.Marshal(actor.path)
	for future := 1; future <= 18; future++ {
		if _, ok := demoActorPrediction(w, actor, future, cameraY); !ok {
			t.Fatalf("supported actor could not predict future%d", future)
		}
	}
	afterPart, _ := json.Marshal(actor.part)
	afterPath, _ := json.Marshal(actor.path)
	if !reflect.DeepEqual(*actor, before) || *w.Pool != pool || w.RandomState() != random || w.Player != player || w.Equipment != equipment || w.Frame != frame || w.ScrollY != camera || w.nextActorID != nextID || string(part) != string(afterPart) || string(path) != string(afterPath) {
		t.Fatal("actor prediction changed live controller, data, pool, random or player state")
	}
}

func TestDemoActorPredictionMatchesOriginalThirdDelayedPathsOptional(t *testing.T) {
	for _, pathID := range []int{23, 24} {
		t.Run(fmt.Sprintf("path%d", pathID), func(t *testing.T) {
			w, err := NewWorld(originalWorldData(t, 3))
			if err != nil {
				t.Fatal(err)
			}
			var wave visualassets.Wave
			for _, candidate := range w.Level.Encounters.Moving {
				if candidate.PathID == pathID && candidate.Count == 2 {
					wave = candidate
					break
				}
			}
			if wave.Count == 0 {
				t.Fatal("original delayed path wave is missing")
			}
			// This compares one original encounter's callbacks with World.Step.
			// New births, weapons and terrain contact are outside this fixture.
			w.Level.Encounters.Moving, w.Level.Encounters.Fixed = nil, nil
			w.Coverage = nil
			w.Ready, w.MaterializationFrames = false, 0
			w.Player.X = 300
			w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = wave.TriggerY, wave.TriggerY, wave.TriggerY
			if err := w.spawnWave(wave); err != nil {
				t.Fatal(err)
			}
			var actor *WorldActor
			for _, candidate := range w.Actors {
				if candidate.part != nil && candidate.part.ResourceTag == 212 && candidate.motion.Remaining < 0 {
					actor = candidate
				}
			}
			if actor == nil {
				t.Fatal("original wave did not create its delayed resource212 member")
			}
			if err := w.Step(Input{}); err != nil {
				t.Fatal(err)
			}
			if actor.Y != actor.PreviousY || actor.motion.Remaining >= 0 {
				t.Fatal("the actual delayed actor did not establish a stationary birth history")
			}
			assertDemoActorPredictionReadOnly(t, w, actor, w.ScrollY)
			var predictions [18]demoActorView
			startY := int(actor.Y)
			for future := range predictions {
				predictions[future], _ = demoActorPrediction(w, actor, future+1, w.ScrollY)
			}
			for future, predicted := range predictions {
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
				if predicted.X != int(actor.X) || predicted.Y != int(actor.Y) || predicted.Sprite != actor.Sprite || predicted.Bounds != actor.Collision || predicted.Active != actor.Active || predicted.Visible != actor.Visible {
					t.Fatalf("future%d path%d prediction%+v actual xy%v,%v sprite%s bounds%+v active%v visible%v", future+1, pathID, predicted, actor.X, actor.Y, actor.Sprite, actor.Collision, actor.Active, actor.Visible)
				}
			}
			future := 8
			if pathID == 24 {
				future = 10
			}
			miss := predictions[future-1].Y - startY
			if miss < 30 || miss > 40 {
				t.Fatalf("birth release did not expose the old stationary forecast error: %d pixels", miss)
			}
		})
	}
}

func TestDemoActorPredictionUsesFutureHeadingCollisionPrefix(t *testing.T) {
	w := testWorld(t)
	w.Level.Encounters.Moving = nil
	w.Level.Paths.SineTable[0], w.Level.Paths.SineTable[64], w.Level.Paths.SineTable[128], w.Level.Paths.SineTable[192] = 0, 64, 0, -64
	path := &visualassets.Path{ID: 99, Commands: []visualassets.PathCommand{{Kind: "origin", X: 100, Y: 100}, {Kind: "curve", Heading: 0, Duration: 1}, {Kind: "curve", Heading: 128, Duration: 20}, {Kind: "end"}}}
	motion, err := NewPathMotion(path, PathMotionConfig{Budget: 1})
	if err != nil {
		t.Fatal(err)
	}
	part := &visualassets.ActorPart{ResourceTag: 200, MotionMode: "path-heading-frames", HeadingFrames: []string{"right", "left"}, HeadingShift: 7}
	w.movingSpriteBoxes = map[string]visualassets.CollisionBox{"right": {X: -2, Y: -3, Width: 5, Height: 7}, "left": {X: -12, Y: -1, Width: 25, Height: 3}}
	actor := &WorldActor{Active: true, Visible: true, ActorList: "moving", part: part, path: path, motion: motion, X: 100, Y: 100}
	actor.selectSprite()
	actor.Collision = ActorCollisionRect(w.movingSpriteBoxes[actor.Sprite], 100, 100)
	if err := w.bindWorldActor(actor); err != nil {
		t.Fatal(err)
	}
	w.Actors = []*WorldActor{actor}
	assertDemoActorPredictionReadOnly(t, w, actor, w.ScrollY)
	predicted, ok := demoActorPrediction(w, actor, 2, w.ScrollY)
	if !ok || predicted.Sprite != "left" || predicted.Bounds.Right-predicted.Bounds.Left != 24 {
		t.Fatal("turn prediction retained the preceding heading's narrow collision prefix")
	}
	for range 2 {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if predicted.X != int(actor.X) || predicted.Y != int(actor.Y) || predicted.Sprite != actor.Sprite || predicted.Bounds != actor.Collision {
		t.Fatal("heading and collision prediction differed from actual world callbacks")
	}
}

func TestDemoActorPredictionUsesActorPhaseCameraUnderReverse(t *testing.T) {
	w := testWorld(t)
	w.Level.Encounters.Moving = nil
	w.Player.X, w.Player.Y = 250, 176
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 1000, 1016, 1016
	w.Rewind = NewTerrainRewind(1000, 250, 176)
	w.movingSpriteBoxes = map[string]visualassets.CollisionBox{"anchor": {X: -2, Y: -3, Width: 5, Height: 7}}
	clip := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "anchor"}}}
	actor := &WorldActor{Active: true, Visible: true, ActorList: "moving", fixed: true, mapY: 1200, X: 100, Y: 200, Sprite: "anchor", animation: clip, animationState: NewAnimation(clip), part: &visualassets.ActorPart{ResourceTag: 200, MotionMode: "world-anchored"}}
	actor.Collision = ActorCollisionRect(w.movingSpriteBoxes[actor.Sprite], 100, 200)
	if err := w.bindWorldActor(actor); err != nil {
		t.Fatal(err)
	}
	w.Actors = []*WorldActor{actor}
	original := *actor
	forecast := newDemoMotionForecast(w)
	for future := 1; future <= 18; future++ {
		cameraY := forecast.scroll.Y
		predicted, ok := demoActorPrediction(w, &original, future, cameraY)
		if !ok || !forecast.advance(w, MotionInput{Down: true}) {
			t.Fatal("supported reverse-scroll prediction failed")
		}
		if err := w.Step(Input{Motion: MotionInput{Down: true}}); err != nil {
			t.Fatal(err)
		}
		if predicted.Y != int(actor.Y) || predicted.Bounds != actor.Collision {
			t.Fatalf("future%d camera%d predicted%+v actual y%v bounds%+v", future, cameraY, predicted, actor.Y, actor.Collision)
		}
	}
}

func TestDemoActorPredictionRejectsUnknownControllers(t *testing.T) {
	w := testWorld(t)
	actor := &WorldActor{Active: true, Visible: true, ActorList: "moving", X: 100, PreviousX: 90, Y: 100, PreviousY: 90, part: &visualassets.ActorPart{MotionMode: "fourth-falling"}, fourthFalling: &FourthFallingActor{}}
	if _, ok := demoActorPrediction(w, actor, 18, w.ScrollY); ok {
		t.Fatal("unknown controller received an unsupported linear guarantee")
	}
	actor.part.MotionMode = "path"
	if _, ok := demoActorPrediction(w, actor, 1, w.ScrollY); ok {
		t.Fatal("missing path data was accepted")
	}
	if _, ok := demoActorPrediction(w, actor, 19, w.ScrollY); ok {
		t.Fatal("prediction exceeded its bounded horizon")
	}
}

func BenchmarkDemoActorPredictionOriginal(b *testing.B) {
	w, err := NewWorld(originalWorldData(b, 3))
	if err != nil {
		b.Fatal(err)
	}
	var wave visualassets.Wave
	for _, candidate := range w.Level.Encounters.Moving {
		if candidate.PathID == 24 && candidate.Count == 1 && candidate.MotionBudget == 9 {
			wave = candidate
			break
		}
	}
	if wave.Count == 0 {
		b.Fatal("original path24 wave with movement budget9 is missing")
	}
	if err := w.spawnWave(wave); err != nil {
		b.Fatal(err)
	}
	var actor *WorldActor
	for _, candidate := range w.Actors {
		if candidate.part != nil && candidate.part.ResourceTag == 212 && candidate.path != nil && candidate.path.ID == 24 {
			actor = candidate
			break
		}
	}
	if actor == nil {
		b.Fatal("original resource212 path24 actor is missing")
	}
	var view demoActorView
	var supported bool
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		view, supported = demoActorPrediction(w, actor, 18, w.ScrollY)
		if !supported {
			b.Fatal("original actor prediction became unsupported")
		}
	}
	if !view.Active || view.Bounds.Empty() {
		b.Fatal("benchmark did not forecast the original live actor geometry")
	}
}
