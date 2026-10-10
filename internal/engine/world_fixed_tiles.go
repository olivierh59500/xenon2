package engine

import "xenon2/internal/visualassets"

func (w *World) spawnFixedTile(record visualassets.FixedEncounter) bool {
	if w.Level.FixedTiles == nil {
		return false
	}
	for i := range w.Level.FixedTiles.Kinds {
		kind := &w.Level.FixedTiles.Kinds[i]
		if kind.Kind != record.EnemyKind {
			continue
		}
		if kind.Behavior != "first-tile-cannon" && kind.Behavior != "tile-fire-cycle" {
			return false
		}
		for j := range kind.Variants {
			variant := &kind.Variants[j]
			if variant.ID != record.Variant {
				continue
			}
			state := FixedTileState{Kind: kind.Kind, Variant: variant.ID, X: record.X + variant.OriginOffsetX, WorldY: record.Y + variant.OriginOffsetY}
			tag := variant.ResourceTag
			actor := &WorldActor{Active: true, ActorList: "moving", Health: kind.Health, Score: 100, fixedTileState: &state, fixedTileArt: kind, fixedTileVariant: variant,
				X: float64(state.X), Y: float64(state.WorldY - w.ScrollY), part: &visualassets.ActorPart{ResourceTag: tag, DamageMode: "fixed-tile"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
			if w.Level.Number == 4 {
				// This level awards a constant 400 without initializing the slot's score word.
				actor.Score = 400
			}
			actor.PreviousX, actor.PreviousY = actor.X, actor.Y
			if err := w.bindWorldActor(actor); err != nil {
				w.poolError = err
				return true
			}
			// These terrain constructors leave the reused strength byte intact.
			actor.part.StrongHealth = actor.Binding.Residue.StrongHealth
			residue := &actor.Binding.Residue
			residue.X, residue.Y = int16(state.X), int16(state.WorldY)
			residue.Counter, residue.VerticalFraction, residue.EmitterClock = 0, 0, 0
			residue.Health = uint16(kind.Health)
			if w.Level.Number != 4 {
				residue.PowerOrScore = 100
			}
			w.storeWorldResidue(actor.Binding)
			w.setSecondMapPatch(state.X/16, state.WorldY/16, variant.Initial)
			w.Actors = append([]*WorldActor{actor}, w.Actors...)
			return true
		}
		return true
	}
	return false
}

func (w *World) storeFixedTileResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	state, residue := actor.fixedTileState, &actor.Binding.Residue
	// The native terrain entry retains world Y. Screen anchors are only used
	// for drawing and collisions; unrelated reused words remain untouched.
	residue.X, residue.Y, residue.Counter = int16(state.X), int16(state.WorldY), int16(state.Phase)
	residue.Health = uint16(actor.Health)
	residue.SetFireState(state.Accumulator, residue.FireRate())
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}

func (w *World) advanceFixedTile(actor *WorldActor) {
	state, kind := actor.fixedTileState, actor.fixedTileArt
	var event FixedTileEvents
	if kind.Behavior == "first-tile-cannon" {
		event = StepFirstTileCannon(state, *kind, w.ScrollY, w.MaximumScrollY, &w.random)
	} else {
		event = StepTileFireCycle(state, *kind, w.ScrollY, w.MaximumScrollY, &w.random)
	}
	actor.Active, actor.Visible, actor.Collision = !state.Removed, false, event.Collision
	actor.X, actor.Y = float64(state.X), float64(state.WorldY-w.ScrollY)
	actor.Flash = false
	actor.Patch = nil
	if !actor.Active {
		return
	}
	if event.WriteTiles {
		patch := actor.fixedTileVariant.Frames[event.Frame]
		w.setSecondMapPatch(state.X/16, state.WorldY/16, patch)
	}
	if event.Shot {
		w.spawnEnemyShot(event.ShotX, event.ShotY, EnemyShot{Direction: event.ShotDirection, Speed: event.ShotSpeed})
		if w.Level.Rules != nil && event.ShotVariant >= 0 && event.ShotVariant < len(w.Level.Rules.EnemyShotChoices) && len(w.Projectiles) != 0 {
			w.Projectiles[0].Sprite = w.Level.Rules.EnemyShotChoices[event.ShotVariant]
		} else if w.Level.Rules != nil && event.ShotVariant < 0 && len(w.Projectiles) != 0 {
			shot := w.Projectiles[0]
			shot.Sprite = w.Level.Rules.TileShotSprites[state.Variant]
			clip := w.Level.Rules.TileShotAnimations[state.Variant]
			if len(clip.Frames) != 0 {
				shot.animation, shot.animationState = clip, NewAnimation(clip)
			}
		}
	}
}

func (w *World) damageFixedTile(actor *WorldActor, amount uint16) {
	index := w.waveDamageSlot(actor)
	result := ApplyEnemyDamage(uint16(actor.Health), amount)
	actor.Health = int(result.Health)
	actor.Patch = &actor.fixedTileVariant.Initial
	actor.Visible = true
	if !result.Destroyed {
		w.storeFixedTileResidue(actor)
		return
	}
	w.storeFixedTileResidue(actor)
	w.Score += actor.Score
	state := actor.fixedTileState
	patch := actor.fixedTileVariant.Destroyed
	w.spawnSecondNamedExplosion(state.X+patch.Columns*8, state.WorldY-w.ScrollY+patch.Rows*8, "explosion-large")
	w.SoundRequests[2] = "sampled-effect-03"
	// Keep the intended destroyed tiles if the explosion reuses this slot.
	// The original then reads its overwritten animation cursor as tile data.
	w.setSecondMapPatch(state.X/16, state.WorldY/16, actor.fixedTileVariant.Destroyed)
	actor.Active = false
	// The original final dead write follows its physical entry even when the
	// explosion reclaimed that entry during allocation.
	w.retireWaveDamageSlot(index)
}
