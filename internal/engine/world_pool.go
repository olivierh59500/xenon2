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

func (w *World) releaseWorldActor(binding ActorPoolBinding) {
	if slot := w.Pool.Slot(binding.Slot); slot != nil && slot.allocated && slot.EntityID == binding.EntityID {
		delete(w.poolBindings, binding.EntityID)
		if err := w.Pool.Release(binding.Slot); err != nil {
			w.poolError = err
		}
		w.clearPoolReferences(binding.Slot)
	}
}

// Equipment removal visits the current head first and releases a cannon's
// projectile support before its owner, matching the source's direct removal.
func (w *World) releaseEquipmentActors() {
	for index := w.Pool.First(ActorPoolEquipment); index != NoActorSlot; {
		next := w.Pool.Next(index)
		slot := w.Pool.Slot(index)
		if slot.ResourceTag == 52 {
			for support := w.Pool.First(ActorPoolProjectile); support != NoActorSlot; support = w.Pool.Next(support) {
				n := w.Pool.Slot(support)
				if n.ResourceTag == 64 && n.Residue.OwnerSlot == index {
					w.releaseWorldActor(ActorPoolBinding{Slot: support, EntityID: n.EntityID})
					break
				}
			}
		}
		w.releaseWorldActor(ActorPoolBinding{Slot: index, EntityID: slot.EntityID})
		index = next
	}
}

func (w *World) moveWorldActor(binding ActorPoolBinding, list ActorPoolList, tail bool) error {
	if slot := w.Pool.Slot(binding.Slot); slot != nil && slot.allocated && slot.EntityID == binding.EntityID {
		return w.Pool.Move(binding.Slot, list, tail)
	}
	return nil
}

// Saving the equipment head retains pending dead entries as well as installed
// records. The source restores the entire list before its next removal pass.
func (w *World) moveEquipmentActors(from, to ActorPoolList) error {
	for index := w.Pool.First(from); index != NoActorSlot; {
		next := w.Pool.Next(index)
		if err := w.Pool.Move(index, to, true); err != nil {
			return err
		}
		index = next
	}
	return nil
}

