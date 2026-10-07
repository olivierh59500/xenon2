package engine

func (w *World) advanceSecondGuardian() {
	actor, art := w.secondGuardianActor, w.secondGuardianArt
	event := w.SecondGuardian.Advance(SecondGuardianInput{Frame: w.Frame, ScrollY: w.ScrollY, MaximumScrollY: w.MaximumScrollY,
		PlayerX: w.Player.X, PlayerY: w.Player.Y, ActorCount: w.MovingEnemyCount,
		FireRate: uint8(art.MotionParameters["fire_rate"]), ShotSpeed: art.MotionParameters["shot_speed"]}, &w.random)
	w.MaximumScrollY, w.VisitedScrollY = event.MaximumScrollY, event.MaximumScrollY
	actor.X, actor.Y = float64(art.BodyX), float64(w.SecondGuardian.BodyWorldY-w.ScrollY)
	actor.Visible = w.ScrollY < 352
	actor.Collision = w.SecondGuardian.BodyCollision
	actor.Patch = &w.secondGuardianBody
	actor.Flash = false
	for _, animation := range art.BodyAnimations {
		frame, changed := 0, false
		switch animation.ID {
		case "center-hatch":
			frame, changed = event.CenterHatchFrame, event.CenterHatchChanged
		case "left-thruster", "right-thruster":
			frame, changed = event.ThrusterFrame, event.ThrustersChanged
		}
		if !changed || frame >= len(animation.Frames) {
			continue
		}
		patch := animation.Frames[frame]
		for row := range patch.Rows {
			copy(w.secondGuardianBody.Tiles[(animation.Row+row)*w.secondGuardianBody.Columns+animation.Column:][:patch.Columns], patch.Tiles[row*patch.Columns:][:patch.Columns])
		}
	}
	if event.MovementSound {
		w.SoundRequests[2] = "synthesized-effect-03"
	}
	for _, animation := range art.Animations {
		if animation.ID != event.ShotAnimation {
			continue
		}
		for _, direction := range event.ShotDirections[:event.ShotCount] {
			w.spawnEnemyShot(event.ShotX, event.ShotY, EnemyShot{Direction: direction, Speed: event.ShotSpeed})
			shot := w.Projectiles[0]
			shot.Atlas, shot.animation = "guardians", animation.Animation
			shot.animationState = NewAnimation(animation.Animation)
			shot.Sprite = shot.animationState.Sprite(animation.Animation)
		}
	}
	if event.SpawnMinion {
		w.PendingGuardianMinions = append(w.PendingGuardianMinions, event)
		w.spawnSecondMinion(event)
	}
}

func (w *World) damageSecondGuardian(actor *WorldActor, amount uint16) {
	damage := ApplyEnemyDamage(uint16(actor.Health), amount)
	actor.Health, actor.Flash = int(damage.Health), true
	if !damage.Destroyed {
		return
	}
	actor.Active = false
	w.spawnSecondCashPairs(10, true)
	w.spawnSecondRandomExplosions(30, 80, w.SecondGuardian.BodyWorldY-w.ScrollY, 160, 112)
	for _, candidate := range w.Actors {
		if candidate.ActorList == "moving" {
			candidate.Active = false
		}
	}
}
