package engine

import "xenon2/internal/visualassets"

func (w *World) spawnFifthFormation(record visualassets.FixedEncounter) bool {
	if w.Level.Number != 5 || record.EnemyKind != 7 || w.Level.FixedSprites == nil || w.Level.FixedSprites.FifthFormation == nil {
		return false
	}
	art := w.Level.FixedSprites.FifthFormation
	if record.State2 < 0 || record.State2 >= len(art.Paths) {
		return true
	}
	for member := 0; member < 10; member++ {
		state, err := NewFifthFormation(*art, record.State2, member, w.ScrollY)
		if err != nil {
			w.poolError = err
			return true
		}
		binding, err := w.reserveWorldActor(220, ActorPoolMoving, true)
		if err != nil {
			w.poolError = err
			return true
		}
		state.Motion.X += int32(binding.Residue.XFraction)
		state.Motion.Y += int32(binding.Residue.YFraction)
		actor := &WorldActor{ID: binding.EntityID, Binding: binding, Active: true, ActorList: "moving", Atlas: "fixed", Health: art.Health, Score: 200, fifthFormation: &state, Sprite: state.Sprite, part: &visualassets.ActorPart{ResourceTag: 220, DamageMode: "individual"}, Collision: CollisionRect{Left: 1000, Right: 1000}}
		actor.X, actor.Y = float64(state.Motion.X>>16), float64(state.Motion.Y>>16)
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		w.poolActors[binding.Slot] = actor
		w.Actors = append(w.Actors, actor)
		w.storeActorResidue(actor)
	}
	return true
}

func (w *World) advanceFifthFormation(actor *WorldActor) error {
	art := w.Level.FixedSprites.FifthFormation
	state := actor.fifthFormation
	burst, err := state.Advance(*art, w.ScrollDelta, &w.Level.Paths.SineTable, func() uint16 { return uint16(w.random.Next()) })
	if err != nil {
		return err
	}
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	actor.Active, actor.Visible = state.Motion.Active, state.Motion.Active
	actor.X, actor.Y, actor.Sprite = float64(state.Motion.X>>16), float64(state.Motion.Y>>16), state.Sprite
	w.updateSecondActorCollision(actor)
	if burst {
		for direction := 7; direction >= 0; direction-- {
			w.spawnEnemyShot(int(actor.X), int(actor.Y), EnemyShot{Direction: uint8(direction), Speed: art.ShotSpeed})
			if len(w.Projectiles) > 0 {
				w.Projectiles[0].Sprite, w.Projectiles[0].Atlas = art.ShotSprite, "fixed"
			}
		}
	}
	return w.poolError
}

func (w *World) storeFifthFormationResidue(actor *WorldActor) {
	state := actor.fifthFormation
	r := &actor.Binding.Residue
	r.X, r.Y = int16(state.Motion.X>>16), int16(state.Motion.Y>>16)
	r.XFraction, r.YFraction = uint16(state.Motion.X), uint16(state.Motion.Y)
	r.Counter, r.MotionBudget = int16(state.Motion.Remaining), int16(state.Motion.Budget)
	r.Direction = int16(state.Motion.AngleFixed >> 16)
	r.Health, r.PowerOrScore = uint16(actor.Health), uint16(actor.Score)
	r.SetFireState(state.FireAccumulator, r.FireRate())
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}
