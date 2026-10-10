package engine

import "xenon2/internal/visualassets"

func (w *World) spawnThirdFixed(record visualassets.FixedEncounter) bool {
	if w.Level.Number != 3 || w.Level.FixedSprites == nil || w.Level.FixedSprites.Third == nil {
		return false
	}
	art := w.Level.FixedSprites.Third
	switch record.EnemyKind {
	case 6:
		if record.State2 < 0 || record.State2 >= 8 {
			return true
		}
		state := NewThirdCrawler(record, w.ScrollY, art.Crawler)
		actor := &WorldActor{Active: true, ActorList: "moving", Atlas: "fixed", Health: art.Crawler.Health, Score: 100, thirdCrawler: &state, part: &visualassets.ActorPart{ResourceTag: 220, DamageMode: "individual"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
		actor.X, actor.Y = float64(state.X), float64(state.Y)
		actor.Sprite = state.Animation.Sprite(art.Crawler.Animations[state.Direction])
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		if err := w.bindWorldActor(actor); err != nil {
			w.poolError = err
			return true
		}
		actor.Binding.Residue.Counter = 0
		actor.Binding.Residue.StrongHealth = false
		actor.Binding.Residue.EmitterClock = 0
		w.storeActorResidue(actor)
		w.updateSecondActorCollision(actor)
		w.Actors = append([]*WorldActor{actor}, w.Actors...)
	case 5:
		state := NewThirdCannon(record, art.Cannon)
		actor := &WorldActor{Active: true, ActorList: "moving", Health: art.Cannon.Health[0], Score: 500, thirdCannon: &state, part: &visualassets.ActorPart{ResourceTag: 248, DamageMode: "third-compound-cannon"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
		actor.X, actor.Y = float64(state.X), float64(state.WorldY-w.ScrollY)
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		if err := w.bindWorldActor(actor); err != nil {
			w.poolError = err
			return true
		}
		// The common terrain constructor preserves the reused slot's strength.
		actor.part.StrongHealth = actor.Binding.Residue.StrongHealth
		actor.Binding.Residue.Counter, actor.Binding.Residue.VerticalVelocity = 0, 0
		actor.Binding.Residue.EmitterClock = 0
		w.storeActorResidue(actor)
		w.setSecondMapPatch(state.X/16, state.WorldY/16, art.Cannon.Base)
		w.Actors = append([]*WorldActor{actor}, w.Actors...)
	case 1:
		w.spawnThirdChain(record)
	default:
		return false
	}
	return true
}

func (w *World) storeThirdCannonResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	s, r := actor.thirdCannon, &actor.Binding.Residue
	// Drawing uses a screen anchor, but native terrain actors retain world Y.
	// The fixed reward belongs to the callback and does not overwrite 0x42.
	r.X, r.Y, r.Counter, r.Health = int16(s.X), int16(s.WorldY), int16(s.Phase), s.Health
	r.SetFireState(s.FireAccumulator, r.FireRate())
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}

// The crawler uses whole-pixel motion. Its native callback writes direction and
// the high emitter byte while retaining fractional coordinates and spare words.
func (w *World) storeThirdCrawlerResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	s, r := actor.thirdCrawler, &actor.Binding.Residue
	r.X, r.Y, r.Direction = int16(s.X), int16(s.Y), int16(s.Direction)
	r.Health, r.PowerOrScore, r.WaveBonusToken = uint16(actor.Health), uint16(actor.Score), actor.WaveToken
	r.SetFireState(s.FireAccumulator, r.FireRate())
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}

func (w *World) advanceThirdCrawler(actor *WorldActor) {
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	art := w.Level.FixedSprites.Third.Crawler
	state := actor.thirdCrawler
	event := state.Advance(ThirdCrawlerInput{ScrollDelta: w.ScrollDelta, ScrollY: w.ScrollY, MaximumScrollY: w.MaximumScrollY, PlayerX: w.Player.X, PlayerY: w.Player.Y, Columns: w.Level.Terrain.Columns, Map: w.Level.Terrain.Map}, art, &w.random)
	actor.Active, actor.Visible = !state.Removed, !state.Removed
	actor.X, actor.Y = float64(state.X), float64(state.Y)
	actor.Sprite = state.Animation.Sprite(art.Animations[state.Direction])
	w.updateSecondActorCollision(actor)
	actor.Binding.Residue.SetFireState(state.FireAccumulator, actor.Binding.Residue.FireRate())
	if event.Shot {
		w.spawnEnemyShot(event.X, event.Y, EnemyShot{Speed: event.Speed, Direction: event.Direction})
		if len(w.Projectiles) > 0 {
			w.Projectiles[0].Sprite, w.Projectiles[0].Atlas = art.ShotSprite, "fixed"
		}
	}
}
func (w *World) spawnThirdChain(record visualassets.FixedEncounter) {
	art := w.Level.FixedSprites.Third.Chain
	variant := record.Variant
	if variant < 0 || variant > 1 {
		return
	}
	state := NewThirdChain(record, w.ScrollY, art.Tails[variant])
	tag := 200
	if variant == 1 {
		tag = 204
	}
	sentinel := &WorldActor{Active: true, ActorList: "moving", thirdChainSentinel: true, part: &visualassets.ActorPart{ResourceTag: 188, DamageMode: "block-shot"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
	if err := w.bindWorldActor(sentinel); err != nil {
		w.poolError = err
		return
	}
	w.Actors = append([]*WorldActor{sentinel}, w.Actors...)
	var leader *WorldActor
	var group [8]*WorldActor
	for index := range 8 {
		part := state.Parts[index]
		actor := &WorldActor{Active: true, ActorList: "moving", Atlas: "fixed", Health: art.Health, Score: 400, thirdChain: &state, thirdChainPart: index + 1, part: &visualassets.ActorPart{ResourceTag: tag, DamageMode: "group"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
		actor.X, actor.Y = float64(part.X), float64(part.Y)
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		actor.Sprite = art.Bodies[variant]
		if index == 7 {
			actor.Sprite = part.Animation.Sprite(art.Tails[variant])
		}
		if leader == nil {
			leader = actor
		} else {
			actor.leader = leader
		}
		if index == 0 {
			if err := w.bindWorldActor(actor); err != nil {
				w.poolError = err
				return
			}
			w.Actors = append([]*WorldActor{actor}, w.Actors...)
		} else {
			binding, err := w.reserveWorldActor(int16(tag), ActorPoolMoving, true)
			if err != nil {
				w.poolError = err
				return
			}
			actor.ID, actor.Binding = binding.EntityID, binding
			w.Pool.unlink(binding.Slot)
			previous := group[index-1].Binding.Slot
			following := w.Pool.Next(previous)
			slot := w.Pool.Slot(binding.Slot)
			slot.list, slot.previous, slot.next = ActorPoolMoving, previous, following
			w.Pool.Slot(previous).next = binding.Slot
			if following != NoActorSlot {
				w.Pool.Slot(following).previous = binding.Slot
			} else {
				w.Pool.last[ActorPoolMoving] = binding.Slot
			}
			w.poolActors[binding.Slot] = actor
			w.Actors = append(w.Actors, actor)
		}
		w.Pool.Slot(actor.Binding.Slot).Linked = true
		actor.Binding.Residue.OwnerSlot = leader.Binding.Slot
		actor.Binding.Residue.StrongHealth = false
		actor.Binding.Residue.VerticalVelocity = int16(part.X)
		actor.Binding.Residue.FollowingSlot = NoActorSlot
		w.storeActorResidue(actor)
		w.updateSecondActorCollision(actor)
		group[index] = actor
		if index > 0 {
			previous := group[index-1]
			previous.Binding.Residue.FollowingSlot = actor.Binding.Slot
			w.storeWorldResidue(previous.Binding)
		}
	}
	leader.thirdChainMembers = group
	sentinel.leader = leader
	second := &WorldActor{Active: true, ActorList: "moving", thirdChainSentinel: true, leader: leader, part: &visualassets.ActorPart{ResourceTag: 184, DamageMode: "block-shot"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
	if err := w.bindWorldActor(second); err != nil {
		w.poolError = err
		return
	}
	w.Actors = append([]*WorldActor{second}, w.Actors...)
}

func (w *World) storeThirdChainResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	index := actor.thirdChainPart - 1
	s, r := actor.thirdChain, &actor.Binding.Residue
	r.X, r.Y = int16(s.Parts[index].X), int16(s.Parts[index].Y)
	r.Health, r.PowerOrScore, r.WaveBonusToken = uint16(actor.Health), uint16(actor.Score), actor.WaveToken
	r.Counter, r.Direction, r.VerticalFraction = 0, 0, uint16(int16(s.Parts[index].Spacing))
	if index == 0 {
		r.Counter, r.Direction, r.VerticalFraction = int16(s.Phase), int16(s.Speed), uint16(int16(s.AmplitudeOrCooldown))
	}
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}

func (w *World) advanceThirdChain(leader *WorldActor) {
	art := w.Level.FixedSprites.Third.Chain
	state := leader.thirdChain
	state.Advance(ThirdChainInput{ScrollDelta: w.ScrollDelta, PlayerY: w.Player.Y}, art, &w.random)
	for index, part := range state.Parts {
		actor := leader.thirdChainMembers[index]
		if actor == nil || !actor.Active {
			continue
		}
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		actor.X, actor.Y = float64(part.X), float64(part.Y)
		actor.Visible, actor.Active = part.Visible, !part.Removed
		actor.Sprite = art.Bodies[state.Variant]
		if index == 7 {
			actor.Sprite = part.Animation.Sprite(art.Tails[state.Variant])
		}
		w.updateSecondActorCollision(actor)
		w.storeActorResidue(actor)
	}
	if state.Parts[0].Removed {
		for _, actor := range w.Actors {
			if actor.thirdChainSentinel && actor.leader == leader {
				actor.Active = false
				w.storeActorResidue(actor)
			}
		}
	}
}
func (w *World) advanceThirdCannon(actor *WorldActor) {
	art := w.Level.FixedSprites.Third.Cannon
	state := actor.thirdCannon
	event := state.Advance(w.ScrollY, w.MaximumScrollY, art, &w.random)
	actor.Active, actor.Visible = !state.Removed, false
	actor.Collision = event.Collision
	actor.Patch = nil
	actor.Flash = false
	actor.X, actor.Y = float64(state.X), float64(state.WorldY-w.ScrollY)
	if event.WriteFrame {
		patch := art.FirstFrames[event.Frame]
		row := 4
		if state.Stage == 1 {
			patch = art.SecondFrames[event.Frame]
			row = 0
		}
		w.setSecondMapPatch(state.X/16+1, state.WorldY/16+row, patch)
	}
	actor.Binding.Residue.SetFireState(state.FireAccumulator, actor.Binding.Residue.FireRate())
	for _, direction := range event.Directions[:event.ShotCount] {
		w.spawnEnemyShot(event.X, event.Y, EnemyShot{Speed: event.Speed, Direction: direction})
		if len(w.Projectiles) == 0 {
			continue
		}
		shot := w.Projectiles[0]
		shot.Atlas = "fixed"
		if state.Stage == 0 {
			shot.Sprite = art.FirstShotSprite
		} else {
			shot.animation = art.RadialAnimations[7-direction]
			shot.animationState = NewAnimation(shot.animation)
			shot.Sprite = shot.animationState.Sprite(shot.animation)
		}
	}
}
func (w *World) damageThirdCannon(actor *WorldActor, amount uint16) {
	art := w.Level.FixedSprites.Third.Cannon
	state := actor.thirdCannon
	stage := state.Stage
	transition, removed := state.Strike(amount, art)
	actor.Health = int(state.Health)
	actor.Visible, actor.Flash = true, true
	if stage == 0 {
		actor.Patch = &art.FlashFirst
		actor.Y = float64(state.WorldY - w.ScrollY + 32)
	} else {
		actor.Patch = &art.FlashSecond
	}
	if !transition && !removed {
		return
	}
	w.Score += 500
	w.spawnSecondNamedExplosion(state.X+32, state.WorldY-w.ScrollY+64-stage*48, "explosion-large")
	if transition {
		w.setSecondMapPatch(state.X/16, state.WorldY/16, art.SecondBase)
		actor.Collision = CollisionRect{Right: -1, Bottom: -1}
	} else {
		actor.Active = false
		w.storeActorResidue(actor)
		w.setSecondMapPatch(state.X/16, state.WorldY/16, art.Destroyed)
	}
}
func (w *World) initializeThirdScenery() {
	if w.Level.FixedSprites == nil || w.Level.FixedSprites.Third == nil {
		return
	}
	actor := &WorldActor{Active: true, ActorList: "scenery", thirdScenery: true, Patch: &w.Level.FixedSprites.Third.Scenery, part: &visualassets.ActorPart{ResourceTag: 80, DamageMode: "block-shot"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
	w.Actors = append(w.Actors, actor)
}
