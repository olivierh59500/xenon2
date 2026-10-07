package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

func (w *World) initializeFourthStage() error {
	if w.Level.Number != 4 || len(w.Level.GuardianGroups) == 0 {
		return nil
	}
	for i := range w.Level.GuardianGroups {
		group := &w.Level.GuardianGroups[i]
		if group.ID == "middle-guardian" {
			w.fourthMiddleArt = group
		}
		if group.ID == "final-guardian" {
			w.fourthFinalArt = group
		}
	}
	if w.fourthMiddleArt == nil || w.fourthFinalArt == nil || w.Level.GuardianParts == nil {
		return fmt.Errorf("fourth stage guardian resources are incomplete")
	}
	for _, sprite := range w.Level.GuardianParts.Sprites {
		if sprite.Collision != nil {
			w.movingSpriteBoxes[sprite.Name] = *sprite.Collision
		}
	}
	return nil
}

func (w *World) activateFourthGuardian(final bool) error {
	if final && w.FourthFinal != nil || !final && w.FourthMiddle != nil {
		return nil
	}
	art := w.fourthMiddleArt
	if final {
		art = w.fourthFinalArt
	} else {
		// The constructor changes the restart location while keeping the wallet
		// and loadout recorded by the preceding checkpoint record.
		w.Checkpoint.ScrollY, w.Checkpoint.PlayerX, w.Checkpoint.WorldY = 2480, 160, 176
		w.RestartCheckpoint()
		if w.poolError != nil {
			return w.poolError
		}
	}
	if art == nil {
		return fmt.Errorf("fourth guardian art is missing")
	}
	actors := make([]*WorldActor, len(art.Components))
	residues := make([]ActorResidue, len(actors))
	for index := range actors {
		descriptor := &art.Components[index]
		binding, err := w.reserveWorldActor(int16(descriptor.ResourceTag), ActorPoolMoving, true)
		if err != nil {
			return err
		}
		residues[index] = binding.Residue
		part := &visualassets.ActorPart{ResourceTag: descriptor.ResourceTag, StrongHealth: descriptor.StrongHealth, DamageMode: "fourth-guardian", MotionMode: descriptor.Behavior}
		actor := &WorldActor{ID: binding.EntityID, Binding: binding, Active: true, Visible: true, ActorList: "moving", Atlas: "guardian-parts", part: part, Health: descriptor.Health, fourthIndex: index + 1, fourthFinal: final, Collision: CollisionRect{Left: 1000, Right: 1000}}
		actor.X, actor.Y = float64(descriptor.InitialX), float64(descriptor.InitialWorldY-w.ScrollY)
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		actor.animation, actor.animationState = descriptor.Animation, NewAnimation(descriptor.Animation)
		actor.Sprite = actor.animationState.Sprite(actor.animation)
		if final && index < 3 {
			actor.Patch = &descriptor.TileFrames[0]
			if index == 0 {
				actor.X, actor.PreviousX = 112, 112
			}
		}
		if !final {
			w.updateSecondActorCollision(actor)
		}
		w.poolActors[binding.Slot] = actor
		actors[index] = actor
		w.Actors = append(w.Actors, actor)
	}
	if final {
		state, err := NewFourthFinalGuardian(art, w.ScrollY, residues)
		if err != nil {
			return err
		}
		w.FourthFinal = &state
		copy(w.fourthFinalActors[:], actors)
	} else {
		state, err := NewFourthMiddleGuardian(art, w.ScrollY, residues)
		if err != nil {
			return err
		}
		w.FourthMiddle = &state
		copy(w.fourthMiddleActors[:], actors)
	}
	for _, actor := range actors {
		w.storeFourthPartResidue(actor)
	}
	return nil
}

func (w *World) fourthInput() FourthGuardianInput {
	return FourthGuardianInput{Frame: w.Frame, ScrollY: w.ScrollY, ScrollDelta: w.ScrollDelta, MaximumScrollY: w.MaximumScrollY, PlayerX: w.Player.X, PlayerY: w.Player.Y}
}

