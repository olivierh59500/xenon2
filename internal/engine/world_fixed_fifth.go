package engine

import "xenon2/internal/visualassets"

func (w *World) spawnFifthTile(record visualassets.FixedEncounter) bool {
	if w.Level.Number != 5 || (record.EnemyKind != 1 && record.EnemyKind != 3 && record.EnemyKind != 4) || w.Level.FixedTiles == nil {
		return false
	}
	var kind *visualassets.FixedTileKind
	for i := range w.Level.FixedTiles.Kinds {
		if w.Level.FixedTiles.Kinds[i].Kind == record.EnemyKind {
			kind = &w.Level.FixedTiles.Kinds[i]
			break
		}
	}
	if kind == nil {
		return false
	}
	if record.EnemyKind == 1 {
		var group []*WorldActor
		for i := range kind.Parts {
			part := &kind.Parts[i]
			state := FifthTileState{X: record.X - 8 + part.OffsetX, WorldY: record.Y - 8, Part: i}
			binding, err := w.reserveWorldActor(int16(part.ResourceTag), ActorPoolMoving, true)
			if err != nil {
				w.poolError = err
				return true
			}
			actor := &WorldActor{ID: binding.EntityID, Binding: binding, Active: true, ActorList: "moving", Health: kind.Health, fifthTile: &state, fixedTileArt: kind, part: &visualassets.ActorPart{ResourceTag: part.ResourceTag, DamageMode: "fifth-tile"}, Collision: CollisionRect{Left: 1000, Right: 1000}}
			actor.X, actor.Y = float64(state.X), float64(state.WorldY-w.ScrollY)
			actor.PreviousX, actor.PreviousY = actor.X, actor.Y
			w.poolActors[binding.Slot] = actor
			w.Actors = append(w.Actors, actor)
			group = append(group, actor)
			w.storeActorResidue(actor)
		}
		for _, actor := range group {
			actor.fifthTileGroup = group
		}
		w.setSecondMapPatch((record.X-8)/16, (record.Y-8)/16, kind.Variants[0].Initial)
		return true
	}
	variantID := record.Variant
	if kind.Kind == 4 {
		variantID = 0
	}
	if variantID < 0 || variantID >= len(kind.Variants) {
		return true
	}
	variant := &kind.Variants[variantID]
	state := FifthTileState{X: record.X - 8, WorldY: record.Y - 8, Heading: uint8(variantID)}
	actor := &WorldActor{Active: true, ActorList: "moving", Health: kind.Health, fifthTile: &state, fixedTileArt: kind, fixedTileVariant: variant, part: &visualassets.ActorPart{ResourceTag: variant.ResourceTag, DamageMode: "fifth-tile"}, Collision: CollisionRect{Left: 1000, Right: 1000}}
	actor.X, actor.Y = float64(state.X), float64(state.WorldY-w.ScrollY)
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	if err := w.bindWorldActor(actor); err != nil {
		w.poolError = err
		return true
	}
	actor.Binding.Residue.EmitterClock = 0
	w.storeActorResidue(actor)
	w.replaceFifthAdjacentTiles(state.X/16, state.WorldY/16, kind.InitialChanges)
	w.setSecondMapPatch(state.X/16, state.WorldY/16, variant.Initial)
	w.Actors = append([]*WorldActor{actor}, w.Actors...)
	return true
}

func (w *World) replaceFifthAdjacentTiles(column, row int, changes []visualassets.ConditionalTileReplacement) {
	for _, change := range changes {
		position := row*w.Level.Terrain.Columns + column + change.ColumnOffset
		if position >= 0 && position < len(w.Level.Terrain.Map) && w.Level.Terrain.Map[position] == change.Before {
			w.setSecondMapPatch(column+change.ColumnOffset, row, change.After)
		}
	}
}

