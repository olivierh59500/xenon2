package engine

// The next moving-list contact precedes the next input's movement. Visible is
// not a contact filter; only the first published moving collision is relevant.
func fourthOpeningTerminalContactUnsafe(w *World) bool {
	if w == nil || w.Level.Number != 4 || w.FourthMiddle != nil || w.ScrollY <= 176 || w.Ready || w.GameOver || !w.PlayerAlive || w.Dive.Phase != 0 || w.Equipment.ShadesFrames > 0 {
		return false
	}
	prefix := thirdMiddlePlayerBounds(w, w.Player)
	var order [ActorPoolCapacity]*WorldActor
	for _, actor := range w.orderedMovingActors(&order) {
		if !actor.Active || actor.ActorList != "moving" || !actor.Collision.Intersects(prefix) {
			continue
		}
		if actor.part == nil {
			return false
		}
		damage := ApplyShieldDamage(w.Equipment.Shield, ContactDamage(actor.part.StrongHealth), w.Equipment.Protection, w.InvulnerableFrames != 0 || w.PendingExitDrops != 0 || w.Cheats.InfiniteEnergy)
		return damage.Applied && (damage.Lost > 0 || damage.Destroyed)
	}
	return false
}
