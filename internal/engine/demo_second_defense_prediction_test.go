package engine

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"xenon2/internal/visualassets"
)

func secondDefensePredictionFixture(t *testing.T, warmup int, reverse bool) *World {
	t.Helper()
	w, err := NewWorld(originalWorldData(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	w.Level.Encounters = &visualassets.Encounters{}
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2700, 2896, 2896
	w.Player.X, w.Player.Y = 14, 16
	w.Ready, w.MaterializationFrames = false, 0
	// Geometry fixture: underwater contact cannot kill a member while these
	// comparisons exercise its actual world callbacks. No survival is claimed.
	w.Dive = DiveState{Phase: 4, Remaining: 136}
	for range warmup {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if reverse {
		w.Player.Y = 176
	}
	return w
}

func secondDefensePredictionMembers(w *World) []*WorldActor {
	var storage [ActorPoolCapacity]*WorldActor
	var members []*WorldActor
	for _, actor := range w.orderedMovingActors(&storage) {
		if actor.Active && actor.secondSegment != nil {
			members = append(members, actor)
		}
	}
	return members
}

func assertSecondDefensePredictionReadOnly(t *testing.T, w *World, members []*WorldActor, deltas [8]int, dst []demoSecondDefenseView) {
	t.Helper()
	pool, random, player, equipment := *w.Pool, w.RandomState(), w.Player, w.Equipment
	scheduler, gates, heartbeat := *w.secondScheduler, w.secondGateCounters, w.secondStreamsUpdated
	frame, camera, maximum, nextID := w.Frame, w.ScrollY, w.MaximumScrollY, w.nextActorID
	tiles := append([]uint16(nil), w.Level.Terrain.Map...)
	actors := make([]WorldActor, len(members))
	states := make([]SecondDefenseSegment, len(members))
	data := make([][]byte, len(members))
	for index, actor := range members {
		actors[index], states[index] = *actor, *actor.secondSegment
		data[index], _ = json.Marshal([]any{actor.path, actor.secondPart})
	}
	for range 3 {
		count, ok := demoSecondDefenseForecast(w, 8, deltas, dst)
		if !ok || count != len(members) {
			t.Fatal("live original defense formation was rejected")
		}
	}
	if *w.Pool != pool || w.RandomState() != random || w.Player != player || w.Equipment != equipment || *w.secondScheduler != scheduler || w.secondGateCounters != gates || w.secondStreamsUpdated != heartbeat || w.Frame != frame || w.ScrollY != camera || w.MaximumScrollY != maximum || w.nextActorID != nextID || !slices.Equal(w.Level.Terrain.Map, tiles) {
		t.Fatal("defense prediction changed live game, scheduler, random, camera or terrain state")
	}
	for index, actor := range members {
		afterData, _ := json.Marshal([]any{actor.path, actor.secondPart})
		if !reflect.DeepEqual(*actor, actors[index]) || *actor.secondSegment != states[index] || string(afterData) != string(data[index]) {
			t.Fatal("defense prediction changed a live member, controller or resource")
		}
	}
}

func assertSecondDefensePredictionWorldSteps(t *testing.T, w *World, motion MotionInput) (hidden, materializing, retired int) {
	t.Helper()
	members := secondDefensePredictionMembers(w)
	if len(members) == 0 {
		t.Fatal("actual source scheduler did not create a defense formation")
	}
	var deltas [8]int
	forecast, delta := newDemoMotionForecast(w), w.ScrollDelta
	for future := range deltas {
		deltas[future] = delta
		if !forecast.advance(w, motion) {
			t.Fatal("geometry fixture encountered unrelated terrain")
		}
		delta = forecast.scroll.ActualStep
	}
	views := make([]demoSecondDefenseView, len(members)*8)
	assertSecondDefensePredictionReadOnly(t, w, members, deltas, views)
	for future := range 8 {
		if w.ScrollDelta != deltas[future] {
			t.Fatalf("future %d actor-phase scroll displacement changed", future+1)
		}
		if err := w.Step(Input{Motion: motion}); err != nil {
			t.Fatal(err)
		}
		for index, actor := range members {
			got := views[future*len(members)+index]
			if got.ID != actor.ID || got.PartIndex != actor.secondPart.Index || got.Stream != actor.secondSegment.Stream || got.X != int(actor.X) || got.Y != int(actor.Y) || got.Sprite != actor.Sprite || got.Bounds != actor.Collision || got.Active != actor.Active || got.Visible != actor.Visible || got.Materializing != actor.Materializing {
				t.Fatalf("future %d part %d stream %d: prediction %+v actual xy %v/%v sprite %s bounds %+v active %v visible %v materializing %v", future+1, actor.secondPart.Index, actor.secondSegment.Stream, got, actor.X, actor.Y, actor.Sprite, actor.Collision, actor.Active, actor.Visible, actor.Materializing)
			}
			if got.Active && !got.Visible {
				hidden++
				if got.Bounds.Empty() {
					t.Fatal("source hidden updater failed to publish its collision prefix")
				}
			}
			if got.Materializing {
				materializing++
			}
			if !got.Active {
				retired++
			}
		}
	}
	return
}

func TestDemoSecondDefenseForecastOriginalWorldStepsOptional(t *testing.T) {
	for _, warmup := range []int{1, 12, 44} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("warmup%d/reverse%v", warmup, reverse), func(t *testing.T) {
				w := secondDefensePredictionFixture(t, warmup, reverse)
				members := secondDefensePredictionMembers(w)
				if len(members) != 24 {
					t.Fatalf("source two-stream constructor created %d live members", len(members))
				}
				for index, actor := range members {
					if actor.secondPart.Index != 11-index%12 || actor.secondSegment.Stream != 1-index/12 {
						t.Fatal("source constructor order did not visit tail through head")
					}
				}
				hidden, materializing, retired := assertSecondDefensePredictionWorldSteps(t, w, MotionInput{Down: reverse})
				if warmup == 1 && (hidden == 0 || materializing == 0) {
					t.Fatal("birth comparison omitted hidden and materializing source callbacks")
				}
				if warmup == 44 && retired == 0 {
					t.Fatal("late-path comparison omitted actual retirement")
				}
				t.Logf("Compared 24 original members across 8 actual passes: hidden %d, materializing %d, retired %d", hidden, materializing, retired)
			})
		}
	}
}

