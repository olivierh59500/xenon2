package engine

// Completion first performs ordinary turn cleanup, then releases every remaining
// level actor. The four player shadows and all free-record residue survive.
func (w *World) clearCompletedStageActors() error {
	if w.Equipment.SuperLoadoutActive {
		w.Checkpoint.Loadout = w.Equipment.SavedLoadout
	} else {
		w.Checkpoint.Loadout = w.Equipment.WeaponLoadout
	}
	w.suspendTurn()
	if w.poolError != nil {
		return w.poolError
	}
	for _, list := range []ActorPoolList{ActorPoolMoving, ActorPoolScenery, ActorPoolProjectile} {
		for index := w.Pool.First(list); index != NoActorSlot; {
			next := w.Pool.Next(index)
			binding := ActorPoolBinding{Slot: index, EntityID: w.Pool.Slot(index).EntityID}
			w.discardWorldEntity(binding.EntityID)
			w.releaseWorldActor(binding)
			index = next
		}
	}
	w.Actors, w.Projectiles, w.SmallShots, w.Collectibles = nil, nil, nil, nil
	w.firstGuardianActor, w.secondGuardianActor = nil, nil
	return w.poolError
}

// The outgoing current player's saved weapons are reconstructed before the
// next level's actors. READY later restores the admitted player's weapons again.
func (w *World) restoreStageEquipment() error {
	shield, advance := w.Equipment.Shield, w.Equipment.FireAdvance
	w.Equipment.RestoreCheckpointLoadout(w.Checkpoint.Loadout)
	w.Equipment.Shield, w.Equipment.FireAdvance = shield, advance
	if w.Weapons != nil {
		if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
			return err
		}
	}
	w.turnPrepared = false
	return nil
}