// Laser owner pointers refer to physical memory even after the equipment has
// been released. The slot's residue survives release and subsequent reuse.
func (w *World) readWeaponOwnerResidue(index int) (ActorResidue, bool) {
	if slot := w.Pool.Slot(index); slot != nil {
		return slot.Residue, true
	}
	return ActorResidue{}, false
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
	if actor.firstSegment > 0 {
		w.storeFirstGuardianSegmentResidue(actor)
		return
	}
	if actor.fixedAiming != nil {
		w.storeFixedAimingResidue(actor)
		return
	}
	if actor.fixedTileState != nil {
		w.storeFixedTileResidue(actor)
		return
	}
	if actor.secondNode != nil {
		w.storeSecondNodeResidue(actor)
		return
	}
	if actor.thirdChainPart > 0 {
		w.storeThirdChainResidue(actor)
		return
	}
	if actor.fifthIndex > 0 {
		w.storeFifthGuardianResidue(actor)
		return
	}
	if actor.fifthColumn != nil {
		w.storeFifthColumnResidue(actor)
		return
	}
	if actor.thirdCannon != nil {
		w.storeThirdCannonResidue(actor)
		return
	}
	if actor.thirdCrawler != nil {
		w.storeThirdCrawlerResidue(actor)
		return
	}
	if actor.fixedHatch != nil || actor.fixedPod != nil || actor.hatchCreature != nil || actor.podCreature != nil {
		w.storeSecondSpecializedResidue(actor)
		return
	}
	if actor.invulnerability != nil {
		w.storeInvulnerabilityResidue(actor)
		return
	}
	if actor.fifthFormation != nil {
		w.storeFifthFormationResidue(actor)
		return
	}
	if actor.fixedKind != nil {
		w.storeFixedSpriteResidue(actor)
		return
	}
	if actor.fifthSeeking != nil {
		w.storeFifthSeekingResidue(actor)
		return
	}
	if actor.fourthFalling != nil || actor.fourthPod != nil || actor.fourthChild != nil {
		w.storeFourthStageResidue(actor)
		return
	}
	if actor.fifthTile != nil {
		w.storeFifthTileResidue(actor)
		return
	}
	if actor.fourthCrawler != nil {
		w.storeFourthCrawlerResidue(actor)
		return
	}
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
	if actor.thirdChainSentinel {
		// Native chain markers only maintain aggregate bounds and list links;
		// their display state does not overwrite retained gameplay words.
		w.storeWorldResidue(actor.Binding)
		if !actor.Active {
			w.retireWorldActor(actor.Binding)
		}
		return
	}
	r := &actor.Binding.Residue
	r.X, r.Y = int16(actor.X), int16(actor.Y)
	if actor.part != nil && actor.part.MotionMode == "finite-effect" {
		// Common explosions only animate. Their constructors and updater leave
		// health, rewards, fractions and emitter state in physical memory.
		w.storeWorldResidue(actor.Binding)
		if !actor.Active {
			w.retireWorldActor(actor.Binding)
		}
		return
	}
	r.Health, r.PowerOrScore = uint16(actor.Health), uint16(actor.Score)
	r.WaveBonusToken = actor.WaveToken
	r.XFraction, r.YFraction = uint16(actor.motion.X), uint16(actor.motion.Y)
	if actor.part != nil && (actor.part.MotionMode == "path" || actor.part.MotionMode == "path-heading-frames" || actor.part.MotionMode == "path-entry-edge-frames" || actor.part.MotionMode == "follow-leader") {
		r.SetFireState(actor.fire.Accumulator, actor.fire.Rate)
		r.Counter, r.MotionBudget = int16(actor.motion.Remaining), int16(actor.motion.Budget)
		r.Direction, r.HorizontalDriftRemainder = int16(uint16(actor.motion.AngleFixed)), uint16(uint32(actor.motion.AngleFixed)>>16)
		if actor.motion.curveStarted {
			// The curve's velocity low word and acceleration share the words
			// later used as mount offsets by other owners of this slot.
			r.MountOffsetX = int16(actor.motion.AngularVelocity)
			r.MountOffsetY = int16(actor.motion.AngularAcceleration)
		}
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
	if actor.snapDisplayHistory {
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		for i := range actor.Extras {
			actor.Extras[i].PreviousX, actor.Extras[i].PreviousY = actor.Extras[i].X, actor.Extras[i].Y
		}
		for i := range actor.TileOverlays {
			actor.TileOverlays[i].PreviousX, actor.TileOverlays[i].PreviousY = actor.TileOverlays[i].X, actor.TileOverlays[i].Y
		}
		actor.snapDisplayHistory = false
	}
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
	return w.prepareCheckpointPool(true)
}

func (w *World) prepareCheckpointPool(rebuildEquipment bool) error {
	if w.Weapons != nil && w.Weapons.savedMountsActive {
		if err := w.Weapons.restoreSavedMounts(w.weaponContext(Input{}, false)); err != nil {
			return err
		}
	}
	keep := make(map[int]bool, len(w.Actors)+4)
	for _, binding := range w.poolShadows {
		keep[binding.EntityID] = true
	}
	for _, actor := range w.Actors {
		keep[actor.ID] = true
	}
	// The outgoing turn restores Nashwan's saved list before visiting scenery,
	// projectiles, moving actors and finally equipment. Released slots form the
	// next turn's free stack in that order.
	for _, list := range []ActorPoolList{ActorPoolScenery, ActorPoolProjectile, ActorPoolMoving} {
		for index := w.Pool.First(list); index != NoActorSlot; {
			next := w.Pool.Next(index)
			slot := w.Pool.Slot(index)
			cannonSupport := false
			if list == ActorPoolProjectile && slot.ResourceTag == 64 {
				owner := w.Pool.Slot(slot.Residue.OwnerSlot)
				cannonSupport = owner != nil && owner.allocated && owner.list == ActorPoolEquipment && owner.ResourceTag == 52
			}
			if !keep[slot.EntityID] && !cannonSupport {
				w.releaseWorldActor(ActorPoolBinding{Slot: index, EntityID: slot.EntityID})
			}
			index = next
		}
	}
	w.releaseEquipmentActors()
	if w.Weapons != nil {
		w.Weapons.resetMounts()
		w.Weapons.ResetProjectiles()
		if rebuildEquipment {
			return w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false))
		}
	}
	return nil
}
