package engine

var playerShadowAnimationIDs = [4]string{"player-shadow-0", "player-shadow-1", "player-shadow-2", "player-shadow-3"}

// PlayerShadowSprite resolves the static image selected by a thrust ordinal.
// Animation IDs and atlas image names are separate parts of the export schema.
func (w *World) PlayerShadowSprite(counter int) string {
	if counter < 0 || counter >= len(playerShadowAnimationIDs) {
		return ""
	}
	// Exported IDs follow the constructor's image table, while its stored
	// ordinal counts down from three to zero as that table advances.
	animation := w.commonAnimations[playerShadowAnimationIDs[3-counter]].Animation
	if len(animation.Frames) == 0 {
		return ""
	}
	return animation.Frames[0].Sprite
}

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
