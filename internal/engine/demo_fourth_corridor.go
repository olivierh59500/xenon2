package engine

const (
	fourthForkStartCamera = 4160
	fourthForkEndCamera   = 3968
	fourthForkTargetX     = 100
	fourthForkTargetY     = 4140
)

// fourthCorridorActive covers the original forest fork before the right-hand
// pocket closes. Its source stage prelude keeps ordinary scroll bounds here;
// the later falling-actor region and middle arena remain separate.
func fourthCorridorActive(w *World) bool {
	return w != nil && w.Level.Number == 4 && w.FourthMiddle == nil && w.ScrollY <= fourthForkStartCamera && w.ScrollY > fourthForkEndCamera
}

// FourthCorridorInput prepares the known left exit using original terrain and
// committed native commands. A moving geometric target alone can postpone the
// braking turn or carry the ship into the right-hand pocket.
func (p *DemoPilot) FourthCorridorInput(w *World) (Input, bool) {
	if !p.practicedRoute || !fourthCorridorActive(w) || w.Ready || !w.PlayerAlive || w.GameOver || w.Rewind.Timer != 0 || w.Coverage == nil || w.Level.PlayerStencil == nil {
		return Input{}, false
	}
	if w.Player.X <= 140 && w.Player.Y+w.ScrollY <= fourthForkTargetY {
		return Input{}, false
	}
	if motion, ok := p.nativeMotion.continueRoute(w, fourthForkTargetX, fourthForkTargetY); ok {
		return Input{Motion: motion}, true
	}
	if p.navigation == nil {
		p.navigation = &demoNavigation{}
	}
	p.navigation.practiced = true
	x, y, found := p.navigation.pointWaypoint(w, fourthForkTargetX, fourthForkTargetY)
	if !found {
		return Input{}, false
	}
	motion, found := p.nativeMotion.command(w, x, y, fourthForkTargetX, fourthForkTargetY)
	if !found {
		return Input{}, false
	}
	return Input{Motion: motion}, true
}
