package controls

import "testing"

func TestMobilePanelsPreserveCanvasAndContainControls(t *testing.T) {
	for _, size := range [][2]int{{2424, 1080}, {1920, 1080}, {1080, 2424}, {0, 0}, {6000, 1000}} {
		l := NewLayout(LogicalWidth(size[0], size[1]))
		if l.Width < 1200 || l.Width > 1680 || l.SceneX*2+SceneWidth != float64(l.Width) {
			t.Fatalf("bad centered layout for %v: %+v", size, l)
		}
		if l.StickX-l.StickRadius < 12 || l.StickX+l.StickRadius > l.SceneX-12 {
			t.Fatalf("joystick clips scene or edge for %v", size)
		}
		for _, b := range l.Buttons {
			r := b.Bounds
			if r.X < 12 || r.X+r.Width > float64(l.Width)-12 || r.Y < 12 || r.Y+r.Height > Height-12 {
				t.Fatalf("button %s clips edge for %v", b.Label, size)
			}
			if r.X < l.SceneX+SceneWidth && r.X+r.Width > l.SceneX {
				t.Fatalf("button %s overlaps scene for %v", b.Label, size)
			}
		}
	}
}

func TestMobileStickFireDiveAndReleaseAreIndependent(t *testing.T) {
	var p Pad
	p.Place(NewLayout(1344))
	l := p.Layout
	center := func(id int, b Button) Touch {
		r := l.Buttons[b].Bounds
		return Touch{ID: id, X: r.X + r.Width/2, Y: r.Y + r.Height/2, Pressed: true}
	}
	stick := Touch{ID: 1, X: l.StickX - 30, Y: l.StickY - 30, Pressed: true}
	fire, dive := center(2, Fire), center(3, Dive)
	f := p.Update([]Touch{stick, fire, dive})
	if f.X != -1 || f.Y != -1 || !f.Held[Fire] || !f.Pressed[Fire] || !f.Pressed[Dive] {
		t.Fatalf("simultaneous diagonal/fire/dive lost: %+v", f)
	}
	stick.Pressed, fire.Pressed, dive.Pressed = false, false, false
	f = p.Update([]Touch{stick, fire, dive})
	if !f.Held[Fire] || !f.Held[Dive] || f.Pressed[Dive] || f.Pressed[Fire] || f.UpPressed {
		t.Fatalf("held controls emitted another edge: %+v", f)
	}
	f = p.Update([]Touch{stick, fire})
	if f.X != -1 || f.Y != -1 || !f.Held[Fire] || f.Held[Dive] {
		t.Fatalf("dive release altered other contacts: %+v", f)
	}
	f = p.Update([]Touch{fire})
	if f.X != 0 || f.Y != 0 || !f.Held[Fire] {
		t.Fatalf("stick release lost fire: %+v", f)
	}
	f = p.Update(nil)
	if f != (Frame{}) {
		t.Fatalf("release left a stuck control: %+v", f)
	}
}

func TestMobileSceneTapUsesOriginalPixelsAndMenuEdges(t *testing.T) {
	var p Pad
	p.Place(NewLayout(1344))
	f := p.Update([]Touch{{ID: 1, X: p.Layout.SceneX + 80*3, Y: 100 * 3, Pressed: true}})
	if !f.Tap || f.TapX != 80 || f.TapY != 100 {
		t.Fatalf("scene tap was not translated: %+v", f)
	}
	b := p.Layout.Buttons[Menu].Bounds
	f = p.Update([]Touch{{ID: 2, X: b.X + b.Width/2, Y: b.Y + b.Height/2, Pressed: true}})
	if f.Tap || !f.Pressed[Menu] {
		t.Fatalf("menu press leaked into canvas: %+v", f)
	}
	f = p.Update([]Touch{{ID: 2, X: b.X + b.Width/2, Y: b.Y + b.Height/2}})
	if f.Pressed[Menu] || !f.Held[Menu] {
		t.Fatalf("menu repeats while held: %+v", f)
	}
}
