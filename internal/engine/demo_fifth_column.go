package engine

// demoActorHazard includes the fifth guardian's projectile-list columns. Their
// growing collision rectangles are not ordinary moving-list sprite actors.
func demoActorHazard(actor *WorldActor) bool {
	return actor != nil && actor.Active && (actor.fifthColumn != nil || !actor.Collision.Empty() && (actor.ActorList == "moving" || actor.ActorList == "scenery"))
}

// demoFifthColumnPrediction copies the source growth, travel and expiry rules.
// Like ordinary projectile prediction, it holds the current scroll delta. It
// does not predict future camera reversals or intervening player contact.
func demoFifthColumnPrediction(w *World, actor *WorldActor, passes int) (demoActorView, bool) {
	if w == nil || actor == nil || actor.fifthColumn == nil || w.fifthMiddleArt == nil || passes < 0 || passes > 18 {
		return demoActorView{}, false
	}
	state := *actor.fifthColumn
	for range passes {
		state.Advance(w.ScrollDelta, w.fifthMiddleArt.MotionParameters["body_laser_speed"])
	}
	return demoActorView{X: state.X, Y: state.Y, Bounds: state.Collision, Active: actor.Active && state.Active, Visible: state.Active}, true
}
