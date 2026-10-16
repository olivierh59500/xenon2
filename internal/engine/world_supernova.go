package engine

// sourceMovingGroupSuccessor reproduces the group-aware saved-next traversal.
// Linked pieces share the first part's owner slot and are visited as one target.
func (w *World) sourceMovingGroupSuccessor(index int) int {
	slot := w.Pool.Slot(index)
	next := w.Pool.Next(index)
	if !slot.Linked {
		return next
	}
	owner := slot.Residue.OwnerSlot
	if w.Pool.Slot(owner) == nil {
		return next
	}
	next = w.Pool.Next(owner)
	for next != NoActorSlot {
		candidate := w.Pool.Slot(next)
		if !candidate.Linked || candidate.Residue.OwnerSlot != index {
			break
		}
		next = w.Pool.Next(next)
	}
	return next
}

func (w *World) forEachSupernovaTarget(hit func(*WorldActor)) {
	if w.Pool != nil {
		for index := w.Pool.First(ActorPoolMoving); index != NoActorSlot; {
			next := w.sourceMovingGroupSuccessor(index)
			slot := w.Pool.Slot(index)
			if slot.ResourceTag != 80 && slot.ResourceTag != 84 {
				actor := w.poolActors[index]
				if actor != nil && actor.Binding.EntityID == slot.EntityID {
					hit(actor)
				}
			}
			index = next
		}
	}
	for _, actor := range w.Actors {
		if actor.Binding.EntityID == 0 && actor.ActorList == "moving" && actor.part != nil && actor.part.ResourceTag != 80 && actor.part.ResourceTag != 84 && !(actor.part.Linked && actor.leader != nil) {
			hit(actor)
		}
	}
}

func (w *World) finishSupernova() {
	w.forEachSupernovaTarget(func(actor *WorldActor) {
		if actor.Active {
			w.damageActor(actor, 127)
		}
	})
	if w.Pool != nil {
		for index := w.Pool.First(ActorPoolProjectile); index != NoActorSlot; {
			next := w.Pool.Next(index)
			slot := w.Pool.Slot(index)
			if slot.ResourceTag == 20 {
				if shot := w.poolProjectiles[index]; shot != nil && shot.ID == slot.EntityID {
					shot.Active = false
				}
				if err := w.Pool.MarkDead(index); err != nil {
					w.poolError = err
				}
			}
			index = next
		}
	}
	for _, shot := range w.Projectiles {
		if shot.Binding.EntityID == 0 {
			shot.Active = false
		}
	}
	clear(w.WaveBonuses.Entries[:])
}
