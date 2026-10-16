package engine

// The boss continuation repeatedly queries the same small map window. These
// bits memoize exact stencil results, never approximate bounding-box contact.
type demoNavTouchCache struct {
	top          int
	known, solid [256][10]uint32
}

func (n *demoNavigation) cacheTouchWindow(camera int) {
	if n.touchCache == nil {
		n.touchCache = &demoNavTouchCache{top: camera - 32}
	} else if camera < n.touchCache.top+16 || camera+192 >= n.touchCache.top+256 {
		n.touchCache.top = camera - 32
		clear(n.touchCache.known[:])
	}
}
