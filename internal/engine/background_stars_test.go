package engine

import "testing"

func TestBackgroundStarfieldConsumesOriginalDrawsAndWrapsFractionally(t *testing.T) {
	random := NewRandomState()
	expected := random
	stars := NewBackgroundStarfield(&random)
	for range 144 {
		expected.Next()
	}
	if random != expected {
		t.Fatal("initialization changed shared random draw count")
	}
	before := random
	for _, star := range stars.Stars {
		if star.X < 0 || star.X >= 320 || star.Y < 0 || star.Y >= 192 || star.Fraction != 0 {
			t.Fatalf("invalid initial star %+v", star)
		}
	}
	stars.Stars[0].Y = 191
	stars.Stars[47].Y = 191
	stars.Advance(1)
	if stars.Stars[0].Y != 0 || stars.Stars[0].Fraction != 0 || stars.Stars[47].Y != 0 || stars.Stars[47].Fraction != 376 {
		t.Fatal("source speed bands or row wrap changed")
	}
	stars.Advance(-1)
	if stars.Stars[0].Y != 191 || stars.Stars[47].Y != 191 || stars.Stars[47].Fraction != 0 || random != before {
		t.Fatal("reverse movement changed remainder or random state")
	}
}

func BenchmarkBackgroundStarfield(b *testing.B) {
	random := NewRandomState()
	stars := NewBackgroundStarfield(&random)
	b.ReportAllocs()
	for b.Loop() {
		stars.Advance(1)
	}
}
