package engine

// SynchronizeShopEquipment applies successful trades while gameplay is paused.
// Sold records remain retired until normal traversal, preserving their reusable
// words and the allocation order of later purchases in the same visit.
func (w *World) SynchronizeShopEquipment() error {
	if w.Weapons == nil {
		return nil
	}
	return w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false))
}

// ResumeShop preserves the current camera and all living actor states. The
// original return restores weapon records but does not restart the checkpoint.
func (w *World) ResumeShop() {
	w.ShopReady, w.ExitReady = false, false
	w.PendingExitDrops = 0
	w.AdviceIndex = 12
	if w.Equipment.SuperLoadoutActive {
		w.Checkpoint.Loadout = w.Equipment.SavedLoadout
	} else {
		w.Checkpoint.Loadout = w.Equipment.WeaponLoadout
	}
}