func TestDemoSecondDefenseForecastSharesCopiedRandomInPhysicalOrder(t *testing.T) {
	w := testSecondArenaWorld(t)
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2700, 2896, 2896
	w.Player.X, w.Player.Y = 14, 16
	w.Dive = DiveState{Phase: 4, Remaining: 136}
	// The original paths do not branch. This valid two-choice diagnostic path
	// tests that simultaneous member releases share source-order random draws.
	for index := range w.secondWaveArt.Launches {
		path := &w.secondWaveArt.Launches[index].Path
		path.Commands = []visualassets.PathCommand{{Kind: "random-branch", Targets: []int{1, 2, 1, 2, 1, 2, 1, 2}}, {Kind: "curve", Heading: 0, Duration: 256}, {Kind: "curve", Heading: 128, Duration: 256}, {Kind: "end"}}
	}
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	before := w.RandomState()
	assertSecondDefensePredictionWorldSteps(t, w, MotionInput{})
	if w.RandomState() == before {
		t.Fatal("diagnostic branch did not consume the original shared stream")
	}
}

func TestDemoSecondDefenseForecastRejectsIncompleteRequests(t *testing.T) {
	w := testSecondArenaWorld(t)
	w.ScrollY = 2700
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	var output [8 * ActorPoolCapacity]demoSecondDefenseView
	for _, horizon := range []int{0, 9} {
		if _, ok := demoSecondDefenseForecast(w, horizon, [8]int{}, output[:]); ok {
			t.Fatal("unsupported forecast horizon was accepted")
		}
	}
	if _, ok := demoSecondDefenseForecast(w, 8, [8]int{}, output[:1]); ok {
		t.Fatal("short output buffer silently dropped live source members")
	}
	member := secondDefensePredictionMembers(w)[0]
	member.path = nil
	if _, ok := demoSecondDefenseForecast(w, 8, [8]int{}, output[:]); ok {
		t.Fatal("missing source path received a collision guarantee")
	}
}
