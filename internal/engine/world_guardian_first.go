package engine

import "xenon2/internal/visualassets"

func (w *World) advanceFirstGuardian() {
	state, actor, art := w.FirstGuardian, w.firstGuardianActor, w.firstGuardianArt
	maximum, activate, warning := state.Advance(w.Frame, w.ScrollY, w.MaximumScrollY)
	w.MaximumScrollY, w.VisitedScrollY = maximum, maximum
	actor.Active = !state.Defeated
	actor.Visible = actor.Active && w.ScrollY < 640
	actor.Collision = state.BodyCollision
	actor.X, actor.Y = float64(art.BodyX), float64(state.BodyWorldY-w.ScrollY)
	actor.Patch = &art.Body
	actor.Flash = state.Flash
	actor.Extras = actor.Extras[:0]
	if actor.Visible && len(art.EyeFrames) != 0 {
		actor.Extras = append(actor.Extras, WorldSpriteAttachment{Atlas: "guardians", Sprite: art.EyeFrames[int(state.EyeClock>>6)%len(art.EyeFrames)], X: float64(art.EyeX), Y: actor.Y + float64(art.EyeOffsetY)})
	}
	if warning {
		w.SoundRequests[1] = "synthesized-effect-15"
	}
	if activate {
		segments := NewFirstGuardianSegments()
		w.FirstGuardianSegments = &segments
		group := make([]*WorldActor, 0, 8)
		for i := range 8 {
			w.nextActorID++
			part := &visualassets.ActorPart{ResourceTag: 252, DamageMode: "block-shot", MotionMode: "first-guardian-segment"}
			if i == 7 {
				part.ResourceTag = 256
			}
			segment := &WorldActor{ID: w.nextActorID, X: 0, Y: -100, PreviousY: -100, Active: true, Visible: true,
				ActorList: "moving", Atlas: "guardians", Sprite: art.SegmentSprite, part: part, firstSegment: i + 1,
				Collision: CollisionRect{Right: -1, Bottom: -1}}
			w.firstGuardianParts[i] = segment
			group = append(group, segment)
		}
		w.Actors = append(group, w.Actors...)
	}
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
		w.LevelFinished = true
		w.spawnExitCash(9)
	}
}
