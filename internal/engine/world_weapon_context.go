package engine

// A retained target refers to its physical entry, even if that entry now belongs
// to a linked body or another actor list. Acquisition has separate group rules.
func (w *World) currentHomingTarget(index int) (WeaponTarget, bool) {
	if w.Pool == nil {
		return WeaponTarget{}, false
	}
	slot := w.Pool.Slot(index)
	if slot == nil || !slot.allocated {
		return WeaponTarget{}, false
	}
	actor := w.poolActors[index]
	if actor == nil || actor.ID != slot.EntityID {
		return WeaponTarget{}, false
	}
	return WeaponTarget{ID: actor.ID, SlotIdentity: index + 1, ResourceTag: int(slot.ResourceTag), Bounds: actor.Collision, Active: actor.Active || slot.ResourceTag == 4}, true
}

// currentHomingTargets reads physical group flags and colliders at the moment
// of acquisition. Earlier damage can turn linked parts into separate tombstones.
func (w *World) currentHomingTargets() []WeaponTarget {
	w.weaponTargets = w.weaponTargets[:0]
	var ordered [ActorPoolCapacity]*WorldActor
	for _, actor := range w.orderedMovingActors(&ordered) {
		if actor.ActorList == "moving" {
			slotIdentity := 0
			tag, active, linked := actor.part.ResourceTag, actor.Active, actor.part.Linked
			if actor.Binding.EntityID != 0 {
				slotIdentity = actor.Binding.Slot + 1
				if slot := w.Pool.Slot(actor.Binding.Slot); slot != nil && slot.EntityID == actor.ID && slot.list == ActorPoolMoving {
					tag, linked = int(slot.ResourceTag), slot.Linked
					active = actor.Active || tag == 4
				}
			}
			if linked && actor.leader != nil {
				continue
			}
			w.weaponTargets = append(w.weaponTargets, WeaponTarget{ID: actor.ID, ResourceTag: tag, Active: active, Bounds: actor.Collision,
				SlotIdentity: slotIdentity})
		}
	}
	return w.weaponTargets
}

func (w *World) weaponContext(input Input, pulse bool) WeaponContext {
	targets := w.currentHomingTargets()
	centerX, centerY := w.currentShipCenter()
	callbacks := WeaponContext{}
	if w.Weapons != nil && w.Weapons.context.Equipment == &w.Equipment {
		callbacks = w.Weapons.context
	} else {
		callbacks = w.newWeaponCallbacks()
	}
	return WeaponContext{
		Equipment: &w.Equipment, ShipX: w.Player.X, ShipY: w.Player.Y,
		ShipCenterX: centerX, ShipCenterY: centerY, MaterializationFrames: w.MaterializationFrames,
		PreviousShipX: w.PreviousPlayer.X, PreviousShipY: w.PreviousPlayer.Y,
		TrailX: w.shipTrail[0].X, TrailY: w.shipTrail[0].Y, Motion: input.Motion,
		Held: input.Fire, Pulse: pulse, Diving: w.Dive.Phase != 0, Materializing: w.MaterializationFrames != 0,
		HomingTargetCount: &w.MovingEnemyCount,
		ReadHomingTargets: callbacks.ReadHomingTargets,
		ReadHomingTarget:  callbacks.ReadHomingTarget,
		ShipDestroyed:     !w.PlayerAlive, NextRandom: callbacks.NextRandom, Targets: targets,
		NextID: callbacks.NextID, ReserveActor: callbacks.ReserveActor, RetireActor: callbacks.RetireActor,
		ReleaseActor: callbacks.ReleaseActor, MoveActor: callbacks.MoveActor,
		ReleaseEquipmentActors: callbacks.ReleaseEquipmentActors, MoveEquipmentActors: callbacks.MoveEquipmentActors,
		StoreActorResidue: callbacks.StoreActorResidue, ReadActorResidue: callbacks.ReadActorResidue,
		ReadWeaponOwnerResidue: callbacks.ReadWeaponOwnerResidue,
		HitPoint:               callbacks.HitPoint, HitRect: callbacks.HitRect, HitLaser: callbacks.HitLaser, HitCannon: callbacks.HitCannon,
		ObservePointImpact: callbacks.ObservePointImpact,
		ObserveRectImpact:  callbacks.ObserveRectImpact,
		Sound:              callbacks.Sound, SoundVoice: callbacks.SoundVoice, SoundVoiceIfEmpty: callbacks.SoundVoiceIfEmpty,
		ImmediateSoundVoice: callbacks.ImmediateSoundVoice, EffectActive: callbacks.EffectActive, StopEffects: callbacks.StopEffects,
	}
}

