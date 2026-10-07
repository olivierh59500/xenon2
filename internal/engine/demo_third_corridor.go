package engine

// ThirdCorridorInput follows the original terrain after the middle merchant.
func (p *DemoPilot) ThirdCorridorInput(w *World) (Input, bool) {
	if w == nil || w.Level.Number != 3 || w.Ready || w.GameOver || !w.PlayerAlive || p.Config.DisableBossAlignment || w.ThirdMiddle == nil || !w.ThirdMiddle.Defeated || w.PendingExitDrops != 0 || w.ShopReady || w.ScrollY <= 208 || w.Coverage == nil || w.Level.PlayerStencil == nil {
		return Input{}, false
	}
	if p.navigation == nil {
		p.navigation = &demoNavigation{}
	}
	n := p.navigation
	n.practiced = true

	// Clear the nearest passed row first. A post on the same row may offer a
	// reachable firing lane even when a neighboring gun is behind solid tiles.
	var candidates [9]*WorldActor
	count := 0
	for _, actor := range w.Actors {
		if !actor.Active || actor.thirdCannon == nil || w.ScrollY >= actor.thirdCannon.WorldY+200 || w.ScrollY <= actor.thirdCannon.WorldY-496 {
			continue
		}
		if count == len(candidates) {
			break
		}
		position := count
		for position > 0 {
			other := candidates[position-1]
			nearer := actor.thirdCannon.WorldY > other.thirdCannon.WorldY || actor.thirdCannon.WorldY == other.thirdCannon.WorldY && absDemo(w.Player.X-actor.thirdCannon.X-32) < absDemo(w.Player.X-other.thirdCannon.X-32)
			if !nearer {
				break
			}
			candidates[position] = other
			position--
		}
		candidates[position] = actor
		count++
	}
	for _, actor := range candidates[:count] {
		firingY := actor.thirdCannon.WorldY + 136
		firingX := actor.thirdCannon.X + 32
		x, y, found := n.pointWaypoint(w, firingX, firingY)
		if !found {
			continue
		}
		comfort := 136
		if absDemo(w.Player.X-firingX) < 16 && w.Player.Y+w.ScrollY < firingY+32 {
			comfort = 176
		}
		motion := demoRouteMotionWithOptions(w, x, y, comfort, true)
		if absDemo(w.Player.X-firingX) < 9 && w.Player.Y+w.ScrollY < firingY+36 && w.Player.Y >= 176 {
			motion.Down = true
		}
		return Input{Motion: motion, Fire: (w.Frame+1)%2 != 0 && !w.blockedFireUntilRelease}, true
	}

	n.targetX = 0
	x, y, found := n.waypoint(w, w.ScrollY+w.Player.Y-192)
	if !found {
		return Input{}, false
	}
	return Input{Motion: demoRouteMotionWithOptions(w, x, y, 136, true), Fire: (w.Frame+1)%2 != 0 && !w.blockedFireUntilRelease}, true
}
