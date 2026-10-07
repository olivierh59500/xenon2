package engine

// ResetBackgroundStars is the source presentation-restoration boundary. It
// consumes the same shared random stream as actors, pickups and shop dialogue.
func (w *World) ResetBackgroundStars() {
	stars := NewBackgroundStarfield(&w.random)
	w.BackgroundStars = &stars
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
	}
}
