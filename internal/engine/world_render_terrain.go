package engine

// captureRenderTerrain retains the map drawn before stage and actor callbacks.
// Combat keeps using the live mutable map; presentation uses this reusable copy.
func (w *World) captureRenderTerrain() {
	if len(w.RenderTerrainMap) != len(w.Level.Terrain.Map) {
		w.RenderTerrainMap = make([]uint16, len(w.Level.Terrain.Map))
	}
	copy(w.RenderTerrainMap, w.Level.Terrain.Map)
}

// captureActorRenderTerrain records the live masks used by the original draw
// callbacks after updates, before timers, encounter constructors and scrolling.
func (w *World) captureActorRenderTerrain() {
	if len(w.ActorRenderTerrainMap) != len(w.Level.Terrain.Map) {
		w.ActorRenderTerrainMap = make([]uint16, len(w.Level.Terrain.Map))
	}
	copy(w.ActorRenderTerrainMap, w.Level.Terrain.Map)
	w.ActorRenderScrollY = w.ScrollY
}
