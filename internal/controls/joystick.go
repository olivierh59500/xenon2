// Package controls contains device-independent touch controls.
package controls

import "math"

// Touch is one current contact in logical screen coordinates. Pressed is true
// only on its first update, so a fire finger cannot acquire the stick by drifting.
type Touch struct {
	ID      int
	X, Y    float64
	Pressed bool
}

// Joystick owns one contact until release, including drags outside the base.
// Its output is digital, matching Xenon2's eight-way joystick input, while the
// thumb position follows the finger continuously within its circular travel.
type Joystick struct {
	CenterX, CenterY, Radius, Travel float64
	X, Y                             int
	OffsetX, OffsetY                 float64
	active                           bool
	contact, sector                  int
}

// Place keeps the stick centered in the available sidebar. A changed geometry
// cancels a contact instead of retaining a direction across an orientation change.
func (j *Joystick) Place(x, y, radius float64) {
	if j.CenterX == x && j.CenterY == y && j.Radius == radius {
		return
	}
	j.Reset()
	j.CenterX, j.CenterY, j.Radius, j.Travel = x, y, radius, radius*.62
}

func (j *Joystick) Active() bool     { return j.active }
func (j *Joystick) Owns(id int) bool { return j.active && j.contact == id }

func (j *Joystick) Reset() {
	j.active = false
	j.X, j.Y = 0, 0
	j.OffsetX, j.OffsetY = 0, 0
	j.sector = -1
}

func (j *Joystick) Update(touches []Touch) {
	var contact *Touch
	if j.active {
		for i := range touches {
			if touches[i].ID == j.contact {
				contact = &touches[i]
				break
			}
		}
		if contact == nil {
			j.Reset()
		}
	}
	if !j.active {
		for i := range touches {
			t := &touches[i]
			if t.Pressed && math.Hypot(t.X-j.CenterX, t.Y-j.CenterY) <= j.Radius && j.Radius > 0 {
				j.contact = t.ID
				j.active = true
				contact = t
				break
			}
		}
	}
	if contact == nil {
		return
	}
	dx, dy := contact.X-j.CenterX, contact.Y-j.CenterY
	distance := math.Hypot(dx, dy)
	scale := 1.0
	if distance > j.Travel {
		scale = j.Travel / distance
	}
	j.OffsetX, j.OffsetY = dx*scale, dy*scale
	if distance <= j.Travel*.22 {
		j.X, j.Y = 0, 0
		j.sector = -1
		return
	}
	angle := math.Atan2(dy, dx)
	// Keep a small angular margin around the selected sector to avoid flicker
	// when a thumb rests near the boundary between a cardinal and diagonal.
	if j.sector >= 0 {
		delta := math.Remainder(angle-float64(j.sector)*math.Pi/4, 2*math.Pi)
		if math.Abs(delta) <= math.Pi/8+3*math.Pi/180 {
			return
		}
	}
	j.sector = (int(math.Floor((angle+math.Pi/8)/(math.Pi/4))) + 8) % 8
	directions := [8][2]int{{1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}}
	j.X, j.Y = directions[j.sector][0], directions[j.sector][1]
}
