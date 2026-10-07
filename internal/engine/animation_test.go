package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestAnimationCountdownAndNonzeroLoop(t *testing.T) {
	a := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{
		{Sprite: "entry", Duration: 2}, {Sprite: "open", Duration: 1}, {Sprite: "close", Duration: 3},
	}, LoopFrom: 1}
	s := NewAnimation(a)
	for tick, want := range []string{"entry", "entry", "open", "close", "close", "close", "open", "close"} {
		if got := s.Sprite(a); got != want {
			t.Fatalf("tick %d: got %s, want %s", tick, got, want)
		}
		s.Advance(a)
	}
}

func TestAnimationZeroDurationStopsOnItsFrame(t *testing.T) {
	a := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{
		{Sprite: "moving", Duration: 1}, {Sprite: "resting", Duration: 0},
	}}
	s := NewAnimation(a)
	for range 50 {
		s.Advance(a)
	}
	if s.Sprite(a) != "resting" {
		t.Fatal("a zero-duration frame must remain displayed")
	}
}
