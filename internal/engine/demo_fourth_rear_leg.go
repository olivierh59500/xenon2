package engine

// A 32-pass route in this window cannot reach the falling-actor admission at 3664
// or the reverse-bound change at 3568. The existing forest fork stays separate.
func fourthRearLegWindow(w *World) bool {
	return w != nil && w.Level.Number == 4 && w.FourthMiddle == nil && w.ScrollY <= fourthForkEndCamera && w.ScrollY > 3664+nativeMotionDepthLimit
}

func (p *DemoPilot) fourthRearLegOwnsMotion(w *World, motion MotionInput) bool {
	if !p.fourthRearCommitted || !fourthRearLegWindow(w) {
		return false
	}
	_, ok := p.nativeMotion.guardSequence(w, motion)
	return ok && p.nativeMotion.targetX == p.fourthRearTargetX && p.nativeMotion.targetY == p.fourthRearTargetY
}

// FourthRearLegInput admits only a fresh, source-clear rearward geometric leg.
// Declined probes use a separate cache and leave ordinary navigation untouched.
func (p *DemoPilot) FourthRearLegInput(w *World) (Input, bool) {
	if !p.practicedRoute || !fourthRearLegWindow(w) || w.Ready || w.GameOver || !w.PlayerAlive || w.Rewind.Timer != 0 || w.Dive.Phase != 0 || w.Coverage == nil || w.Level.PlayerStencil == nil {
		return Input{}, false
	}
	if p.fourthRearCommitted {
		if motion, ok := p.nativeMotion.continueRoute(w, p.fourthRearTargetX, p.fourthRearTargetY); ok {
			return Input{Motion: motion}, true
		}
		p.fourthRearCommitted = false
	}
	if p.fourthRearNavigation == nil || p.fourthRearNavigation.world != w {
		p.fourthRearNavigation = &demoNavigation{}
	}
	n := p.fourthRearNavigation
	n.refresh(w)
	goal := w.Player.Y + w.ScrollY - 128
	if ordinary := p.navigation; ordinary != nil && ordinary.world == w && len(ordinary.path) != 0 {
		// Copy only route metadata. Rows are refreshed independently above;
		// path storage and any occupancy memo must never alias the fallback.
		n.path = append(n.path[:0], ordinary.path...)
		n.goal, n.frame = ordinary.goal, ordinary.frame
		n.targetX, n.pathTargetX = ordinary.targetX, ordinary.pathTargetX
		n.pointTargetX, n.retreat = ordinary.pointTargetX, ordinary.retreat
		goal = ordinary.goal
	}
	n.practiced = true
	worldY := w.Player.Y + w.ScrollY
	x, y, found := n.waypoint(w, goal)
	if !found || y <= worldY || len(n.path) == 0 || !n.clearSegment(w.Player.X, worldY, demoNavPoint{x, y}) {
		return Input{}, false
	}
	endpoint := n.path[len(n.path)-1]
	motion, found := p.nativeMotion.command(w, x, y, endpoint.x, endpoint.y)
	if !found {
		return Input{}, false
	}
	p.fourthRearTargetX, p.fourthRearTargetY = endpoint.x, endpoint.y
	p.fourthRearCommitted = true
	return Input{Motion: motion}, true
}