// Callbacks retain this world's address, while every context rebuilds its input
// and phase values. A forecast's distinct Equipment address forces rebinding.
func (w *World) newWeaponCallbacks() WeaponContext {
	return WeaponContext{
		ReadHomingTargets: w.currentHomingTargets,
		ReadHomingTarget:  w.currentHomingTarget,
		NextRandom:        w.random.Next,
		NextID:            func() int { w.nextActorID++; return w.nextActorID },
		ReserveActor:      w.reserveWorldActor, RetireActor: w.retireWorldActor,
		ReleaseActor: w.releaseWorldActor, MoveActor: w.moveWorldActor,
		ReleaseEquipmentActors: w.releaseEquipmentActors,
		MoveEquipmentActors:    w.moveEquipmentActors,
		StoreActorResidue:      w.storeWorldResidue, ReadActorResidue: w.readWorldResidue,
		ReadWeaponOwnerResidue: w.readWeaponOwnerResidue,
		HitPoint:               w.weaponHitPoint, HitRect: w.weaponHitRect, HitLaser: w.weaponHitLaser, HitCannon: w.weaponHitCannon,
		Sound: func(effect string) { w.SoundRequests[2] = effect },
		SoundVoice: func(voice int, effect string) {
			if voice >= 0 && voice < len(w.SoundRequests) {
				w.SoundRequests[voice] = effect
			}
		},
		SoundVoiceIfEmpty: func(voice int, effect string) {
			if voice >= 0 && voice < len(w.SoundRequests) && w.SoundRequests[voice] == "" {
				w.SoundRequests[voice] = effect
			}
		},
		ImmediateSoundVoice: w.requestImmediateSound,
		EffectActive: func(voice int) bool {
			return voice >= 0 && voice < len(w.EffectActive) && w.EffectActive[voice]
		},
		StopEffects: w.requestEffectStop,
	}
}

func (w *World) weaponHitPoint(x, y int, damage uint16) bool {
	point := CollisionRect{Left: x, Top: y, Right: x, Bottom: y}
	return w.weaponHitRect(point, damage, false)
}

// weaponHitRect retains the newest-first order and skips linked body pieces
// after one damage call on their shared leader during a multi-target attack.
func (w *World) weaponHitRect(area CollisionRect, damage uint16, all bool) bool {
	return w.weaponHitRectOutcome(area, damage, all, false).Hit
}

type weaponHitOutcome struct {
	Hit, StopProjectile bool
}

func (w *World) weaponHitRectOutcome(area CollisionRect, damage uint16, all, inspectConsumption bool) weaponHitOutcome {
	var groups [159]int
	count := 0
	var outcome weaponHitOutcome
	var ordered [ActorPoolCapacity]*WorldActor
	for _, actor := range w.orderedMovingActors(&ordered) {
		if actor.ActorList != "moving" || !actor.Collision.Intersects(area) {
			continue
		}
		if !actor.Active {
			// A source death leaves its collider linked until the next moving
			// traversal. Later shots are absorbed by its now-empty callback.
			if actor.Binding.EntityID == 0 || w.Pool == nil {
				continue
			}
			slot := w.Pool.Slot(actor.Binding.Slot)
			if slot == nil || slot.EntityID != actor.ID || slot.ResourceTag != 4 || slot.list != ActorPoolMoving {
				continue
			}
			outcome.Hit = true
			if !all {
				return outcome
			}
			continue
		}
		leader := actor
		if actor.leader != nil && actor.part.DamageMode == "group" {
			leader = actor.leader
		}
		seen := false
		for _, id := range groups[:count] {
			seen = seen || id == leader.ID
		}
		if seen {
			continue
		}
		if count < len(groups) {
			groups[count] = leader.ID
			count++
		}
		outcome.Hit = true
		absorbed := inspectConsumption && w.rectangleCallbackConsumes(actor, area)
		if actor.fourthIndex > 0 {
			w.damageFourthGuardian(actor, area, damage)
		} else if actor.firstGuardian {
			w.strikeFirstGuardian(area, damage)
		} else {
			w.damageActor(actor, damage)
		}
		if absorbed {
			outcome.StopProjectile = true
			return outcome
		}
		if !all {
			return outcome
		}
	}
	if w.strikeSecondTerrain(area) {
		outcome.Hit = true
	}
	return outcome
}
