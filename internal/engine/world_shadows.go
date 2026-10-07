package engine

// advancePlayerShadows runs after the dedicated ship in the player phase. The
// four static silhouettes retain allocation order and their word-30 ordinal.
func (w *World) advancePlayerShadows(input Input) error {
	for i, binding := range w.poolShadows {
		state := &w.Shadows[i]
		wasVisible := state.Visible
		state.Counter = int(binding.Residue.VerticalFraction)
		state.PreviousX, state.PreviousY = state.X, state.Y
		if err := StepPlayerShadow(state, w.Player, input.Motion, w.Dive.Phase != 0); err != nil {
			return err
		}
		if !wasVisible {
			state.PreviousX, state.PreviousY = state.X, state.Y
		}
		w.poolShadows[i].Residue.X, w.poolShadows[i].Residue.Y = int16(state.X), int16(state.Y)
		w.storeWorldResidue(w.poolShadows[i])
	}
	return nil
}
