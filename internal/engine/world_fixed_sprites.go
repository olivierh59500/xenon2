package engine

func (w *World) advanceFixedSprite(actor *WorldActor) {
	state, kind := &actor.fixedState, actor.fixedKind
	events := StepFixedSpriteMotion(state, *kind, FixedSpriteInputs{
		ScrollDelta: w.ScrollDelta, ScrollY: w.ScrollY, MaximumScrollY: w.MaximumScrollY,
		PlayerX: w.Player.X, PlayerY: w.Player.Y,
	}, &w.random)
	actor.Active = !state.Removed
	actor.X, actor.Y = float64(state.X), float64(state.Y)
	actor.animation = FixedSpriteAnimation(*state, *kind)
	actor.animationState = state.Animation
	actor.selectSprite()
	if kind.CollisionMode == "sprite-prefix" {
		if box, ok := w.movingSpriteBoxes[actor.Sprite]; ok {
			actor.Collision = ActorCollisionRect(box, state.X, state.Y)
		}
	}
	if events.ContactDamage != 0 && w.PlayerAlive && w.Dive.Phase == 0 {
		r := events.ContactRectangle
		if (CollisionRect{Left: r[0], Top: r[1], Right: r[2], Bottom: r[3]}).Intersects(w.playerCollision) {
			w.damagePlayer(events.ContactDamage)
		}
	}
	if events.MoveToTransientList {
		actor.ActorList = "transient"
		w.nextActorID++
		actor.Order = w.nextActorID
		actor.Collision = CollisionRect{Right: -1, Bottom: -1}
	}
	if events.ShotCount != 0 {
		switch events.ShotMode {
		case "point", "point-burst":
			for _, direction := range events.ShotDirections[:events.ShotCount] {
				w.spawnEnemyShot(events.ShotX, events.ShotY, EnemyShot{Direction: uint8(direction), Speed: events.ShotSpeed})
				if len(w.Projectiles) != 0 {
					w.Projectiles[0].Atlas = "fixed"
					w.Projectiles[0].Sprite = events.ShotSprite
				}
			}
		default:
			if !w.spawnSpecializedFixedShot(events) {
				w.PendingFixedShots = append(w.PendingFixedShots, events)
			}
		}
	}
}
