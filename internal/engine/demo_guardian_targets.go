package engine

// presentationGuardianTargetBounds follows the source damage callback's role
// and current controller gates. Tiled eyes and cores do not require a sprite
// renderer, and retained wrecks are excluded before their next draw callback.
func presentationGuardianTargetBounds(w *World, actor *WorldActor) (CollisionRect, bool) {
	if w == nil || actor == nil || !actor.Active || actor.ActorList != "moving" || actor.Collision.Empty() || actor.Collision.Left >= 320 || actor.Collision.Right < 0 {
		return CollisionRect{}, false
	}
	bounds := actor.Collision
	if index := actor.fourthIndex - 1; index >= 0 {
		if actor.fourthFinal {
			if w.FourthFinal == nil || w.FourthFinal.Defeated || index >= len(w.FourthFinal.Parts) || w.FourthFinal.Parts[index].Disabled {
				return CollisionRect{}, false
			}
			return bounds, index == 1 || index == 2 || index == 0 && w.FourthFinal.EyesRemaining == 0
		}
		if w.FourthMiddle == nil || w.FourthMiddle.Defeated || index >= len(w.FourthMiddle.Parts) || w.FourthMiddle.Parts[index].Disabled {
			return CollisionRect{}, false
		}
		if index >= 16 && index <= 19 {
			// The satellite's two outer lines invoke its armored callback.
			bounds.Top++
			bounds.Bottom--
			return bounds, !bounds.Empty()
		}
		return bounds, index == 15 || (index == 4 || index == 5) && w.FourthMiddle.OuterTargets == 0
	}
	if index := actor.fifthIndex - 1; index >= 0 {
		if actor.fifthFinal {
			if w.FifthFinal == nil || w.FifthFinal.Defeated || index >= len(w.FifthFinal.Parts) {
				return CollisionRect{}, false
			}
			part := w.FifthFinal.Parts[index]
			if !part.Active || part.Destroyed {
				return CollisionRect{}, false
			}
			return bounds, index >= 3 && index <= 20 || index == 21 && w.FifthFinal.OuterRemaining == 0
		}
		if w.FifthMiddle == nil || w.FifthMiddle.Defeated || index >= len(w.FifthMiddle.Parts) {
			return CollisionRect{}, false
		}
		part := w.FifthMiddle.Parts[index]
		return bounds, part.Active && !part.Destroyed && index >= 1 && index <= 5
	}
	return CollisionRect{}, false
}
