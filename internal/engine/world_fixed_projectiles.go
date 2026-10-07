package engine

import "xenon2/internal/visualassets"

func (w *World) fixedProjectileInputs() FixedProjectileInputs {
	return FixedProjectileInputs{ScrollDelta: w.ScrollDelta, PlayerX: w.Player.X, PlayerY: w.Player.Y,
		PlayerBounds: w.playerCollision, CanHitPlayer: w.PlayerAlive && w.Dive.Phase == 0,
		Invulnerable: w.InvulnerableFrames != 0}
}

func (w *World) fixedProjectileRegion(name string) visualassets.SpriteRegion {
	if w.Level.FixedSprites != nil {
		for _, sprite := range w.Level.FixedSprites.Atlas.Sprites {
			if sprite.Name == name {
				return sprite
			}
		}
	}
	return visualassets.SpriteRegion{}
}

func (w *World) spawnSpecializedFixedShot(event FixedSpriteEvents) bool {
	if w.Level.FixedSprites == nil || w.Level.FixedSprites.Projectile == nil {
		return false
	}
	art := w.Level.FixedSprites.Projectile
	switch event.ShotMode {
	case "turning-projectile":
		binding, err := w.reserveWorldActor(int16(art.ResourceTag), ActorPoolProjectile, false)
		if err != nil {
			w.poolError = err
			return true
		}
		state, err := NewTurningFixedProjectile(event, art, binding.Residue.XFraction, binding.Residue.YFraction)
		if err != nil {
			return false
		}
		shot := &WorldProjectile{ID: binding.EntityID, Binding: binding, X: float64(event.ShotX), Y: float64(event.ShotY),
			PreviousX: float64(event.ShotX), PreviousY: float64(event.ShotY), Sprite: event.ShotSprite,
			Atlas: "fixed", Active: true, turning: &state}
		w.poolProjectiles[binding.Slot] = shot
		w.Projectiles = append([]*WorldProjectile{shot}, w.Projectiles...)
	case "animated-aiming-projectile":
		state, err := NewAnimatedAimingFixedProjectile(event, art)
		if err != nil {
			return false
		}
		part := &visualassets.ActorPart{ResourceTag: art.ResourceTag, Score: art.Score, MotionMode: event.ShotMode, DamageMode: "individual"}
		actor := &WorldActor{X: float64(state.X), Y: float64(state.Y),
			PreviousX: float64(state.X), PreviousY: float64(state.Y), Atlas: "fixed", ActorList: "moving",
			Active: true, Visible: true, Health: art.Health, Score: art.Score, part: part, fixedAiming: &state,
			Sprite: state.Sprite(art)}
		actor.Collision = ActorCollisionRect(w.movingSpriteBoxes[actor.Sprite], state.X, state.Y)
		if err := w.bindWorldActor(actor); err != nil {
			w.poolError = err
			return true
		}
		w.Actors = append([]*WorldActor{actor}, w.Actors...)
	default:
		return false
	}
	return true
}

func (w *World) advanceTurningFixedShot(shot *WorldProjectile) error {
	shot.PreviousX, shot.PreviousY = shot.X, shot.Y
	state := shot.turning
	event, err := state.Advance(w.Level.FixedSprites.Projectile, w.fixedProjectileInputs(), w.movingSpriteBoxes[shot.Sprite])
	if err != nil {
		return err
	}
	shot.X, shot.Y = float64(state.Motion.X>>16), float64(state.Motion.Y>>16)
	shot.Sprite, shot.Active = state.Sprite, !state.Removed
	if event.PlayerDamage != 0 {
		w.damagePlayer(event.PlayerDamage)
	}
	return nil
}

func (w *World) advanceFixedAimingActor(actor *WorldActor) {
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	actor.Flash = false
	state, art := actor.fixedAiming, w.Level.FixedSprites.Projectile
	event := state.Advance(art, w.fixedProjectileInputs(), func(name string) visualassets.CollisionBox {
		return w.movingSpriteBoxes[name]
	}, w.fixedProjectileRegion)
	actor.X, actor.Y = float64(state.X), float64(state.Y)
	actor.Sprite, actor.Active, actor.Collision = state.Sprite(art), !state.Removed, event.Collision
	if event.Explosion {
		w.spawnSecondExplosion(event.ExplosionX, event.ExplosionY)
	}
	if event.PlayerDamage != 0 {
		w.damagePlayer(event.PlayerDamage)
	}
}
