package engine

import "xenon2/internal/visualassets"

func (w *World) spawnSecondMinion(event SecondGuardianEvents) {
	if w.secondMinionConfig == nil {
		return
	}
	state := NewSecondGuardianMinion(event.MinionX, event.MinionY, event.MinionHeading, *w.secondMinionConfig)
	w.addSecondMinionActor(state)
}
func (w *World) addSecondMinionActor(state SecondMinionState) {
	config := w.secondMinionConfig
	clip := state.Clip(*config)
	tag, health, score, damage := 268, 0, 0, "block-shot"
	if state.Turret {
		tag, health, score, damage = 272, w.secondGuardianArt.MotionParameters["turret_health"], 30, "individual"
	}
	actor := &WorldActor{X: float64(state.X), Y: float64(state.Y), PreviousX: float64(state.X), PreviousY: float64(state.Y), Active: true, Visible: true, Atlas: "guardians", ActorList: "moving", Health: health, Score: score, animation: clip, animationState: state.Animation, secondMinion: &state,
		part: &visualassets.ActorPart{ResourceTag: tag, DamageMode: damage}, Collision: CollisionRect{Right: -1, Bottom: -1}}
	actor.Sprite = state.Animation.Sprite(clip)
	if state.Turret {
		w.updateSecondActorCollision(actor)
	}
	if err := w.bindWorldActor(actor); err != nil {
		w.poolError = err
		return
	}
	w.Actors = append([]*WorldActor{actor}, w.Actors...)
}
func (w *World) updateSecondActorCollision(actor *WorldActor) {
	if box, ok := w.movingSpriteBoxes[actor.Sprite]; ok {
		actor.Collision = ActorCollisionRect(box, int(actor.X), int(actor.Y))
	}
}
func (w *World) advanceSecondMinion(actor *WorldActor) {
	event := actor.secondMinion.Advance(SecondMinionInput{Frame: w.Frame, ScrollY: w.ScrollY, ScrollDelta: w.ScrollDelta, PlayerX: w.Player.X, PlayerY: w.Player.Y}, *w.secondMinionConfig, &w.random)
	// Every source branch finishes with a bounded direction, clamped X or the
	// terrain-contact rectangle's left edge, all in the guardian's arena.
	w.secondCrowdedReverse = false
	actor.X, actor.Y = float64(actor.secondMinion.X), float64(actor.secondMinion.Y)
	actor.Active = !actor.secondMinion.Removed
	actor.animation = actor.secondMinion.Clip(*w.secondMinionConfig)
	actor.Sprite = actor.secondMinion.Animation.Sprite(actor.animation)
	actor.Flash = false
	if actor.secondMinion.Turret {
		w.updateSecondActorCollision(actor)
	}
	if event.Transform {
		w.addSecondMinionActor(event.TurretState)
	}
	if event.Shot {
		w.spawnEnemyShot(event.ShotX, event.ShotY, EnemyShot{Direction: event.ShotDirection, Speed: event.ShotSpeed})
		if len(w.Projectiles) != 0 {
			for _, animation := range w.secondGuardianArt.Animations {
				if animation.ID == "guardian-turret-shot" {
					shot := w.Projectiles[0]
					shot.Atlas, shot.animation = "guardians", animation.Animation
					shot.animationState = NewAnimation(animation.Animation)
					shot.Sprite = shot.animationState.Sprite(animation.Animation)
					break
				}
			}
		}
	}
	if event.RestoreTerrain && w.secondTerrainCells != nil {
		if index := w.secondTerrainCells.RestoreOverlap(actor.Collision, w.ScrollY); index >= 0 {
			cell := w.secondTerrainCells.Cells[index]
			w.setSecondMapCell(cell.X/16, cell.WorldY/16, cell.RestoredTile)
		}
	}
}
func (w *World) strikeSecondTerrain(rect CollisionRect) bool {
	if w.secondTerrainCells == nil {
		return false
	}
	index := w.secondTerrainCells.FindBullet(rect, w.ScrollY)
	if index < 0 {
		return false
	}
	w.secondTerrainCells.Intact[index] = false
	cell := w.secondTerrainCells.Cells[index]
	w.setSecondMapCell(cell.X/16, cell.WorldY/16, 0)
	w.spawnSecondExplosion(cell.X+8, cell.WorldY-w.ScrollY+8)
	return true
}
func (w *World) spawnSecondExplosion(x, y int) {
	w.spawnSecondNamedExplosion(x, y, "explosion-small")
	w.SoundRequests[2] = "sampled-effect-05"
}

func (w *World) spawnSecondNamedExplosion(x, y int, name string) {
	w.SoundRequests[2] = "sampled-effect-05"
	if name == "explosion-large" {
		w.SoundRequests[2] = "sampled-effect-03"
	}
	animation, ok := w.commonAnimations[name]
	if !ok {
		return
	}
	actor := &WorldActor{X: float64(x), Y: float64(y), PreviousX: float64(x), PreviousY: float64(y), Active: true, Visible: true, Atlas: "common", ActorList: "transient", animation: animation.Animation, animationState: NewAnimation(animation.Animation), part: &visualassets.ActorPart{ResourceTag: 12, DamageMode: "block-shot"}}
	actor.Sprite = actor.animationState.Sprite(actor.animation)
	if err := w.bindWorldActor(actor); err != nil {
		w.poolError = err
		return
	}
	w.Actors = append([]*WorldActor{actor}, w.Actors...)
}
func (w *World) spawnSecondRandomExplosions(count, left, top, width, height int) {
	for range count {
		x, _ := w.random.Below(uint16(width))
		y, _ := w.random.Below(uint16(height))
		w.spawnSecondNamedExplosion(left+int(x), top+int(y), "explosion-large")
	}
	w.SoundRequests[1], w.SoundRequests[2] = "sampled-effect-03", "sampled-effect-03"
	w.ImmediateSoundRequests[0] = "sampled-effect-03"
}
func (w *World) spawnSecondCashPairs(pairs int, exit bool) {
	if exit {
		w.LevelFinished = true
	}
	w.spawnExitCash(pairs)
}
