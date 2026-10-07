package engine

func (w *World) initializeWorldPool() error {
	w.Pool = NewActorPool()
	w.poolBindings = make(map[int]ActorPoolBinding, ActorPoolCapacity)
	for i := range w.poolShadows {
		binding, err := w.reserveWorldActor(196, ActorPoolPlayer, true)
		if err != nil {
			return err
		}
		w.poolShadows[i] = binding
		w.poolShadows[i].Residue.VerticalFraction = uint16(3 - i)
		w.storeWorldResidue(w.poolShadows[i])
	}
	if w.Weapons != nil {
		return w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false))
	}
	return nil
}

// reserveWorldActor preserves physical-slot identity independently of creation
// order. Stealing discards state without damage, score or reward callbacks.
func (w *World) reserveWorldActor(tag int16, list ActorPoolList, tail bool) (ActorPoolBinding, error) {
	allocation, err := w.Pool.Allocate()
	if err != nil {
		return ActorPoolBinding{Slot: NoActorSlot}, err
	}
	if allocation.Stolen {
		w.discardWorldEntity(allocation.PreviousEntityID)
	}
	w.poolActors[allocation.Slot] = nil
	w.poolProjectiles[allocation.Slot] = nil
	w.poolSmallShots[allocation.Slot] = nil
	w.poolCollectibles[allocation.Slot] = nil
	w.nextActorID++
	slot := w.Pool.Slot(allocation.Slot)
	binding := ActorPoolBinding{Slot: allocation.Slot, EntityID: w.nextActorID, Residue: slot.Residue, AllocationPhase: slot.AllocationPhase}
	if tail {
		err = w.Pool.AttachTail(binding.Slot, list, binding.EntityID, tag)
	} else {
		err = w.Pool.AttachHead(binding.Slot, list, binding.EntityID, tag)
	}
	if err != nil {
		return ActorPoolBinding{Slot: NoActorSlot}, err
	}
	w.poolBindings[binding.EntityID] = binding
	return binding, nil
}

func (w *World) discardWorldEntity(id int) {
	delete(w.poolBindings, id)
	for _, actor := range w.Actors {
		if actor.ID == id {
			actor.Active, actor.Binding.EntityID = false, 0
		}
	}
	for _, shot := range w.Projectiles {
		if shot.ID == id {
			shot.Active, shot.Binding.EntityID = false, 0
		}
	}
	for _, shot := range w.SmallShots {
		if shot.ID == id {
			shot.Active, shot.Binding.EntityID = false, 0
		}
	}
	for _, item := range w.Collectibles {
		if item.ID == id {
			item.Active, item.Binding.EntityID = false, 0
		}
	}
	if w.Weapons != nil {
		w.Weapons.DropActor(id)
	}
}

func (w *World) clearPoolReferences(index int) {
	w.poolActors[index] = nil
	w.poolProjectiles[index] = nil
	w.poolSmallShots[index] = nil
	w.poolCollectibles[index] = nil
}

func (w *World) storeWorldResidue(binding ActorPoolBinding) {
	if slot := w.Pool.Slot(binding.Slot); slot != nil && slot.allocated && slot.EntityID == binding.EntityID {
		slot.Residue = binding.Residue
		w.poolBindings[binding.EntityID] = binding
	}
}

func (w *World) retireWorldActor(binding ActorPoolBinding) {
	if slot := w.Pool.Slot(binding.Slot); slot != nil && slot.allocated && slot.EntityID == binding.EntityID {
		if err := w.Pool.MarkDead(binding.Slot); err != nil {
			w.poolError = err
		}
	}
}

func (w *World) readWorldResidue(index int) (ActorResidue, bool) {
	if slot := w.Pool.Slot(index); slot != nil && slot.allocated {
		return slot.Residue, true
	}
	return ActorResidue{}, false
}

// releaseDeadPoolEntries runs at the start of the owning phase. Entries killed
// by a later phase remain stealable until that phase is visited again.
func (w *World) releaseDeadPoolEntries(list ActorPoolList) {
	for index := w.Pool.First(list); index != NoActorSlot; {
		next := w.Pool.Next(index)
		slot := w.Pool.Slot(index)
		if slot.ResourceTag == 4 {
			delete(w.poolBindings, slot.EntityID)
			if err := w.Pool.Release(index); err != nil {
				w.poolError = err
			}
		}
		index = next
	}
}

func worldActorPoolList(actor *WorldActor) ActorPoolList {
	switch actor.ActorList {
	case "scenery":
		return ActorPoolScenery
	case "transient":
		return ActorPoolProjectile
	default:
		return ActorPoolMoving
	}
}

func (w *World) bindWorldActor(actor *WorldActor) error {
	if actor.Binding.EntityID != 0 {
		return nil
	}
	tag := int16(0)
	if actor.part != nil {
		tag = int16(actor.part.ResourceTag)
	}
	binding, err := w.reserveWorldActor(tag, worldActorPoolList(actor), false)
	if err != nil {
		return err
	}
	actor.ID, actor.Binding = binding.EntityID, binding
	w.poolActors[binding.Slot] = actor
	return nil
}

