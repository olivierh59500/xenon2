package engine

// guardianRectImpactUseful follows native moving-list order, linked-group
// suppression and laser absorption. Damage eligibility currently covers the
// fourth/fifth guardians; other collider callbacks conservatively block a
// single-hit weapon without claiming useful damage.
func guardianRectImpactUseful(w *World, impact WeaponRectImpact) bool {
	if w == nil || impact.Area.Empty() || impact.Damage == 0 {
		return false
	}
	var groups [ActorPoolCapacity]int
	count := 0
	useful := false
	var storage [ActorPoolCapacity]*WorldActor
	for _, actor := range w.orderedMovingActors(&storage) {
		if !actor.Active || actor.ActorList != "moving" || !actor.Collision.Intersects(impact.Area) {
			continue
		}
		leader := actor
		if actor.leader != nil && actor.part.DamageMode == "group" {
			leader = actor.leader
		}
		seen := false
		for _, id := range groups[:count] {
			seen = seen || id == leader.ID
		}
		if seen {
			continue
		}
		if count < len(groups) {
			groups[count], count = leader.ID, count+1
		}
		useful = useful || guardianActorAcceptsImpact(w, actor, impact.Area, impact.Damage)
		if !impact.AllTargets || impact.Laser && w.laserCallbackConsumes(actor, impact.Area) {
			return useful
		}
	}
	return useful
}

func guardianActorAcceptsImpact(w *World, actor *WorldActor, area CollisionRect, damage uint16) bool {
	if damage == 0 {
		return false
	}
	if index := actor.fourthIndex - 1; index >= 0 {
		if actor.fourthFinal && w.FourthFinal != nil {
			before := *w.FourthFinal
			after := before
			after.Strike(index, damage, w.ScrollY)
			for part := range before.Parts {
				if after.Parts[part].Health != before.Parts[part].Health {
					return true
				}
			}
		} else if !actor.fourthFinal && w.FourthMiddle != nil {
			before := *w.FourthMiddle
			after := before
			// The native callback rejects a rectangle touching either armored
			// satellite border even when it also intersects the interior.
			after.Strike(index, area, damage)
			for part := range before.Parts {
				if after.Parts[part].Health != before.Parts[part].Health {
					return true
				}
			}
		}
	}
	if index := actor.fifthIndex - 1; index >= 0 {
		if actor.fifthFinal && w.FifthFinal != nil {
			before := *w.FifthFinal
			after := before
			after.DamagePart(w.fifthFinalArt, index, damage)
			if after.CoreHealth != before.CoreHealth {
				return true
			}
			for part := range before.Parts {
				if after.Parts[part].Health != before.Parts[part].Health {
					return true
				}
			}
		} else if !actor.fifthFinal && w.FifthMiddle != nil {
			before := *w.FifthMiddle
			after := before
			after.Damage(index, damage)
			for part := range before.Parts {
				if after.Parts[part].Health != before.Parts[part].Health {
					return true
				}
			}
		}
	}
	return false
}
