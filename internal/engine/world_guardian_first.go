package engine

import "xenon2/internal/visualassets"

func (w *World) advanceFirstGuardian() {
	state, actor, art := w.FirstGuardian, w.firstGuardianActor, w.firstGuardianArt
	wasVisible := actor.Visible
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	maximum, activate, warning := state.Advance(w.Frame, w.ScrollY, w.MaximumScrollY)
	w.MaximumScrollY, w.VisitedScrollY = maximum, maximum
	actor.Active = !state.Defeated
	actor.Visible = actor.Active && w.ScrollY < 640
	actor.Collision = state.BodyCollision
	actor.X, actor.Y = float64(art.BodyX), float64(state.BodyWorldY-w.ScrollY)
	if !wasVisible {
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	}
	if w.firstGuardianBody.Columns == 0 {
		w.firstGuardianBody = art.Body
		w.firstGuardianBody.Tiles = append([]uint16(nil), art.Body.Tiles...)
	}
	if state.Active && !state.Defeated {
		for _, animation := range art.BodyAnimations {
			patch := animation.Frames[(w.Frame&12)>>2]
			for row := range patch.Rows {
				copy(w.firstGuardianBody.Tiles[(animation.Row+row)*w.firstGuardianBody.Columns+animation.Column:][:patch.Columns], patch.Tiles[row*patch.Columns:][:patch.Columns])
			}
		}
	}
	actor.Patch = &w.firstGuardianBody
	actor.Flash = state.Flash
	actor.Extras = actor.Extras[:0]
	if actor.Visible && len(art.EyeFrames) != 0 {
		actor.Extras = append(actor.Extras, WorldSpriteAttachment{Atlas: "guardians", Sprite: art.EyeFrames[int(state.EyeClock>>6)%len(art.EyeFrames)], X: float64(art.EyeX), Y: actor.Y + float64(art.EyeOffsetY), PreviousX: float64(art.EyeX), PreviousY: actor.PreviousY + float64(art.EyeOffsetY), Interpolate: true})
	}
	if warning {
		w.SoundRequests[1] = "synthesized-effect-15"
	}
	if activate {
		segments := NewFirstGuardianSegments()
		w.FirstGuardianSegments = &segments
		group := make([]*WorldActor, 0, 8)
		for i := range 8 {
			part := &visualassets.ActorPart{ResourceTag: 252, DamageMode: "block-shot", MotionMode: "first-guardian-segment"}
			if i == 7 {
				part.ResourceTag = 256
			}
			segment := &WorldActor{X: 0, Y: -100, PreviousY: -100, Active: true, Visible: true,
				ActorList: "moving", Atlas: "guardians", Sprite: art.SegmentSprite, part: part, firstSegment: i + 1,
				Collision: CollisionRect{Right: -1, Bottom: -1}}
			if err := w.bindWorldActor(segment); err != nil {
				w.poolError = err
				return
			}
			// The constructor initializes only whole Y, firing
			// state and contact strength. Other gameplay words survive reuse.
			residue := &segment.Binding.Residue
			residue.Y, residue.EmitterClock, residue.StrongHealth = -100, 0, false
			segment.X, segment.PreviousX = float64(residue.X), float64(residue.X)
			w.Pool.Slot(segment.Binding.Slot).AuxiliaryFlags[0] = true
			w.storeWorldResidue(segment.Binding)
			if i > 0 {
				w.Pool.unlink(segment.Binding.Slot)
				if err := w.Pool.AttachAfter(segment.Binding.Slot, ActorPoolMoving, segment.ID, int16(part.ResourceTag), group[i-1].Binding.Slot); err != nil {
					w.poolError = err
					return
				}
			}
			w.firstGuardianParts[i] = segment
			group = append(group, segment)
		}
		w.Actors = append(group, w.Actors...)
	}
}

func (w *World) storeFirstGuardianSegmentResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
		return
	}
	index := actor.firstSegment - 1
	motion, residue := w.FirstGuardianSegments.Pieces[index], &actor.Binding.Residue
	residue.X, residue.Y = int16(actor.X), int16(actor.Y)
	residue.XFraction, residue.YFraction = uint16(motion.X), uint16(motion.Y)
	residue.Counter, residue.MotionBudget = int16(100-motion.Budget), int16(motion.Budget)
	residue.Direction, residue.HorizontalDriftRemainder = int16(uint16(motion.AngleFixed)), uint16(uint32(motion.AngleFixed)>>16)
	residue.MountOffsetX, residue.MountOffsetY = int16(motion.AngularVelocity), int16(motion.AngularAcceleration)
	if index == 7 {
		residue.SetFireState(w.FirstGuardianSegments.TailFire.Accumulator, residue.FireRate())
	}
	w.storeWorldResidue(actor.Binding)
}

func (w *World) advanceFirstGuardianSegments() error {
	segments := w.FirstGuardianSegments
	for _, actor := range w.firstGuardianParts {
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	}
	shot, fired, err := segments.Advance(*w.FirstGuardian, w.ScrollY, w.Player.X, w.Player.Y, &w.Level.Paths.SineTable, w.random.Next)
	if err != nil {
		return err
	}
	for i, motion := range segments.Pieces {
		actor := w.firstGuardianParts[i]
		actor.Active, actor.Visible = segments.Alive, segments.Alive
		actor.X, actor.Y = float64(motion.X>>16), float64(motion.Y>>16)
		if i == 7 && motion.Budget > 0 && len(w.firstGuardianArt.TailHeadingFrames) != 0 {
			actor.Sprite = w.firstGuardianArt.TailHeadingFrames[segments.TailFrame]
		}
		actor.Collision = CollisionRect{Right: -1, Bottom: -1}
		if motion.Budget != 0 {
			for _, sprite := range w.Level.Guardians.Atlas.Sprites {
				if sprite.Name == actor.Sprite && sprite.Collision != nil {
					actor.Collision = ActorCollisionRect(*sprite.Collision, int(actor.X), int(actor.Y))
					break
				}
			}
		}
	}
	if fired {
		tail := w.firstGuardianParts[7]
		w.spawnEnemyShot(int(tail.X), int(tail.Y), shot)
		if len(w.Projectiles) != 0 && len(w.firstGuardianArt.FlameAnimation.Frames) != 0 {
			w.Projectiles[0].Atlas = "guardians"
			w.Projectiles[0].Sprite = w.firstGuardianArt.FlameAnimation.Frames[0].Sprite
			w.Projectiles[0].animation = w.firstGuardianArt.FlameAnimation
			w.Projectiles[0].animationState = NewAnimation(w.firstGuardianArt.FlameAnimation)
		}
	}
	return nil
}

func (w *World) strikeFirstGuardian(hit CollisionRect, amount uint16) {
	damaged, destroyed := w.FirstGuardian.Strike(hit, amount, w.ScrollY)
	if !damaged {
		return
	}
	w.firstGuardianActor.Flash = true
	w.firstGuardianActor.Extras = w.firstGuardianActor.Extras[:0]
	if destroyed {
		w.firstGuardianActor.Active = false
		for _, segment := range w.firstGuardianParts {
			if segment != nil {
				segment.Active = false
			}
		}
		w.syncDeadActors()
		w.LevelFinished = true
		w.spawnExitCash(9)
		w.spawnSecondRandomExplosions(20, 0, 0, 320, 192)
	}
}
