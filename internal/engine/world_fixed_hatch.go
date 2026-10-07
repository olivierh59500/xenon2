package engine

import "xenon2/internal/visualassets"

func (w *World) spawnFixedHatch(record visualassets.FixedEncounter) bool {
	if w.Level.Number != 2 || record.EnemyKind != 2 || w.Level.FixedTiles == nil {
		return false
	}
	for i := range w.Level.FixedTiles.Kinds {
		kind := &w.Level.FixedTiles.Kinds[i]
		if kind.Behavior != "second-hatch" {
			continue
		}
		for j := range kind.Variants {
			variant := &kind.Variants[j]
			if variant.ID != record.Variant {
				continue
			}
			state := FixedHatchState{X: record.X + variant.OriginOffsetX, WorldY: record.Y + variant.OriginOffsetY}
			actor := &WorldActor{Active: true, ActorList: "scenery", fixedHatch: &state, fixedTileArt: kind, fixedTileVariant: variant,
				part: &visualassets.ActorPart{ResourceTag: variant.ResourceTag, DamageMode: "block-shot"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
			if err := w.bindWorldActor(actor); err != nil {
				w.poolError = err
				return true
			}
			w.initializeSecondEmitterResidue(actor)
			w.setSecondMapPatch(state.X/16, state.WorldY/16, variant.Initial)
			w.Actors = append([]*WorldActor{actor}, w.Actors...)
			return true
		}
	}
	return false
}

func (w *World) advanceFixedHatch(actor *WorldActor) {
	actor.Visible = false
	event := actor.fixedHatch.Advance(w.ScrollY, actor.fixedTileArt.TriggerScreenY)
	if event.WriteTiles {
		state := actor.fixedHatch
		w.setSecondMapPatch(state.X/16, state.WorldY/16, actor.fixedTileVariant.Frames[event.Frame])
	}
	if event.Spawn {
		w.spawnHatchCreatures(event.SpawnX, event.SpawnY)
	}
	actor.Active = !actor.fixedHatch.Removed
}

func (w *World) spawnHatchCreatures(x, y int) {
	if w.Level.FixedSprites == nil || w.Level.FixedSprites.HatchCreatures == nil {
		return
	}
	art := w.Level.FixedSprites.HatchCreatures
	for i := range 8 {
		clip := art.InitialAnimations[i]
		state := HatchCreatureState{X: x + art.Offsets[i][0], Y: y + art.Offsets[i][1], Direction: uint8(7 - i), Clip: clip, Animation: NewAnimation(clip)}
		actor := &WorldActor{Active: true, Visible: true, ActorList: "moving", Atlas: "fixed", Health: 1, Score: 20, hatchCreature: &state,
			X: float64(state.X), Y: float64(state.Y), PreviousX: float64(state.X), PreviousY: float64(state.Y), Sprite: state.Animation.Sprite(clip), part: &visualassets.ActorPart{ResourceTag: 264, DamageMode: "individual"}}
		if err := w.bindWorldActor(actor); err != nil {
			w.poolError = err
			return
		}
		state.Timer = int(w.random.Next() & 31)
		w.initializeSecondCreatureResidue(actor)
		w.updateSecondActorCollision(actor)
		w.Actors = append([]*WorldActor{actor}, w.Actors...)
	}
}

func (w *World) advanceHatchCreature(actor *WorldActor) {
	actor.Flash = false
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	state := actor.hatchCreature
	event := state.Advance(w.Level.FixedSprites.HatchCreatures, w.fixedProjectileInputs(), func(name string) visualassets.CollisionBox { return w.movingSpriteBoxes[name] })
	actor.Active, actor.Visible = !state.Removed, !state.Removed
	actor.X, actor.Y = float64(state.X), float64(state.Y)
	actor.Sprite = state.Animation.Sprite(state.Clip)
	w.updateSecondActorCollision(actor)
	if event.Explosion {
		region := w.fixedProjectileRegion(actor.Sprite)
		w.spawnSecondExplosion(state.X-region.AnchorX+region.Width/2, state.Y-region.AnchorY+(region.Height-1)/2)
	}
	if event.PlayerDamage != 0 {
		w.damagePlayer(event.PlayerDamage)
	}
}
