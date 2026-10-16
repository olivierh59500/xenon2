package engine

// countSourceMovingActors runs before updater traversal, so tombstones still
// count until their original list's cleanup pass releases them.
func (w *World) countSourceMovingActors() {
	w.MovingEnemyCount = 0
	if w.Pool != nil {
		for index := w.Pool.First(ActorPoolMoving); index != NoActorSlot; {
			slot := w.Pool.Slot(index)
			if slot.ResourceTag != 80 && slot.ResourceTag != 84 {
				w.MovingEnemyCount++
			}
			next := w.Pool.Next(index)
			if slot.Linked {
				owner := slot.Residue.OwnerSlot
				if owner >= 0 && owner < ActorPoolCapacity {
					next = w.Pool.Next(owner)
					for next != NoActorSlot {
						candidate := w.Pool.Slot(next)
						if !candidate.Linked || candidate.Residue.OwnerSlot != index {
							break
						}
						next = w.Pool.Next(next)
					}
				}
			}
			index = next
		}
	}
	for _, actor := range w.Actors {
		if actor.Active && actor.Binding.EntityID == 0 && actor.ActorList == "moving" && actor.part.ResourceTag != 80 && actor.part.ResourceTag != 84 && !(actor.part.Linked && actor.leader != nil) {
			w.MovingEnemyCount++
		}
	}
}

// advanceStageBeforeActors consumes the preceding pass's stream/heartbeat flags.
// Stream births made here enter the current updater traversal; ordinary encounter
// births remain after actor updates and therefore start on the following pass.
func (w *World) advanceStageBeforeActors() error {
	w.countSourceMovingActors()
	if err := w.advanceFirstMiddleStage(); err != nil {
		return err
	}
	if err := w.advanceThirdStage(); err != nil {
		return err
	}
	if err := w.advanceSecondDefenseWaves(); err != nil {
		return err
	}
	if err := w.advanceFourthStage(); err != nil {
		return err
	}
	w.secondStreamsUpdated = [2]bool{}
	w.thirdMiddleUpdated, w.thirdFinalUpdated = false, false
	return nil
}