func (w *World) advanceFourthPart(actor *WorldActor) error {
	index := actor.fourthIndex - 1
	art := w.fourthMiddleArt
	var part FourthGuardianPart
	var event FourthGuardianEvents
	var err error
	lookup := func(name string) visualassets.CollisionBox { return w.movingSpriteBoxes[name] }
	if actor.fourthFinal {
		art = w.fourthFinalArt
		event, err = w.FourthFinal.AdvancePart(index, art, w.fourthInput(), &w.Level.Paths.SineTable, lookup, w.random.Next)
		part = w.FourthFinal.Parts[index]
	} else {
		event, err = w.FourthMiddle.AdvancePart(index, art, w.fourthInput(), &w.Level.Paths.SineTable, lookup, w.random.Next)
		part = w.FourthMiddle.Parts[index]
	}
	if err != nil {
		return err
	}
	w.MaximumScrollY, w.VisitedScrollY = event.MaximumScrollY, event.MaximumScrollY
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	actor.X, actor.Y = float64(part.Arc.X>>16), float64(part.Arc.Y>>16)
	actor.Health, actor.Collision, actor.Visible, actor.Flash = int(part.Health), part.Collision, part.Visible, part.Flash
	if actor.fourthFinal {
		if index == 0 {
			actor.X = 112
			patch := &art.Components[index].TileFrames[part.Pose]
			actor.Patch = patch
		}
		if index == 1 || index == 2 {
			if part.Disabled {
				actor.Patch = art.Components[index].DestroyedTiles
			} else {
				actor.Patch = &art.Components[index].TileFrames[part.Pose]
			}
		}
		if index >= 3 {
			actor.Sprite = art.Components[index].Sprite
		}
	} else {
		actor.Sprite = w.FourthMiddle.Sprite(index, art)
	}
	if event.WarningSound {
		w.SoundRequests[2] = "synthesized-effect-15"
	}
	for _, shot := range event.Shots[:event.ShotCount] {
		w.spawnFourthShot(shot, art)
	}
	w.storeFourthPartResidue(actor)
	return nil
}

func (w *World) storeFourthPartResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	index := actor.fourthIndex - 1
	part := w.fourthPartState(index, actor.fourthFinal)
	r := &actor.Binding.Residue
	r.X, r.Y = int16(part.Arc.X>>16), int16(part.Arc.Y>>16)
	r.XFraction, r.YFraction = uint16(part.Arc.X), uint16(part.Arc.Y)
	r.Counter, r.Direction, r.HorizontalDriftRemainder = int16(part.Counter), int16(uint16(part.Arc.AngleFixed)), uint16(uint32(part.Arc.AngleFixed)>>16)
	r.MotionBudget, r.Health = int16(part.Arc.Budget), part.Health
	r.SetFireState(part.FireAccumulator, r.FireRate())
	w.storeWorldResidue(actor.Binding)
}

func (w *World) fourthPartState(index int, final bool) FourthGuardianPart {
	if final {
		return w.FourthFinal.Parts[index]
	}
	return w.FourthMiddle.Parts[index]
}

func (w *World) spawnFourthShot(event FourthGuardianShot, art *visualassets.GuardianGroup) {
	w.spawnEnemyShot(event.X, event.Y, EnemyShot{Direction: event.Direction, Speed: event.Speed})
	if len(w.Projectiles) == 0 {
		return
	}
	shot := w.Projectiles[0]
	for _, clip := range art.Animations {
		if clip.ID == event.Animation {
			shot.Atlas, shot.animation, shot.animationState = "guardian-parts", clip.Animation, NewAnimation(clip.Animation)
			shot.Sprite = shot.animationState.Sprite(clip.Animation)
			return
		}
	}
}

