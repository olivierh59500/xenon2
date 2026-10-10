package controls

// GestureInsets reserves the edges where Android can claim an entire contact
// sequence for navigation or notifications. Values use logical view coordinates.
type GestureInsets struct{ Left, Top, Right, Bottom float64 }

// GestureGuard keeps a contact's initial classification until release. An edge
// swipe must not become a game tap when its finger subsequently enters the view.
// Fingers that began on a game control can still drag beyond its original area.
type GestureGuard struct{ contacts map[int]bool }

func (g *GestureGuard) Filter(dst, touches []Touch, width, height float64, insets GestureInsets) []Touch {
	if g.contacts == nil {
		g.contacts = make(map[int]bool)
	}
	for id := range g.contacts {
		found := false
		for _, t := range touches {
			found = found || t.ID == id
		}
		if !found {
			delete(g.contacts, id)
		}
	}
	for _, t := range touches {
		ignored, known := g.contacts[t.ID]
		if t.Pressed || !known {
			ignored = t.X < insets.Left || t.Y < insets.Top || t.X >= width-insets.Right || t.Y >= height-insets.Bottom
			g.contacts[t.ID] = ignored
		}
		if !ignored {
			dst = append(dst, t)
		}
	}
	return dst
}
