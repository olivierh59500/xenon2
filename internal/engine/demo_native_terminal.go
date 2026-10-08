package engine

// commandClearTerminal aligns a nearby final target while the ship is at the
// native upper bound. Forward camera time is exact here, and the independent
// horizontal graph supplies its shortest release sequence. Every staged input
// still runs the full source motion/rewind and terrain checks before admission.
func (p *nativeMotionPlanner) commandClearTerminal(w *World, n *demoNavigation, x, worldY int) (MotionInput, bool) {
	if input, ok := p.continueRoute(w, x, worldY); ok {
		return input, true
	}
	if !nativeMotionSupported(w) || w.Level.Number != 3 || w.Player.Y != 16 || w.Player.Inertia != 0 || w.BaseScrollStep != 1 || w.Dive.Phase != 0 || absDemo(w.Player.X-x) > 6 || n == nil || n.world != w {
		return MotionInput{}, false
	}
	passes := w.Player.Y + w.ScrollY - worldY
	if passes <= 0 || passes > nativeMotionDepthLimit || !n.clearSegment(w.Player.X, w.Player.Y+w.ScrollY, demoNavPoint{x, worldY}) {
		return MotionInput{}, false
	}
	horizontal, ok := p.terminalHorizontal.distance(w.Player.X, w.Player.Inertia, x, w.Equipment.SpeedTier)
	if !ok || horizontal > passes {
		return MotionInput{}, false
	}
	var commands [nativeMotionDepthLimit]MotionInput
	var states [nativeMotionDepthLimit + 1]demoMotionForecast
	state := newDemoMotionForecast(w)
	states[0] = state
	count := 0
	stage := func(input MotionInput) bool {
		next := state
		if !next.advance(w, input) || next.rewind.Timer != 0 || next.player.Y != 16 || !n.clearSegment(state.player.X, state.player.Y+state.scroll.Y, demoNavPoint{next.player.X, next.player.Y + next.scroll.Y}) {
			return false
		}
		commands[count] = input
		count++
		states[count] = next
		state = next
		return true
	}
	for range passes - horizontal {
		if !stage(MotionInput{}) {
			return MotionInput{}, false
		}
	}
	for remaining := horizontal; remaining > 0; remaining-- {
		selected := false
		for _, input := range nativeHorizontalInputs {
			player := state.player
			player.Advance(input, MotionContext{ScrollY: state.scroll.Y, VisitedScrollY: state.scroll.Maximum, BaseScrollStep: w.BaseScrollStep})
			distance, ok := p.terminalHorizontal.distance(player.X, player.Inertia, x, w.Equipment.SpeedTier)
			if ok && distance == remaining-1 && stage(input) {
				selected = true
				break
			}
		}
		if !selected {
			return MotionInput{}, false
		}
	}
	if state.player.X != x || state.player.Y+state.scroll.Y != worldY {
		return MotionInput{}, false
	}
	p.world, p.targetX, p.targetY, p.at = w, x, worldY, 0
	p.commands = append(p.commands[:0], commands[:count]...)
	p.states = append(p.states[:0], states[:count+1]...)
	p.tiles = append(p.tiles[:0], w.Coverage.Map...)
	p.expanded = 0
	p.nodes, p.queue = p.nodes[:0], p.queue[:0]
	return p.continueRoute(w, x, worldY)
}
