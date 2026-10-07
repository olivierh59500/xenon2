package engine

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
