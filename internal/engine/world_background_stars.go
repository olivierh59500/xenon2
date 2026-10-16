package engine

// ResetBackgroundStars is the source presentation-restoration boundary. It
// consumes the same shared random stream as actors, pickups and shop dialogue.
func (w *World) ResetBackgroundStars() {
	stars := NewBackgroundStarfield(&w.random)
	w.BackgroundStars = &stars
	w.renderedBackgroundStars = stars
	w.renderedBackgroundY = w.BackgroundY
}

// PrimeBackgroundStars reproduces initial backdrop drawing passes without
// advancing combat, terrain scrolling, actor timers or encounter activation.
func (w *World) PrimeBackgroundStars(passes int) {
	for range max(0, passes) {
		w.advanceBackgroundStars()
	}
}

func (w *World) advanceBackgroundStars() {
	if w.BackgroundStars != nil {
		w.BackgroundStars.Advance(w.ScrollDelta)
		for index, star := range w.BackgroundStars.Stars {
			star.PreviousY = w.renderedBackgroundStars.Stars[index].Y
			w.renderedBackgroundStars.Stars[index] = star
		}
	}
}

// RenderedBackgroundStars retains the points drawn before stage callbacks
// advance the live backdrop for the following pass.
func (w *World) RenderedBackgroundStars() *BackgroundStarfield {
	if w.BackgroundStars == nil {
		return nil
	}
	return &w.renderedBackgroundStars
}

// RenderedBackgroundY is the offset copied before late stage preparation.
func (w *World) RenderedBackgroundY() int {
	if w.Frame == 0 {
		return w.BackgroundY
	}
	return w.renderedBackgroundY
}
