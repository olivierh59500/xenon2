package engine

// retireDeathEquipment follows the current physical equipment head. The source
// skips only an unmodified basic gun, then retires one entry and its cannon
// support. Checkpoint loadout and dormant Nashwan equipment remain unchanged.
func (w *World) retireDeathEquipment() {
	if w.Weapons == nil || w.Pool == nil {
		return
	}
	index := w.Pool.First(ActorPoolEquipment)
	if index == NoActorSlot {
		return
	}
	slot := w.Pool.Slot(index)
	if slot.ResourceTag == 148 && slot.Residue.PowerOrScore == 0 {
		index = w.Pool.Next(index)
		if index == NoActorSlot {
			return
		}
		slot = w.Pool.Slot(index)
	}
	if slot.ResourceTag == 52 {
		for support := w.Pool.First(ActorPoolProjectile); support != NoActorSlot; support = w.Pool.Next(support) {
			entry := w.Pool.Slot(support)
			if entry.ResourceTag == 64 && entry.Residue.OwnerSlot == index {
				w.retireWorldActor(ActorPoolBinding{Slot: support, EntityID: entry.EntityID})
				w.Weapons.DropActor(entry.EntityID)
				break
			}
		}
	}
	// An already retired head has no live catalogue entry to clear.
	if slot.ResourceTag != 4 {
		for position := range w.Weapons.mounts {
			mount := &w.Weapons.mounts[position]
			if mount.Binding.EntityID != slot.EntityID {
				continue
			}
			*w.Equipment.slots()[position] = WeaponSlot{}
			mount.Item, mount.Serial, mount.Visible = ItemNone, 0, false
			break
		}
	}
	w.retireWorldActor(ActorPoolBinding{Slot: index, EntityID: slot.EntityID})
	w.Equipment.EnsureBasicWeapon()
	if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
		w.poolError = err
	}
}