func (w *World) storeActorResidue(actor *WorldActor) {
	if actor.fourthIndex > 0 {
		w.storeFourthPartResidue(actor)
		if !actor.Active {
			w.retireWorldActor(actor.Binding)
		}
		return
	}
	if actor.Binding.EntityID == 0 {
		return
	}
	r := &actor.Binding.Residue
	r.X, r.Y = int16(actor.X), int16(actor.Y)
	r.Health, r.PowerOrScore = uint16(actor.Health), uint16(actor.Score)
	r.WaveBonusToken = actor.WaveToken
	r.XFraction, r.YFraction = uint16(actor.motion.X), uint16(actor.motion.Y)
	if actor.part != nil && actor.part.MotionMode == "path" {
		r.SetFireState(actor.fire.Accumulator, actor.fire.Rate)
	}
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}

func (w *World) transferWorldActor(actor *WorldActor, list ActorPoolList) {
	if actor.Binding.EntityID == 0 {
		return
	}
	slot := w.Pool.Slot(actor.Binding.Slot)
	if slot == nil || slot.EntityID != actor.ID || slot.list == list {
		return
	}
	tag := slot.ResourceTag
	w.Pool.unlink(actor.Binding.Slot)
	if err := w.Pool.AttachHead(actor.Binding.Slot, list, actor.ID, tag); err != nil {
		w.poolError = err
	}
}

func (w *World) releaseWorldBinding(binding *ActorPoolBinding) {
	if binding.EntityID == 0 {
		return
	}
	if slot := w.Pool.Slot(binding.Slot); slot != nil && slot.allocated && slot.EntityID == binding.EntityID {
		delete(w.poolBindings, binding.EntityID)
		if err := w.Pool.Release(binding.Slot); err != nil {
			w.poolError = err
		}
	}
	binding.EntityID = 0
}

func (w *World) finishActorUpdate(actor *WorldActor) {
	w.storeActorResidue(actor)
}

func (w *World) syncDeadActors() {
	for _, actor := range w.Actors {
		if !actor.Active {
			w.storeActorResidue(actor)
		}
	}
}

func (w *World) finishProjectileUpdate(shot *WorldProjectile) {
	if shot.Binding.EntityID == 0 {
		return
	}
	r := &shot.Binding.Residue
	r.X, r.Y = int16(shot.X), int16(shot.Y)
	motion := shot.Motion
	if shot.turning != nil {
		motion = shot.turning.Motion
	}
	r.XFraction, r.YFraction = uint16(motion.X), uint16(motion.Y)
	r.Direction, r.MotionBudget = int16(motion.Direction), int16(motion.Speed)
	w.storeWorldResidue(shot.Binding)
	if !shot.Active {
		w.retireWorldActor(shot.Binding)
	}
}

func (w *World) finishSmallShotUpdate(shot *WorldSmallShot) {
	if shot.Binding.EntityID == 0 {
		return
	}
	r := &shot.Binding.Residue
	r.X, r.Y = int16(shot.Shot.X), int16(shot.Shot.Y)
	r.Direction, r.VerticalVelocity = int16(shot.Shot.VelocityX), int16(shot.Shot.VelocityY)
	w.storeWorldResidue(shot.Binding)
	if !shot.Active {
		w.retireWorldActor(shot.Binding)
	}
}

func (w *World) finishCollectibleUpdate(item *WorldCollectible) {
	if item.Binding.EntityID == 0 {
		return
	}
	r := &item.Binding.Residue
	r.X, r.Y = int16(item.X), int16(item.Y)
	r.Counter, r.Direction = int16(item.Motion.Mode), int16(item.Motion.Direction)
	w.storeWorldResidue(item.Binding)
	if !item.Active {
		w.retireWorldActor(item.Binding)
	}
}

// restoreCheckpointPool performs the bulk cleanup used between ship turns.
// Surviving scripted actors retain their physical slots and named residues.
func (w *World) restoreCheckpointPool() error {
	keep := make(map[int]bool, len(w.Actors)+4)
	for _, binding := range w.poolShadows {
		keep[binding.EntityID] = true
	}
	for _, actor := range w.Actors {
		keep[actor.ID] = true
	}
	for index := range ActorPoolCapacity {
		slot := w.Pool.Slot(index)
		if slot.allocated && !keep[slot.EntityID] {
			w.poolActors[index] = nil
			delete(w.poolBindings, slot.EntityID)
			if err := w.Pool.Release(index); err != nil {
				return err
			}
		}
	}
	if w.Weapons != nil {
		for i := range w.Weapons.mounts {
			w.Weapons.mounts[i] = runtimeMount{Binding: ActorPoolBinding{Slot: NoActorSlot}, SupportBinding: ActorPoolBinding{Slot: NoActorSlot}}
		}
		w.Weapons.ResetProjectiles()
		return w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false))
	}
	return nil
}
