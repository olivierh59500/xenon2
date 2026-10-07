package engine

// weaponHitLaser stops the source traversal when a target callback consumes
// the beam. Immune callbacks which merely return still allow later targets.
func (w *World) weaponHitLaser(area CollisionRect, damage uint16) bool {
	return w.weaponHitRectOutcome(area, damage, true, true).StopProjectile
}

func (w *World) laserCallbackConsumes(actor *WorldActor, area CollisionRect) bool {
	if actor.firstSegment > 0 {
		return true
	}
	if actor.thirdFinalMember != nil && actor.thirdPart != nil && actor.thirdPart.DamageBehavior == "block-shot" {
		return true
	}
	if actor.fourthIndex > 0 {
		index := actor.fourthIndex - 1
		if actor.fourthFinal {
			return w.fourthFinalArt != nil && index < len(w.fourthFinalArt.Components) && w.fourthFinalArt.Components[index].DamageBehavior == "arm-contact"
		}
		if index >= 16 && index < 20 && w.FourthMiddle != nil {
			r := w.FourthMiddle.Parts[index].Collision
			top := CollisionRect{Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Top}
			bottom := CollisionRect{Left: r.Left, Top: r.Bottom, Right: r.Right, Bottom: r.Bottom}
			return area.Intersects(top) || area.Intersects(bottom)
		}
	}
	return actor.fifthFinal && actor.fifthPart != nil && actor.fifthPart.DamageBehavior == "barrier-contact"
}
