package presentation

import (
	"testing"
	"xenon2/internal/engine"
)

func TestStarfieldSignedProjectionAndDepth(t *testing.T) {
	r := engine.NewRandomState()
	s := NewStarfield(&r, []uint8{7, 7, 8, 8, 6, 5, 4, 9})
	s.Stars[0] = Star{X: -1024, Y: 2048, Depth: 8191}
	s.Advance()
	star := s.Stars[0]
	if star.Depth != 7679 || star.ScreenX != 158 || star.ScreenY != 104 {
		t.Fatalf("wrong integer projection %+v", star)
	}
	for frame := 0; frame < 1000; frame++ {
		s.Advance()
		for _, star := range s.Stars {
			if star.ScreenX < 0 || star.ScreenX >= 320 || star.ScreenY < 0 || star.ScreenY >= 200 || star.Depth < 513 {
				t.Fatal("starfield produced an invalid visible point")
			}
		}
	}
}