func (w *World) advanceFifthTile(actor *WorldActor) {
	state, kind := actor.fifthTile, actor.fixedTileArt
	var event FixedTileEvents
	if kind.Kind == 1 {
		event = state.AdvanceBarrier(*kind, w.ScrollY)
	} else if kind.Kind == 4 {
		event = state.AdvanceRadialTurret(*kind, w.ScrollY, w.MaximumScrollY, &w.random)
	} else {
		event = state.AdvanceAimingTurret(*kind, w.ScrollY, w.MaximumScrollY, w.Player.X, w.Player.Y, &w.random)
	}
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	actor.X, actor.Y = float64(state.X), float64(state.WorldY-w.ScrollY)
	actor.Active, actor.Visible, actor.Flash, actor.Patch = !state.Removed, false, false, nil
	actor.Collision = event.Collision
	if event.WriteTiles {
		var patch visualassets.TilePatch
		if kind.Kind == 1 {
			patch = kind.Parts[state.Part].Frames[event.Frame]
		} else {
			patch = actor.fixedTileVariant.Frames[event.Frame]
		}
		w.setSecondMapPatch(state.X/16, state.WorldY/16, patch)
	}
	if event.Shot {
		if kind.Kind == 4 {
			for direction := 7; direction >= 0; direction-- {
				w.spawnEnemyShot(event.ShotX, event.ShotY, EnemyShot{Direction: uint8(direction), Speed: event.ShotSpeed})
				if w.Level.Rules != nil && len(w.Projectiles) != 0 && len(w.Level.Rules.RadialTileShotAnimation.Frames) != 0 {
					clip := w.Level.Rules.RadialTileShotAnimation
					shot := w.Projectiles[0]
					shot.animation, shot.animationState = clip, NewAnimation(clip)
					shot.Sprite = clip.Frames[0].Sprite
				}
			}
			return
		}
		w.spawnEnemyShot(event.ShotX, event.ShotY, EnemyShot{Direction: event.ShotDirection, Speed: event.ShotSpeed})
		if w.Level.Rules != nil && len(w.Level.Rules.AimingTileShotSprites) == 8 && len(w.Projectiles) != 0 {
			choice := int(state.Heading) & 3
			if w.Frame&1 == 0 {
				choice += 4
			}
			w.Projectiles[0].Sprite = w.Level.Rules.AimingTileShotSprites[choice]
		}
	}
}

func (w *World) damageFifthTile(actor *WorldActor, amount uint16) {
	state, kind := actor.fifthTile, actor.fixedTileArt
	if kind.Kind == 1 && !kind.Parts[state.Part].Damageable {
		return
	}
	result := ApplyEnemyDamage(uint16(actor.Health), amount)
	actor.Health = int(result.Health)
	actor.Visible, actor.Flash = true, true
	if kind.Kind == 1 {
		actor.Patch = &kind.Parts[state.Part].Flash
	} else {
		actor.Patch = &kind.Variants[0].Initial
	}
	if !result.Destroyed {
		return
	}
	actor.Active, actor.Visible = false, false
	if kind.Kind == 1 {
		w.spawnSecondNamedExplosion(state.X+8, state.WorldY-w.ScrollY+8, "explosion-small")
		for _, part := range actor.fifthTileGroup {
			part.Active, part.Visible = false, false
			w.storeActorResidue(part)
		}
		origin := state.X - kind.Parts[state.Part].OffsetX
		w.setSecondMapPatch(origin/16, state.WorldY/16, kind.Variants[0].Destroyed)
		w.Score += 200
	} else {
		w.spawnSecondNamedExplosion(state.X+16, state.WorldY-w.ScrollY+16, "explosion-large")
		w.replaceFifthAdjacentTiles(state.X/16, state.WorldY/16, kind.DestroyedChanges)
		w.setSecondMapPatch(state.X/16, state.WorldY/16, actor.fixedTileVariant.Destroyed)
		if kind.Kind == 4 {
			w.Score += 500
		} else {
			w.Score += 400
		}
		w.storeActorResidue(actor)
	}
}

func (w *World) storeFifthTileResidue(actor *WorldActor) {
	state := actor.fifthTile
	r := &actor.Binding.Residue
	r.X, r.Y = int16(state.X), int16(state.WorldY)
	r.Counter, r.Health = int16(state.Phase), uint16(actor.Health)
	if actor.fixedTileArt.Kind != 1 {
		if actor.fixedTileArt.Kind == 3 {
			r.Direction = int16(state.Heading)
		}
		r.SetFireState(state.Accumulator, r.FireRate())
	}
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}
