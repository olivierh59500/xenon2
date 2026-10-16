package controls

import "math"

const (
	Height     = 600
	SceneWidth = 960
)

// Button names the actions available without an Android keyboard.
type Button uint8

const (
	Fire Button = iota
	Dive
	Enter
	Menu
	Pause
	Cheats
	ButtonCount
)

type Rect struct{ X, Y, Width, Height float64 }

func (r Rect) Contains(x, y float64) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.Width && y < r.Y+r.Height
}

type ControlButton struct {
	Label  string
	Bounds Rect
	Round  bool
}

func (b ControlButton) Contains(x, y float64) bool {
	if !b.Round {
		return b.Bounds.Contains(x, y)
	}
	r := b.Bounds.Width / 2
	return math.Hypot(x-b.Bounds.X-r, y-b.Bounds.Y-r) <= r
}

// Layout keeps the original canvas centered between two control panels.
type Layout struct {
	Width                       int
	SceneX                      float64
	StickX, StickY, StickRadius float64
	Buttons                     [ButtonCount]ControlButton
}

func LogicalWidth(outsideWidth, outsideHeight int) int {
	if outsideWidth <= 0 || outsideHeight <= 0 {
		return 1344
	}
	return min(1680, max(1200, (outsideWidth*Height+outsideHeight-1)/outsideHeight))
}

func NewLayout(width int) Layout {
	width = max(1200, width)
	side := float64(width-SceneWidth) / 2
	l := Layout{Width: width, SceneX: side, StickX: side / 2, StickY: 450, StickRadius: math.Min(78, side/2-14)}
	right := float64(width) - side
	l.Buttons[Menu] = ControlButton{Label: "MENU", Bounds: Rect{14, 52, side - 28, 48}}
	l.Buttons[Cheats] = ControlButton{Label: "CHEATS", Bounds: Rect{14, 116, side - 28, 48}}
	l.Buttons[Enter] = ControlButton{Label: "ENTER", Bounds: Rect{right + 14, 52, side - 28, 48}}
	l.Buttons[Pause] = ControlButton{Label: "PAUSE", Bounds: Rect{right + 14, 116, side - 28, 48}}
	circle := func(label string, y, radius float64) ControlButton {
		return ControlButton{label, Rect{right + side/2 - radius, y - radius, radius * 2, radius * 2}, true}
	}
	l.Buttons[Dive] = circle("DIVE", 282, math.Min(44, side/2-14))
	l.Buttons[Fire] = circle("FIRE", 450, math.Min(70, side/2-14))
	return l
}

// Frame distinguishes held gameplay controls from one-shot interface actions.
type Frame struct {
	X, Y                                              int
	LeftPressed, RightPressed, UpPressed, DownPressed bool
	Held, Pressed                                     [ButtonCount]bool
	Tap                                               bool
	TapX, TapY                                        float64
	AnyPressed                                        bool
}

type Pad struct {
	Layout   Layout
	Joystick Joystick
	Frame    Frame
}

func (p *Pad) Place(layout Layout) {
	if p.Layout == layout {
		return
	}
	p.Layout = layout
	p.Frame = Frame{}
	p.Joystick.Place(layout.StickX, layout.StickY, layout.StickRadius)
}

// Update retains the stick's finger while independently sampling every button.
// Held fingers cannot be adopted by the stick after another finger is released.
func (p *Pad) Update(contacts []Touch) Frame {
	previous := p.Frame
	p.Joystick.Update(contacts)
	f := Frame{X: p.Joystick.X, Y: p.Joystick.Y}
	f.LeftPressed = f.X < 0 && previous.X >= 0
	f.RightPressed = f.X > 0 && previous.X <= 0
	f.UpPressed = f.Y < 0 && previous.Y >= 0
	f.DownPressed = f.Y > 0 && previous.Y <= 0
	scene := Rect{p.Layout.SceneX, 0, SceneWidth, Height}
	for _, t := range contacts {
		f.AnyPressed = f.AnyPressed || t.Pressed
		if p.Joystick.Owns(t.ID) {
			continue
		}
		for button, b := range p.Layout.Buttons {
			f.Held[button] = f.Held[button] || b.Contains(t.X, t.Y)
		}
		if t.Pressed && scene.Contains(t.X, t.Y) && !f.Tap {
			f.Tap, f.TapX, f.TapY = true, (t.X-p.Layout.SceneX)/3, t.Y/3
		}
	}
	for button := range f.Held {
		f.Pressed[button] = f.Held[button] && !previous.Held[button]
	}
	p.Frame = f
	return f
}
