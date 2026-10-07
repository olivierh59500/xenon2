package engine

import (
	"fmt"
	"reflect"
	"testing"

	"xenon2/internal/visualassets"
)

func thirdChainPredictionFixture(t *testing.T, variant, y int) (*World, *WorldActor) {
	t.Helper()
	w, err := NewWorld(originalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.Level.Encounters = &visualassets.Encounters{}
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 2700, 0, 2800, 2800
	w.Player.X, w.Player.Y = 300, 110
	w.MaterializationFrames = 0
	// A geometry fixture: diving excludes combat deletion while the real
	// player and actor callbacks establish their source-order positions.
	w.Dive = DiveState{Phase: 4, Remaining: 136}
	w.spawnThirdChain(visualassets.FixedEncounter{X: 128, Y: 2700 + y, Variant: variant})
	for _, actor := range w.Actors {
		if actor.thirdChainPart == 1 {
			return w, actor
		}
	}
	t.Fatal("real chain constructor did not create a leader")
	return nil, nil
}

func assertThirdChainPredictionWorldSteps(t *testing.T, w *World, leader *WorldActor, motion MotionInput) (hidden, retired int) {
	t.Helper()
	var deltas, playerYs [8]int
	forecast, delta := newDemoMotionForecast(w), w.ScrollDelta
	for future := range deltas {
		deltas[future] = delta
		if !forecast.advance(w, motion) {
			t.Fatal("source geometry fixture reached unrelated terrain")
		}
		playerYs[future] = forecast.player.Y
		delta = forecast.scroll.ActualStep
	}
	state, pool, random := *leader.thirdChain, *w.Pool, w.RandomState()
	player, equipment, frame, camera := w.Player, w.Equipment, w.Frame, w.ScrollY
	var before [8]WorldActor
	for index, actor := range leader.thirdChainMembers {
		before[index] = *actor
	}
	var rows [8][8]demoActorView
	nextRandom, ok := demoThirdChainPrediction(w, leader.thirdChainMembers[7], 8, deltas, playerYs, random, rows[:])
	if !ok {
		t.Fatal("source chain forecast was rejected")
	}
	if *leader.thirdChain != state || *w.Pool != pool || w.RandomState() != random || w.Player != player || w.Equipment != equipment || w.Frame != frame || w.ScrollY != camera {
		t.Fatal("prediction changed live controller, storage, random or player state")
	}
	for index, actor := range leader.thirdChainMembers {
		if !reflect.DeepEqual(*actor, before[index]) {
			t.Fatal("prediction changed a live member")
		}
	}
	for future := range rows {
		if err := w.Step(Input{Motion: motion}); err != nil {
			t.Fatal(err)
		}
		for index, actor := range leader.thirdChainMembers {
			view := rows[future][index]
			if view.X != int(actor.X) || view.Y != int(actor.Y) || view.Sprite != actor.Sprite || view.Bounds != actor.Collision || view.Active != actor.Active || view.Visible != actor.Visible {
				t.Fatalf("future%d member%d: predicted %+v actual xy%v/%v image%s collider%+v active%v visible%v", future+1, index, view, actor.X, actor.Y, actor.Sprite, actor.Collision, actor.Active, actor.Visible)
			}
			if view.Active && !view.Visible {
				hidden++
				if view.Bounds.Empty() {
					t.Fatal("blank source member lost its live collision prefix")
				}
			}
			if !view.Active {
				retired++
			}
		}
	}
	if w.RandomState() != nextRandom {
		t.Fatal("copied activation draw differed from actual isolated source callbacks")
	}
	return
}

func TestDemoThirdChainPredictionOriginalWorldStepsOptional(t *testing.T) {
	for _, variant := range []int{0, 1} {
		for _, scenario := range []string{"activation", "cooldown", "retirement"} {
			t.Run(fmt.Sprintf("variant%d/%s", variant, scenario), func(t *testing.T) {
				y := 150
				if scenario == "retirement" {
					y = 230
				}
				w, leader := thirdChainPredictionFixture(t, variant, y)
				if scenario == "cooldown" {
					leader.thirdChain.Phase, leader.thirdChain.Speed, leader.thirdChain.AmplitudeOrCooldown = 2, -2, 14
				}
				hidden, retired := assertThirdChainPredictionWorldSteps(t, w, leader, MotionInput{Down: true})
				if scenario != "retirement" && hidden == 0 {
					t.Fatal("comparison did not cover source blank colliders")
				}
				if scenario == "retirement" && retired == 0 {
					t.Fatal("comparison omitted source screen retirement")
				}
			})
		}
	}
}

func TestDemoThirdChainPredictionDependsOnCandidateShipHeightOptional(t *testing.T) {
	w, leader := thirdChainPredictionFixture(t, 0, 150)
	var near, far [8]int
	for index := range near {
		near[index], far[index] = 150, 80
	}
	var closeRows, farRows [8][8]demoActorView
	seed := w.RandomState()
	closeRandom, closeOK := demoThirdChainPrediction(w, leader, 8, [8]int{}, near, seed, closeRows[:])
	farRandom, farOK := demoThirdChainPrediction(w, leader, 8, [8]int{}, far, seed, farRows[:])
	if !closeOK || !farOK || closeRows[0][7].X == farRows[0][7].X || closeRandom == seed || farRandom != seed || farRows[0][0].Visible || farRows[0][0].Bounds.Empty() {
		t.Fatal("source candidate height did not control activation and its copied random draw")
	}
}

func TestDemoThirdChainPredictionSixPassReturnCooldownOptional(t *testing.T) {
	w, leader := thirdChainPredictionFixture(t, 0, 150)
	leader.thirdChain.Phase, leader.thirdChain.Speed, leader.thirdChain.AmplitudeOrCooldown = 2, -2, 14
	deltas := [8]int{1, -1, 2, 0, -2, 1, 0, 0}
	seed, liveState := w.RandomState(), *leader.thirdChain
	var expected [8][8]demoActorView
	for height := 16; height <= 176; height++ {
		var playerYs [8]int
		for future := range playerYs {
			playerYs[future] = height
		}
		var rows [8][8]demoActorView
		next, ok := demoThirdChainPrediction(w, leader, 6, deltas, playerYs, seed, rows[:])
		if !ok || next != seed {
			t.Fatalf("candidate height %d drew during the six-pass return/cooldown", height)
		}
		if height == 16 {
			expected = rows
		} else if rows != expected {
			t.Fatalf("candidate height %d affected an already active sweep's return/cooldown", height)
		}
	}
	y := 150
	for future := 0; future < 6; future++ {
		y += deltas[future]
		for member, view := range expected[future] {
			if !view.Active || view.Y != y || view.X != leader.thirdChain.Parts[0].X || view.Bounds.Empty() {
				t.Fatalf("future %d member %d lost source return pose or blank collider: %+v", future+1, member, view)
			}
			if member < 7 && view.Visible != (future == 0 && member == 0) {
				t.Fatalf("future %d member %d returned with incorrect visibility", future+1, member)
			}
		}
	}
	// The first callback returns to phase zero; five further callbacks consume
	// -5..0. Only callback seven may activate again and consult candidate height.
	var near [8]int
	for future := range near {
		near[future] = 150
	}
	var seventh [8][8]demoActorView
	next, ok := demoThirdChainPrediction(w, leader, 7, deltas, near, seed, seventh[:])
	if !ok || next == seed || !seventh[6][0].Visible {
		t.Fatal("seventh callback did not resume candidate-sensitive activation")
	}
	if *leader.thirdChain != liveState || w.RandomState() != seed {
		t.Fatal("cooldown predictions changed live state")
	}
}

func TestDemoThirdChainPredictionRejectsUnsupportedRequests(t *testing.T) {
	w := testWorld(t)
	var rows [8][8]demoActorView
	if _, ok := demoThirdChainPrediction(w, nil, 8, [8]int{}, [8]int{}, w.RandomState(), rows[:]); ok {
		t.Fatal("missing source leader was accepted")
	}
}
