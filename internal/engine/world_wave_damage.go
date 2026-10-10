package engine

import "xenon2/internal/visualassets"

// Wave damage callbacks can allocate while the hit actor still owns its entry.
// Later writes therefore follow the physical slot even if a new effect or
// reward has replaced that actor during the callback.
func (w *World) waveDamageSlot(actor *WorldActor) int {
	if actor.Binding.EntityID != 0 && w.Pool != nil {
		if slot := w.Pool.Slot(actor.Binding.Slot); slot != nil && slot.allocated && slot.EntityID == actor.ID {
			return actor.Binding.Slot
		}
	}
	return NoActorSlot
}

func (w *World) waveDeathState(actor *WorldActor, index int) ActorResidue {
	state := ActorResidue{FollowingSlot: NoActorSlot}
	if w.Pool != nil && index != NoActorSlot {
		state = w.Pool.Slot(index).Residue
	}
	if index == NoActorSlot || w.Pool.Slot(index).EntityID == actor.ID {
		// Semantic actor fields remain authoritative while it owns the slot;
		// synthetic diagnostics may not have published a formation constructor.
		// After replacement, the current physical owner supplies these values.
		state.X, state.Y = int16(actor.X), int16(actor.Y)
		state.Health, state.PowerOrScore, state.WaveBonusToken = uint16(actor.Health), uint16(actor.Score), actor.WaveToken
		if actor.part != nil {
			state.StrongHealth = actor.part.StrongHealth
			if actor.part.DamageMode == "drop-equipment" {
				state.VerticalFraction = uint16(actor.CarriedReward)
			}
		}
	}
	return state
}

func (w *World) retireWaveDamageSlot(index int) {
	if index == NoActorSlot || w.Pool == nil {
		return
	}
	slot := w.Pool.Slot(index)
	if actor := w.poolActors[index]; actor != nil && actor.ID == slot.EntityID {
		actor.Active = false
	}
	if item := w.poolCollectibles[index]; item != nil && item.ID == slot.EntityID {
		item.Active = false
	}
	if shot := w.poolProjectiles[index]; shot != nil && shot.ID == slot.EntityID {
		shot.Active = false
	}
	if shot := w.poolSmallShots[index]; shot != nil && shot.ID == slot.EntityID {
		shot.Active = false
	}
	if err := w.Pool.MarkDead(index); err != nil {
		w.poolError = err
	}
}

func (w *World) waveDeathRegion(actor *WorldActor, index int) visualassets.SpriteRegion {
	if index != NoActorSlot {
		if current := w.poolActors[index]; current != nil {
			return w.actorRegion(current)
		}
		if current := w.poolCollectibles[index]; current != nil {
			return w.commonSpriteRegion(current.Sprite)
		}
	}
	return w.actorRegion(actor)
}

func (w *World) spawnWaveDeathEffect(actor *WorldActor, index int, strong bool) {
	state, image := w.waveDeathState(actor, index), w.waveDeathRegion(actor, index)
	name := "explosion-small"
	if strong {
		name = "explosion-large"
	}
	w.spawnSecondNamedExplosion(int(state.X)-image.AnchorX+image.Width/2, int(state.Y)-image.AnchorY+(image.Height-1)/2, name)
}

func (w *World) finishWaveDeath(actor *WorldActor, index int) {
	state := w.waveDeathState(actor, index)
	w.spawnWaveDeathEffect(actor, index, state.StrongHealth)
	state = w.waveDeathState(actor, index)
	if w.WaveBonuses.Defeat(state.WaveBonusToken) {
		w.spawnWaveCash(int(state.X), int(state.Y), state.StrongHealth)
	}
	state = w.waveDeathState(actor, index)
	w.Score += int(state.PowerOrScore)
	actor.Active = false
	w.retireWaveDamageSlot(index)
}

func (w *World) damageWaveActor(actor *WorldActor, amount uint16) {
	if actor.part != nil && actor.part.DamageMode == "drop-equipment" {
		index := w.waveDamageSlot(actor)
		state := w.waveDeathState(actor, index)
		reward := actor.CarriedReward
		if index != NoActorSlot {
			reward = int(state.VerticalFraction)
		}
		w.spawnPickup(reward, int(state.X), int(state.Y))
		actor.Active = false
		w.retireWaveDamageSlot(index)
		return
	}
	target := actor
	group := actor.part != nil && actor.part.DamageMode == "group"
	if group && actor.leader != nil {
		target = actor.leader
	}
	index := w.waveDamageSlot(target)
	physicalGroup := group && index != NoActorSlot && w.Pool.Slot(index).Residue.OwnerSlot == index
	result := ApplyEnemyDamage(uint16(target.Health), amount)
	target.Health, target.Flash = int(result.Health), true
	if index != NoActorSlot {
		// The subtraction is visible immediately, but the resource tag remains
		// live until the death callback finishes all of its allocations.
		target.Binding.Residue = w.Pool.Slot(index).Residue
		target.Binding.Residue.Health = result.Health
		w.storeWorldResidue(target.Binding)
	}
	if physicalGroup {
		for member, count := index, 0; member != NoActorSlot && count < ActorPoolCapacity; count++ {
			if current := w.poolActors[member]; current != nil {
				current.Flash = true
			}
			member = w.Pool.Slot(member).Residue.FollowingSlot
		}
	}
	if !result.Destroyed {
		return
	}
	if physicalGroup {
		for member, count := w.Pool.Slot(index).Residue.FollowingSlot, 0; member != NoActorSlot && count < ActorPoolCapacity; count++ {
			if !w.Pool.Slot(member).SkipDeathEffect {
				w.spawnWaveDeathEffect(target, member, w.Pool.Slot(index).Residue.StrongHealth)
			}
			next := w.Pool.Slot(member).Residue.FollowingSlot
			w.retireWaveDamageSlot(member)
			member = next
		}
	} else if group {
		// Diagnostic groups may expose only logical members, without the
		// constructors' physical following links.
		for _, member := range append([]*WorldActor(nil), w.Actors...) {
			if member != target && member.leader == target {
				if !member.part.Linked {
					w.spawnActorDeathEffect(member)
				}
				member.Active = false
				w.storeActorResidue(member)
			}
		}
	}
	w.finishWaveDeath(target, index)
}
