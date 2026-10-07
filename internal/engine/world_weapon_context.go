package engine

func (w *World) weaponContext(input Input, pulse bool) WeaponContext {
	w.weaponTargets = w.weaponTargets[:0]
	if w.weaponTargetActors == nil {
		w.weaponTargetActors = make(map[int]*WorldActor)
	}
	clear(w.weaponTargetActors)
	var ordered [ActorPoolCapacity]*WorldActor
	for _, actor := range w.orderedMovingActors(&ordered) {
		if actor.ActorList == "moving" && !(actor.part.Linked && actor.leader != nil) {
			slotIdentity := 0
			if actor.Binding.EntityID != 0 {
				slotIdentity = actor.Binding.Slot + 1
			}
			w.weaponTargetActors[actor.ID] = actor
			w.weaponTargets = append(w.weaponTargets, WeaponTarget{ID: actor.ID, ResourceTag: actor.part.ResourceTag, Active: actor.Active, Bounds: actor.Collision,
				SlotIdentity: slotIdentity})
		}
	}
	centerX, centerY := w.currentShipCenter()
	return WeaponContext{
		Equipment: &w.Equipment, ShipX: w.Player.X, ShipY: w.Player.Y,
		ShipCenterX: centerX, ShipCenterY: centerY, MaterializationFrames: w.MaterializationFrames,
		PreviousShipX: w.PreviousPlayer.X, PreviousShipY: w.PreviousPlayer.Y,
		TrailX: w.shipTrail[0].X, TrailY: w.shipTrail[0].Y, Motion: input.Motion,
		Held: input.Fire, Pulse: pulse, Diving: w.Dive.Phase != 0, Materializing: w.MaterializationFrames != 0,
		ShipDestroyed: !w.PlayerAlive, NextRandom: w.random.Next, Targets: w.weaponTargets,
		NextID:       func() int { w.nextActorID++; return w.nextActorID },
		ReserveActor: w.reserveWorldActor, RetireActor: w.retireWorldActor,
		StoreActorResidue: w.storeWorldResidue, ReadActorResidue: w.readWorldResidue,
		HitPoint: w.weaponHitPoint, HitRect: w.weaponHitRect, HitLaser: w.weaponHitLaser,
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
		ImmediateSoundVoice: func(voice int, effect string) {
			if voice >= 0 && voice < len(w.ImmediateSoundRequests) {
				w.ImmediateSoundRequests[voice] = effect
			}
		},
		EffectActive: func(voice int) bool {
			return voice >= 0 && voice < len(w.EffectActive) && w.EffectActive[voice]
		},
		StopEffects: func() { w.StopEffectsRequested = true },
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

func (w *World) weaponHitRectOutcome(area CollisionRect, damage uint16, all, laser bool) weaponHitOutcome {
	var groups [159]int
	count := 0
	var outcome weaponHitOutcome
	var ordered [ActorPoolCapacity]*WorldActor
	for _, actor := range w.orderedMovingActors(&ordered) {
		if !actor.Active || actor.ActorList != "moving" || !actor.Collision.Intersects(area) {
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
		absorbed := laser && w.laserCallbackConsumes(actor, area)
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
