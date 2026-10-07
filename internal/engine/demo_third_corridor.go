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

	for _, actor := range w.Actors {
		if actor.Active && actor.thirdCannon != nil && actor.thirdCannon.X == 224 && (actor.thirdCannon.WorldY == 1696 && w.ScrollY < 2150 && w.ScrollY > 1200 || actor.thirdCannon.WorldY == 1328 && w.ScrollY < 1750 && w.ScrollY > 1100) {
			firingY := actor.thirdCannon.WorldY + 136
			x, y, found := n.pointWaypoint(w, 256, firingY)
			if !found {
				return Input{}, false
			}
			comfort := 136
			if w.Player.X >= 240 && w.Player.Y+w.ScrollY < firingY+32 {
				comfort = 176
			}
			motion := demoRouteMotionWithOptions(w, x, y, comfort, true)
			if absDemo(w.Player.X-256) < 9 && w.Player.Y+w.ScrollY < firingY+36 && w.Player.Y >= 176 {
				motion.Down = true
			}
			return Input{Motion: motion, Fire: (w.Frame+1)%2 != 0 && !w.blockedFireUntilRelease}, true
		}
	}
	n.targetX = 0
	x, y, found := n.waypoint(w, w.ScrollY+w.Player.Y-192)
	if !found {
		return Input{}, false
	}
	return Input{Motion: demoRouteMotionWithOptions(w, x, y, 136, true), Fire: (w.Frame+1)%2 != 0 && !w.blockedFireUntilRelease}, true
}
