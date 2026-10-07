package engine

// thirdChainPreparation chooses the opposite lane before a source chain reaches
// its player-height activation band. The ordinary motion planner still handles
// terrain, overlapping chains and approaching enemy bodies.
func thirdChainPreparation(w *World) (int, int, bool) {
	nearest, x := 65, 0
	for _, actor := range w.Actors {
		if !actor.Active || actor.thirdChainPart != 1 || actor.thirdChain == nil {
			continue
		}
		head := actor.thirdChain.Parts[0]
		distance := absDemo(head.Y - w.Player.Y)
		if distance < nearest {
			nearest, x = distance, head.X+160
			if actor.thirdChain.Variant != 0 {
				x = 128
			}
		}
	}
	if x == 0 || w.Coverage != nil && w.Level.PlayerStencil != nil && w.Coverage.Touches(x, 120, w.ScrollY, *w.Level.PlayerStencil) {
		return 0, 0, false
	}
	return x, 120, true
}
