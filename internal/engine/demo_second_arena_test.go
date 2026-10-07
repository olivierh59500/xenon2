package engine

import (
	"reflect"
	"testing"
)

func TestSecondArenaBeamReturnsOrdinaryControlsWithoutMutatingSourceOptional(t *testing.T) {
	w := secondDefensePredictionFixture(t, 12, false)
	w.Dive = DiveState{}
	w.Player.X, w.Player.Y = 268, 101
	w.PreviousPlayer = w.Player
	w.secondScheduler.DefenseFlags = 2
	w.advanceSecondNode(w.secondNodes[0])
	player, equipment, random, pool, scheduler := w.Player, w.Equipment, w.RandomState(), *w.Pool, *w.secondScheduler
	members := secondDefensePredictionMembers(w)
	states := make([]SecondDefenseSegment, len(members))
	for index, member := range members {
		states[index] = *member.secondSegment
	}
	p := DemoPilot{practicedRoute: true}
	if _, _, _, found := p.secondDefenseTarget(w); !found {
		t.Fatal("source aiming objective missing")
	}
	input := demoSecondArenaBeam(w, &p, MotionInput{Left: true, Up: true})
	for range 3 {
		if again := demoSecondArenaBeam(w, &p, MotionInput{Left: true, Up: true}); again != input {
			t.Fatal("unchanged source arena produced inconsistent controls")
		}
	}
	if w.Player != player || w.Equipment != equipment || w.RandomState() != random || *w.Pool != pool || *w.secondScheduler != scheduler {
		t.Fatal("beam planning changed live game state")
	}
	for index, member := range members {
		if !reflect.DeepEqual(*member.secondSegment, states[index]) {
			t.Fatal("beam planning advanced a live formation member")
		}
	}
	valid := false
	for _, direction := range demoDirections {
		valid = valid || input == direction
	}
	if !valid {
		t.Fatal("arena beam returned an input outside ordinary directional controls")
	}
}

func TestSecondDefenseCameraTranslationMatchesFullSourceForecastOptional(t *testing.T) {
	w := secondDefensePredictionFixture(t, 12, false)
	var baseline, actual [8 * ActorPoolCapacity]demoSecondDefenseView
	count, ok := demoSecondDefenseForecast(w, 8, [8]int{}, baseline[:])
	if !ok {
		t.Fatal("source baseline forecast rejected")
	}
	deltas := [8]int{1, -1, 2, 0, -2, 1, 1, -1}
	if n, ok := demoSecondDefenseForecast(w, 8, deltas, actual[:]); !ok || n != count {
		t.Fatal("source camera forecast rejected")
	}
	shift := 0
	for future, delta := range deltas {
		shift += delta
		for index := 0; index < count; index++ {
			want := baseline[future*count+index]
			want.Y += shift
			if want.Active {
				want.Bounds.Top += shift
				want.Bounds.Bottom += shift
			}
			if actual[future*count+index] != want {
				t.Fatalf("future%d part%d rigid translation differs from full source callbacks", future, index)
			}
		}
	}
}

func BenchmarkSecondArenaBeamOriginal(b *testing.B) {
	w, err := NewWorld(originalWorldData(b, 2))
	if err != nil {
		b.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2700, 2896, 2896
	w.Ready, w.MaterializationFrames = false, 0
	w.Player.X, w.Player.Y = 14, 16
	w.Dive = DiveState{Phase: 4, Remaining: 136}
	for range 12 {
		if err := w.Step(Input{}); err != nil {
			b.Fatal(err)
		}
	}
	w.Dive = DiveState{}
	w.Player.X, w.Player.Y = 268, 101
	w.secondScheduler.DefenseFlags = 2
	w.advanceSecondNode(w.secondNodes[0])
	p := DemoPilot{practicedRoute: true}
	if _, _, _, found := p.secondDefenseTarget(w); !found {
		b.Fatal("benchmark omitted its source aiming objective")
	}
	demoSecondArenaBeam(w, &p, MotionInput{Left: true, Up: true})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		demoSecondArenaBeam(w, &p, MotionInput{Left: true, Up: true})
	}
}
