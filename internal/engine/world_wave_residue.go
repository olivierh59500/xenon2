package engine

// initializeWaveActorResidue publishes the common formation constructor's
// writes before another allocation can reuse this physical slot. Spare vertical
// and mount words remain unchanged until their owning callback writes them.
func (w *World) initializeWaveActorResidue(actor, leader *WorldActor) {
	r := &actor.Binding.Residue
	r.X, r.Y = int16(actor.X), int16(actor.Y)
	r.XFraction, r.YFraction = 0, 0
	r.Counter = int16(actor.motion.Remaining)
	r.Direction, r.HorizontalDriftRemainder = 0, 0
	r.Health, r.PowerOrScore = uint16(actor.Health), uint16(actor.Score)
	r.StrongHealth = actor.part.StrongHealth
	r.MotionBudget = int16(actor.motion.Budget)
	r.SetFireState(actor.fire.Accumulator, actor.fire.Rate)
	r.WaveBonusToken = w.WaveBonuses.NormalID
	if actor.part.StrongHealth {
		r.WaveBonusToken = w.WaveBonuses.HeavyID
	}
	r.OwnerSlot, r.FollowingSlot = leader.Binding.Slot, NoActorSlot
	w.storeWorldResidue(actor.Binding)
}
