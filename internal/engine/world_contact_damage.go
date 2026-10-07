package engine

// damageActorAt keeps the attacking collider available to source callbacks
// whose armored borders depend on its extent, including ship and Shades hits.
func (w *World) damageActorAt(actor *WorldActor, amount uint16, area CollisionRect) {
	if actor.fourthIndex > 0 {
		w.damageFourthGuardian(actor, area, amount)
		return
	}
	w.damageActor(actor, amount)
}
