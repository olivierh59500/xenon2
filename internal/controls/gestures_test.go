package controls

import "testing"

func TestSystemGestureContactCannotAcquireGameControlsAfterLeavingEdge(t *testing.T) {
	layout := NewLayout(1344)
	areas := GestureInsets{Left: 12, Top: 8, Right: 12, Bottom: 24}
	for _, start := range []Touch{
		{ID: 1, X: 1, Y: 300, Pressed: true},
		{ID: 1, X: 1343, Y: 300, Pressed: true},
		{ID: 1, X: 672, Y: 1, Pressed: true},
		{ID: 1, X: 672, Y: 599, Pressed: true},
	} {
		var guard GestureGuard
		var pad Pad
		pad.Place(layout)
		if got := pad.Update(guard.Filter(nil, []Touch{start}, 1344, Height, areas)); got != (Frame{}) {
			t.Fatalf("system gesture started a game action: %+v", got)
		}
		fire := layout.Buttons[Fire].Bounds
		drag := Touch{ID: 1, X: fire.X + fire.Width/2, Y: fire.Y + fire.Height/2}
		if got := pad.Update(guard.Filter(nil, []Touch{drag}, 1344, Height, areas)); got != (Frame{}) {
			t.Fatalf("navigation swipe acquired fire after entering the panel: %+v", got)
		}
		pad.Update(guard.Filter(nil, nil, 1344, Height, areas))
		drag.Pressed = true
		if got := pad.Update(guard.Filter(nil, []Touch{drag}, 1344, Height, areas)); !got.AnyPressed || !got.Pressed[Fire] || !got.Held[Fire] {
			t.Fatalf("reused contact ID could not fire after the system gesture ended: %+v", got)
		}
	}
}

func TestSystemGestureFilteringPreservesMultitouchAndJoystickDrags(t *testing.T) {
	var guard GestureGuard
	var pad Pad
	pad.Place(NewLayout(1344))
	areas := GestureInsets{Left: 12, Top: 8, Right: 12, Bottom: 24}
	stick := Touch{ID: 1, X: pad.Layout.StickX - 30, Y: pad.Layout.StickY - 30, Pressed: true}
	r := pad.Layout.Buttons[Fire].Bounds
	fire := Touch{ID: 2, X: r.X + r.Width/2, Y: r.Y + r.Height/2, Pressed: true}
	swipe := Touch{ID: 3, X: 500, Y: 599, Pressed: true}
	f := pad.Update(guard.Filter(nil, []Touch{stick, fire, swipe}, 1344, Height, areas))
	if f.X != -1 || f.Y != -1 || !f.Pressed[Fire] || !f.Held[Fire] || f.Tap {
		t.Fatalf("gesture filtering lost real simultaneous controls: %+v", f)
	}
	stick.X, stick.Y, stick.Pressed = 0, 599, false
	fire.Pressed = false
	f = pad.Update(guard.Filter(nil, []Touch{stick, fire}, 1344, Height, areas))
	if f.X != -1 || f.Y != 1 || !f.Held[Fire] || f.Pressed[Fire] || !pad.Joystick.Owns(1) {
		t.Fatalf("a captured joystick finger could not drag into an edge: %+v", f)
	}
	if f = pad.Update(guard.Filter(nil, nil, 1344, Height, areas)); f != (Frame{}) {
		t.Fatalf("application suspension left a control held: %+v", f)
	}
}