func (w *World) damageFourthGuardian(actor *WorldActor, area CollisionRect, amount uint16) {
	index := actor.fourthIndex - 1
	var event FourthGuardianEvents
	if actor.fourthFinal {
		event = w.FourthFinal.Strike(index, amount, w.ScrollY)
	} else {
		event = w.FourthMiddle.Strike(index, area, amount)
	}
	w.Score += event.Score
	parts := w.fourthMiddleActors[:]
	if actor.fourthFinal {
		parts = w.fourthFinalActors[:]
	}
	for i, member := range parts {
		if member == nil {
			continue
		}
		state := w.fourthPartState(i, actor.fourthFinal)
		member.Health, member.Flash = int(state.Health), state.Flash
		if actor.fourthFinal && (i == 1 || i == 2) && state.Disabled {
			member.Patch = w.fourthFinalArt.Components[i].DestroyedTiles
		}
	}
	if !event.Defeated {
		for i := len(parts) - 1; i >= 0; i-- {
			member := parts[i]
			if member != nil && event.RemoveParts[i] {
				w.spawnActorDeathEffect(member)
				member.Active = false
				w.storeActorResidue(member)
			}
		}
	}
	if event.Explosion {
		w.spawnSecondNamedExplosion(event.ExplosionX, event.ExplosionY, "explosion-large")
	}
	if !event.Defeated {
		return
	}
	if event.ClearMoving {
		for index := w.Pool.First(ActorPoolMoving); index != NoActorSlot; {
			next := w.Pool.Next(index)
			slot := w.Pool.Slot(index)
			if member := w.poolActors[index]; member != nil {
				member.Active = false
			}
			if event.ReleaseMoving {
				delete(w.poolBindings, slot.EntityID)
				w.clearPoolReferences(index)
				if err := w.Pool.Release(index); err != nil {
					w.poolError = err
				}
			} else {
				if err := w.Pool.MarkDead(index); err != nil {
					w.poolError = err
				}
			}
			index = next
		}
	}
	if event.ClearTerrainRows > 0 {
		for row := event.ClearTerrainRow; row < event.ClearTerrainRow+event.ClearTerrainRows; row++ {
			for column := 0; column < w.Level.Terrain.Columns; column++ {
				w.setSecondMapCell(column, row, 0)
			}
		}
	}
	if !actor.fourthFinal {
		w.MinimumScrollY = event.MinimumScrollY
		w.ScrollY, w.PreviousScrollY, w.RenderScrollY = event.ForcedScrollY, event.ForcedScrollY, event.ForcedScrollY
		w.MaximumScrollY, w.VisitedScrollY = event.MaximumScrollY, event.MaximumScrollY
		w.BaseScrollStep = 1
	}
	w.spawnSecondCashPairs(event.CashPairs, event.AdvanceLevel)
	if event.ExplosionCount > 0 {
		r := event.ExplosionRectangle
		w.spawnSecondRandomExplosions(event.ExplosionCount, r.Left, r.Top, r.Right-r.Left+1, r.Bottom-r.Top+1)
	}
}

func (w *World) restoreFourthGuardianActors(scrollChange int) {
	if w.FourthMiddle != nil && !w.FourthMiddle.Defeated {
		for i, actor := range w.fourthMiddleActors {
			if actor != nil && actor.Active {
				w.FourthMiddle.Parts[i].Arc.Y += int32(scrollChange) << 16
				actor.Y += float64(scrollChange)
				actor.PreviousY = actor.Y
				w.Actors = append(w.Actors, actor)
			}
		}
	}
	if w.FourthFinal != nil && !w.FourthFinal.Defeated {
		for i, actor := range w.fourthFinalActors {
			if actor != nil && actor.Active {
				w.FourthFinal.Parts[i].Arc.Y += int32(scrollChange) << 16
				actor.Y += float64(scrollChange)
				actor.PreviousY = actor.Y
				w.Actors = append(w.Actors, actor)
			}
		}
	}
}
