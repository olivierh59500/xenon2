package engine

// secondArenaCrossingGoal follows an edge-checked terrain route around the
// middle structure when the surviving side node requires the opposite half.
// Its dedicated cache leaves ordinary combat navigation untouched. Opening
// the node halfway across does not discard the remaining descent to its lane.
func secondArenaCrossingGoal(w *World, p *DemoPilot) (int, int, bool) {
	if w == nil || p == nil || p.Config.DisableBossAlignment || w.Level.Number != 2 || w.Ready || w.GameOver || !w.PlayerAlive || w.secondScheduler == nil || w.secondMiddleReleased || w.ScrollY < 2512 || w.ScrollY > 2896 || w.Rewind.Timer != 0 {
		return 0, 0, false
	}
	var target *SecondDefenseNodeState
	for _, actor := range w.secondNodes {
		if actor != nil && actor.Active && actor.Health > 0 && actor.secondNode != nil {
			target = actor.secondNode
			break
		}
	}
	if target == nil || target.Index == 2 {
		return 0, 0, false
	}
	tx, ty := target.TileX*16+8, target.TileY*16+40
	n := p.secondCrossingNavigation
	retained := n != nil && n.world == w && n.pointTargetX == tx && n.goal == ty && len(n.path) != 0
	opposite := target.Index == 1 && w.Player.X >= 160 || target.Index == 0 && w.Player.X < 160
	if !retained && !(opposite && w.secondScheduler.DefenseFlags != 3) {
		return 0, 0, false
	}
	if absDemo(w.Player.X-tx) <= 6 && absDemo(w.Player.Y+w.ScrollY-ty) <= 6 {
		if n != nil {
			n.path = n.path[:0]
		}
		return 0, 0, false
	}
	if n == nil {
		n = &demoNavigation{practiced: true}
		p.secondCrossingNavigation = n
	}
	// The native top player bound is sixteen pixels. The normal point search's
	// forty-eight-pixel margin excludes the arena's upper cross-screen opening.
	x, worldY, ok := n.pointWaypointMinimum(w, tx, ty, w.ScrollY+16)
	return x, worldY - w.ScrollY, ok
}
