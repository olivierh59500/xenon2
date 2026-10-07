package engine

// orderedMovingActors snapshots physical-list order before a damage callback
// can allocate effects or steal another slot. Synthetic actors follow afterward.
func (w *World) orderedMovingActors(storage *[ActorPoolCapacity]*WorldActor) []*WorldActor {
	count := 0
	if w.Pool != nil {
		for index := w.Pool.First(ActorPoolMoving); index != NoActorSlot; index = w.Pool.Next(index) {
			actor := w.poolActors[index]
			if actor != nil && actor.Binding.EntityID == w.Pool.Slot(index).EntityID && count < len(storage) {
				storage[count] = actor
				count++
			}
		}
	}
	for _, actor := range w.Actors {
		if actor.Binding.EntityID == 0 && actor.ActorList == "moving" && count < len(storage) {
			storage[count] = actor
			count++
		}
	}
	return storage[:count]
}

// ActorDrawOrder returns the order of current entities within their owning
// lists. The renderer applies the separate player/equipment/moving/effect layers.
func (w *World) ActorDrawOrder(dst map[int]int) map[int]int {
	if dst == nil {
		dst = make(map[int]int, ActorPoolCapacity)
	}
	clear(dst)
	if w.Pool == nil {
		return dst
	}
	for _, list := range []ActorPoolList{ActorPoolPlayer, ActorPoolEquipment, ActorPoolMoving, ActorPoolProjectile, ActorPoolScenery} {
		order := ActorPoolCapacity
		for index := w.Pool.First(list); index != NoActorSlot; index = w.Pool.Next(index) {
			dst[w.Pool.Slot(index).EntityID] = order
			order--
		}
	}
	return dst
}
