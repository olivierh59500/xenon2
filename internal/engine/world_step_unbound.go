package engine

// advanceUnboundProjectiles preserves the diagnostic lists' original snapshots
// and merge order across the same blocking callback as the physical lists.
func (w *World) advanceUnboundProjectiles(input Input) error {
	s := w.stepContinuation.unbound
	if s == nil {
		s = &worldUnboundContinuation{actors: w.Actors, projectiles: w.Projectiles, smallShots: w.SmallShots}
		w.stepContinuation.unbound = s
	}
	if !s.actorsDone {
		for s.actorAt < len(s.actors) {
			actor := s.actors[s.actorAt]
			s.actorAt++
			if !actor.Active || actor.ActorList != "transient" || actor.Binding.EntityID != 0 {
				continue
			}
			actor.PreviousX, actor.PreviousY = actor.X, actor.Y
			if actor.secondFragment != nil {
				w.advanceSecondFragment(actor)
				w.finishActorUpdate(actor)
				continue
			}
			if actor.animation.Ending == "remove" && actor.animationState.Frame == len(actor.animation.Frames)-1 && actor.animationState.Remaining == 1 {
				actor.animationState.Remaining = 0
				actor.Active = false
				w.finishActorUpdate(actor)
				continue
			}
			actor.animationState.Advance(actor.animation)
			actor.selectSprite()
			w.finishActorUpdate(actor)
		}
		s.actorsDone = true
		s.collectibles = w.Collectibles
		w.weaponIDs = w.weaponIDs[:0]
	}
	for s.enemyAt < len(s.projectiles) || s.playerAt < len(s.smallShots) || s.collectibleAt < len(s.collectibles) {
		enemyID, playerID, collectibleID := -2147483648, -2147483648, -2147483648
		for s.enemyAt < len(s.projectiles) && (s.projectiles[s.enemyAt].Binding.EntityID != 0 || !s.projectiles[s.enemyAt].Active) {
			s.enemyAt++
		}
		for s.playerAt < len(s.smallShots) && (s.smallShots[s.playerAt].Binding.EntityID != 0 || !s.smallShots[s.playerAt].Active) {
			s.playerAt++
		}
		for s.collectibleAt < len(s.collectibles) && (s.collectibles[s.collectibleAt].Binding.EntityID != 0 || !s.collectibles[s.collectibleAt].Active) {
			s.collectibleAt++
		}
		if s.enemyAt >= len(s.projectiles) && s.playerAt >= len(s.smallShots) && s.collectibleAt >= len(s.collectibles) {
			break
		}
		if s.enemyAt < len(s.projectiles) {
			enemyID = s.projectiles[s.enemyAt].ID
		}
		if s.playerAt < len(s.smallShots) {
			playerID = s.smallShots[s.playerAt].ID
		}
		if s.collectibleAt < len(s.collectibles) {
			collectibleID = s.collectibles[s.collectibleAt].ID
			if s.collectibles[s.collectibleAt].Order != 0 {
				collectibleID = s.collectibles[s.collectibleAt].Order
			}
		}
		if collectibleID > max(enemyID, playerID) {
			item := s.collectibles[s.collectibleAt]
			w.advanceCollectible(item)
			w.finishCollectibleUpdate(item)
			s.collectibleAt++
		} else if enemyID > playerID {
			shot := s.projectiles[s.enemyAt]
			if err := w.advanceEnemyShot(shot); err != nil {
				return err
			}
			w.finishProjectileUpdate(shot)
			s.enemyAt++
		} else {
			shot := s.smallShots[s.playerAt]
			w.advanceSmallShot(shot)
			w.finishSmallShotUpdate(shot)
			s.playerAt++
		}
		if w.ScreenClearFrames != 0 {
			return nil
		}
	}
	w.stepContinuation.unbound = nil
	return nil
}
