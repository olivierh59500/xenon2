package engine

import "math"

// StageInput handles the second level's three-node arena using normal movement
// and trigger commands. It targets stream heads while the nodes are shielded,
// then crosses to the next surviving node even when its side-dependent collider
// is closed. It never changes health, inventory, random state or arena flags.
func (p *DemoPilot) StageInput(w *World) (Input, bool) {
	if w == nil || w.Level.Number != 2 || w.Ready || w.GameOver || !w.PlayerAlive || p.Config.DisableBossAlignment || w.secondScheduler == nil || w.secondMiddleReleased || w.ScrollY < 2512 || w.ScrollY > 2896 {
		return Input{}, false
	}
	x, y, nodeVisible, found := p.secondDefenseTarget(w)
	if !found {
		return Input{}, false
	}
	input := Input{
		Motion: MotionInput{Left: w.Player.X > x+1, Right: w.Player.X < x-1, Up: w.Player.Y > y+1, Down: w.Player.Y < y-1 && w.Player.Y < 176},
		Fire:   (w.Frame+1)%2 != 0 && !w.blockedFireUntilRelease,
	}
	// A visible open node needs several legal firing pulses. Holding down at
	// the bottom keeps that firing window on screen under the native scroll
	// rules; off-screen nodes keep normal scrolling until they enter view.
	if w.secondScheduler.DefenseFlags != 3 && nodeVisible && absDemo(w.Player.X-x) <= 4 && w.Player.Y > 160 {
		input.Motion.Down = true
	}
	return input, true
}

func (p *DemoPilot) secondDefenseTarget(w *World) (x, y int, nodeVisible, found bool) {
	if w.secondScheduler.DefenseFlags != 3 {
		for _, actor := range w.secondNodes {
			if actor == nil || !actor.Active || actor.Health <= 0 || actor.secondNode == nil {
				continue
			}
			node := actor.secondNode
			y = node.TileY*16 - w.ScrollY
			return max(20, min(300, node.TileX*16+8)), max(40, min(176, y+40)), y >= 0 && y < 160, true
		}
	}
	best := math.Inf(1)
	for _, actor := range w.Actors {
		if !actor.Active || !actor.Visible || actor.Materializing || actor.Health <= 0 || actor.secondSegment == nil || actor.secondPart == nil || actor.secondPart.Index != 0 || actor.Collision.Empty() || actor.Y < 0 || actor.Y > float64(w.Player.Y-15) {
			continue
		}
		flight := max(1, min(12, (w.Player.Y-int(actor.Y)-6)/9))
		lead := int(actor.X + (actor.X-actor.PreviousX)*float64(flight))
		distance := math.Abs(float64(lead-w.Player.X)) + math.Abs(actor.Y-float64(w.Player.Y))*0.5
		if distance < best {
			best = distance
			x, y, found = lead, max(80, min(166, int(actor.Y)+48)), true
		}
	}
	// Heads whose predicted firing point is off the left edge leave movement
	// to the ordinary terrain and hazard controller instead of pinning the ship.
	if x < 0 {
		return 0, 0, false, false
	}
	return max(20, min(300, x)), y, false, found
}
