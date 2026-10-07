package controls

import (
	"math"
	"testing"
)

func TestJoystickDirectionsAndNeutral(t *testing.T) {
	for _, d := range [][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}} {
		var j Joystick
		j.Place(100, 360, 80)
		j.Update([]Touch{{ID: 4, X: 100 + float64(d[0])*35, Y: 360 + float64(d[1])*35, Pressed: true}})
		if j.X != d[0] || j.Y != d[1] {
			t.Fatalf("direction %v: got (%d,%d)", d, j.X, j.Y)
		}
	}
	var j Joystick
	j.Place(100, 360, 80)
	j.Update([]Touch{{ID: 4, X: 104, Y: 364, Pressed: true}})
	if j.X != 0 || j.Y != 0 || !j.Active() {
		t.Fatal("center dead zone must retain the contact without movement")
	}
}

func TestJoystickCaptureMultitouchAndRelease(t *testing.T) {
	var j Joystick
	j.Place(100, 360, 80)
	j.Update([]Touch{{ID: 4, X: 100, Y: 360, Pressed: true}, {ID: 9, X: 900, Y: 360, Pressed: true}})
	j.Update([]Touch{{ID: 9, X: 900, Y: 360}, {ID: 4, X: 1000, Y: 360}})
	if !j.Owns(4) || j.Owns(9) || j.X != 1 || math.Abs(j.OffsetX-j.Travel) > 1e-9 {
		t.Fatal("the captured finger must keep control outside the base")
	}
	j.Update([]Touch{{ID: 9, X: 100, Y: 360}})
	if j.Active() || j.X != 0 || j.Y != 0 || j.OffsetX != 0 || j.OffsetY != 0 {
		t.Fatal("release must return to neutral without adopting another held finger")
	}
	j.Update([]Touch{{ID: 9, X: 100, Y: 360, Pressed: true}})
	if !j.Owns(9) {
		t.Fatal("a new press must acquire the stick")
	}
	j.Place(120, 360, 80)
	if j.Active() || j.X != 0 || j.Y != 0 {
		t.Fatal("layout changes must cancel a captured contact")
	}
}

func TestJoystickSectorBoundaryStability(t *testing.T) {
	var j Joystick
	j.Place(100, 360, 80)
	point := func(degrees float64, pressed bool) Touch {
		a := degrees * math.Pi / 180
		return Touch{ID: 1, X: 100 + 40*math.Cos(a), Y: 360 + 40*math.Sin(a), Pressed: pressed}
	}
	j.Update([]Touch{point(0, true)})
	j.Update([]Touch{point(24, false)})
	if j.X != 1 || j.Y != 0 {
		t.Fatal("small boundary movement must preserve the cardinal direction")
	}
	j.Update([]Touch{point(28, false)})
	if j.X != 1 || j.Y != 1 {
		t.Fatal("a deliberate diagonal movement must change direction")
	}
	j.Update([]Touch{point(22, false)})
	if j.X != 1 || j.Y != 1 {
		t.Fatal("diagonal direction must remain stable near the boundary")
	}
}
