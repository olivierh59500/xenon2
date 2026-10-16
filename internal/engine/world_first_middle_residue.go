package engine

// The stream's path anchor and followers own their own fractional coordinates;
// generic moving-wave storage has a different state and would clear them.
func (w *World) storeFirstMiddleResidue(actor *WorldActor) {
	r := &actor.Binding.Residue
	if state := actor.firstMiddleAnchor; state != nil {
		r.X, r.Y = int16(state.Motion.X>>16), int16(state.Motion.Y>>16)
		r.XFraction, r.YFraction = uint16(state.Motion.X), uint16(state.Motion.Y)
		r.Counter, r.MotionBudget = int16(state.Motion.Remaining), int16(state.Motion.Budget)
		r.Direction = int16(uint16(state.Motion.AngleFixed))
		r.HorizontalDriftRemainder = uint16(uint32(state.Motion.AngleFixed) >> 16)
	} else if state := actor.firstMiddleFollower; state != nil {
		r.X, r.Y = int16(state.X>>16), int16(state.Y>>16)
		r.XFraction, r.YFraction = uint16(state.X), uint16(state.Y)
		r.Counter = int16(state.Remaining)
		r.Direction = int16(uint16(state.AngleFixed))
		r.HorizontalDriftRemainder = uint16(uint32(state.AngleFixed) >> 16)
		r.Health, r.PowerOrScore = uint16(actor.Health), uint16(actor.Score)
	}
	// Invisible list markers keep their unused words until later slot reuse.
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}
