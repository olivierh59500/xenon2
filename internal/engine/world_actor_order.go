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
