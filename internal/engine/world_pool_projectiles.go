package engine

// advancePooledProjectiles follows the saved-next pointer of the common source
// list. A new head is not visited in the current traversal; a later dead entry
// is released when reached, while self-removal waits for its next visit.
func (w *World) advancePooledProjectiles(input Input) error {
	context := w.weaponContext(input, false)
	for index := w.Pool.First(ActorPoolProjectile); index != NoActorSlot; {
		next := w.Pool.Next(index)
		slot := w.Pool.Slot(index)
		if slot.ResourceTag == 4 {
			delete(w.poolBindings, slot.EntityID)
			if err := w.Pool.Release(index); err != nil {
				return err
			}
		} else if err := w.advancePoolProjectileEntity(slot.EntityID, context); err != nil {
			return err
		}
		index = next
	}
	return nil
}

func (w *World) advancePoolProjectileEntity(id int, context WeaponContext) error {
	for _, actor := range w.Actors {
		if actor.ID != id || !actor.Active || actor.ActorList != "transient" {
			continue
		}
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		if actor.secondFragment != nil {
			w.advanceSecondFragment(actor)
		} else if actor.animation.Ending == "remove" && actor.animationState.Frame == len(actor.animation.Frames)-1 && actor.animationState.Remaining == 1 {
			actor.Active = false
		} else {
			actor.animationState.Advance(actor.animation)
			actor.selectSprite()
		}
		w.finishActorUpdate(actor)
		return nil
	}
	for _, item := range w.Collectibles {
		if item.ID == id && item.Active {
			w.advanceCollectible(item)
			w.finishCollectibleUpdate(item)
			return nil
		}
	}
	for _, shot := range w.Projectiles {
		if shot.ID == id && shot.Active {
			if err := w.advanceEnemyShot(shot); err != nil {
				return err
			}
			w.finishProjectileUpdate(shot)
			return nil
		}
	}
	for _, shot := range w.SmallShots {
		if shot.ID == id && shot.Active {
			w.advanceSmallShot(shot)
			w.finishSmallShotUpdate(shot)
			return nil
		}
	}
	if w.Weapons != nil {
		context.ShipDestroyed = !w.PlayerAlive
		for i := range context.Targets {
			if actor := w.weaponTargetActors[context.Targets[i].ID]; actor != nil {
				context.Targets[i].Active = actor.Active
			}
		}
		return w.Weapons.AdvanceProjectile(id, context)
	}
	return nil
}

func (w *World) hasUnboundProjectiles() bool {
	for _, actor := range w.Actors {
		if actor.Active && actor.ActorList == "transient" && actor.Binding.EntityID == 0 {
			return true
		}
	}
	for _, shot := range w.Projectiles {
		if shot.Active && shot.Binding.EntityID == 0 {
			return true
		}
	}
	for _, shot := range w.SmallShots {
		if shot.Active && shot.Binding.EntityID == 0 {
			return true
		}
	}
	for _, item := range w.Collectibles {
		if item.Active && item.Binding.EntityID == 0 {
			return true
		}
	}
	return false
}
