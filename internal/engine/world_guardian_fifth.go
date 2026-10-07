package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

// The middle body's normal renderer mutates its muzzle tile table. A flash
// draws the last normal table instead, even when the firing clock has advanced.
// Keep the current pass's normal choice pending until all damage callbacks have
// selected its renderer; drawing the resulting frame never mutates this state.
type fifthBodyRenderState struct {
	normal, pending *visualassets.TilePatch
	pass            uint64
}

func (w *World) fifthMiddleBodyRender(body *WorldActor) *fifthBodyRenderState {
	if body.fifthBodyRender == nil {
		body.fifthBodyRender = &fifthBodyRenderState{normal: &body.fifthPart.TileFrames[0]}
	}
	render := body.fifthBodyRender
	if render.pending != nil && render.pass != w.Frame {
		if !body.Flash || body.fifthFlashPass != render.pass {
			render.normal = render.pending
		}
		render.pending = nil
	}
	return render
}

func (w *World) initializeFifthStage() error {
	if w.Level.Number != 5 || len(w.Level.GuardianGroups) == 0 {
		return nil
	}
	for i := range w.Level.GuardianGroups {
		group := &w.Level.GuardianGroups[i]
		if group.ID == "middle-guardian" {
			w.fifthMiddleArt = group
		}
		if group.ID == "final-guardian" {
			w.fifthFinalArt = group
		}
	}
	if w.fifthMiddleArt == nil || w.fifthFinalArt == nil || w.Level.GuardianParts == nil {
		return fmt.Errorf("fifth guardian resources incomplete")
	}
	for _, sprite := range w.Level.GuardianParts.Sprites {
		if sprite.Collision != nil {
			w.movingSpriteBoxes[sprite.Name] = *sprite.Collision
		}
	}
	return nil
}

func (w *World) activateFifthGuardian(record visualassets.FixedEncounter, final bool) error {
	if final && w.FifthFinal != nil || !final && w.FifthMiddle != nil {
		return nil
	}
	group := w.fifthMiddleArt
	if final {
		group = w.fifthFinalArt
		state, err := NewFifthFinalGuardianState(group)
		if err != nil {
			return err
		}
		w.FifthFinal = &state
		w.Checkpoint.ScrollY, w.Checkpoint.PlayerX, w.Checkpoint.WorldY = 416, 160, 176
		w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 416, 416, 416
		w.prepareFifthArena(group)
	} else {
		state, err := NewFifthMiddleGuardianState(group, record.Y-8-w.ScrollY)
		if err != nil {
			return err
		}
		w.FifthMiddle = &state
	}
	bodySlot := NoActorSlot
	for index, descriptor := range group.Components {
		binding, err := w.reserveWorldActor(int16(descriptor.ResourceTag), ActorPoolMoving, true)
		if err != nil {
			return err
		}
		actor := &WorldActor{ID: binding.EntityID, Binding: binding, Active: true, Visible: false, ActorList: "moving", Atlas: "guardian-parts", fifthIndex: index + 1, fifthFinal: final, fifthPart: &group.Components[index], Health: descriptor.Health, Collision: CollisionRect{Right: -1, Bottom: -1}, part: &visualassets.ActorPart{ResourceTag: descriptor.ResourceTag, StrongHealth: descriptor.StrongHealth, DamageMode: "fifth-guardian"}}
		state := w.fifthPartState(actor)
		if index == 0 {
			bodySlot = binding.Slot
		}
		actor.Binding.Residue.OwnerSlot = bodySlot
		// These whole-pixel constructors retain both emitter bytes and spare
		// physical words while initializing phase, direction and component data.
		state.FireAccumulator, state.SecondaryAccumulator = binding.Residue.FireAccumulator(), binding.Residue.FireRate()
		actor.Binding.Residue.Counter, actor.Binding.Residue.Direction = 0, 4
		actor.Binding.Residue.MountOffsetX, actor.Binding.Residue.MountOffsetY = int16(descriptor.OffsetX), int16(descriptor.OffsetY)
		actor.Binding.Residue.WaveBonusToken = 0
		if final {
			actor.Binding.Residue.StrongHealth = descriptor.StrongHealth
		} else {
			actor.part.StrongHealth = binding.Residue.StrongHealth
		}
		actor.X, actor.Y = float64(state.X), float64(state.Y)
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		actor.Sprite = state.Sprite
		w.poolActors[binding.Slot] = actor
		w.storeActorResidue(actor)
		if final {
			w.fifthFinalActors[index] = actor
		} else {
			w.fifthMiddleActors[index] = actor
		}
		w.Actors = append(w.Actors, actor)
	}
	return nil
}

func (w *World) storeFifthGuardianResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	s, r := w.fifthPartState(actor), &actor.Binding.Residue
	r.X, r.Y, r.Health, r.Counter = int16(s.X), int16(s.Y), uint16(s.Health), int16(s.Clock)
	switch actor.fifthPart.Behavior {
	case "middle-body-controller":
		r.Direction = int16(s.MoveRemaining)
	case "follow-middle-body", "final-mount":
		r.Direction = int16(s.Heading)
	}
	r.SetFireState(s.FireAccumulator, s.SecondaryAccumulator)
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}

func (w *World) fifthPartState(actor *WorldActor) *FifthGuardianPartState {
	if actor.fifthFinal {
		return &w.FifthFinal.Parts[actor.fifthIndex-1]
	}
	return &w.FifthMiddle.Parts[actor.fifthIndex-1]
}

func (w *World) advanceFifthGuardian(final bool) {
	var event FifthGuardianEvents
	var creatures []FifthMouthCreature
	var group *visualassets.GuardianGroup
	var middleRender *fifthBodyRenderState
	if final {
		group = w.fifthFinalArt
		result := w.FifthFinal.Advance(group, w.ScrollY, w.ScrollDelta, w.BaseScrollStep, w.MaximumScrollY, int(w.Frame), w.Player.X, w.Player.Y, &w.random)
		event, creatures = result.FifthGuardianEvents, result.Creatures
		if result.ExtraBackdropStep != 0 {
			step := (result.ExtraBackdropStep >> 1) + (int(w.Frame&1) & result.ExtraBackdropStep)
			w.BackgroundY = (w.BackgroundY - step + 192) % 192
			if w.BackgroundStars != nil {
				w.BackgroundStars.Advance(result.ExtraBackdropStep)
			}
		}
	} else {
		group = w.fifthMiddleArt
		// Finalize the preceding pass before clearing its flash request. Player
		// contact may already have selected this pass's replacement renderer.
		middleRender = w.fifthMiddleBodyRender(w.fifthMiddleActors[0])
		event = w.FifthMiddle.Advance(group, w.ScrollY, w.ScrollDelta, w.MaximumScrollY, w.Player.X, w.Player.Y, &w.random)
	}
	w.MaximumScrollY, w.VisitedScrollY = event.MaximumScroll, event.MaximumScroll
	for index := range group.Components {
		actor := w.fifthMiddleActors[0]
		if final {
			actor = w.fifthFinalActors[index]
		} else {
			actor = w.fifthMiddleActors[index]
		}
		if actor == nil {
			continue
		}
		state := w.fifthPartState(actor)
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		actor.X, actor.Y = float64(state.X), float64(state.Y)
		actor.Active, actor.Visible = state.Active, state.Active && actor.fifthPart.RenderMode != "none"
		actor.Sprite, actor.Health = state.Sprite, state.Health
		actor.Collision = CollisionRect{Right: -1, Bottom: -1}
		if actor.fifthPart.RenderMode == "sprite" && state.Active && !state.Destroyed {
			w.updateSecondActorCollision(actor)
		}
		if actor.fifthPart.Behavior == "barrier-band" && state.Active {
			// These source actors have no renderer, but their updater publishes
			// the armor rectangle used by player and projectile callbacks.
			actor.Collision = state.Collision
		}
		if final && index == 21 && w.FifthFinal.OuterRemaining > 0 {
			actor.Collision = CollisionRect{Right: -1, Bottom: -1}
		}
		// Damage can select a renderer before this actor update (player contact)
		// or afterward (projectiles). Preserve only the current pass's request.
		actor.Flash = actor.Flash && actor.fifthFlashPass == w.Frame
		actor.Extras = actor.Extras[:0]
		if index == 0 {
			if !final && len(actor.fifthPart.TileFrames) > state.Clock {
				middleRender.pending, middleRender.pass = &actor.fifthPart.TileFrames[state.Clock], w.Frame
				actor.Patch = middleRender.pending
				if actor.Flash {
					actor.Patch = middleRender.normal
				}
			}
			if final {
				actor.Patch = nil
				if actor.Flash {
					actor.Patch = &actor.fifthPart.TileFrames[0]
				} else {
					w.setFifthMouthOverlays(actor, state)
				}
			}
		}
		w.storeActorResidue(actor)
	}
	for _, shot := range event.Shots {
		w.spawnFifthShot(shot, group)
	}
	for _, column := range event.Lasers {
		w.spawnFifthColumn(column)
	}
	for _, creature := range creatures {
		w.spawnFifthMouth(creature)
	}
	if event.Sound2 >= 0 {
		w.queueFifthSound(2, event.Sound2)
	}
	if event.Sound1 >= 0 {
		w.queueFifthSound(1, event.Sound1)
	}
}

func (w *World) setFifthMouthOverlays(actor *WorldActor, state *FifthGuardianPartState) {
	clock := state.Clock
	if clock < 0 {
		clock = -clock
	}
	if clock == 0 {
		return
	}
	for _, overlay := range actor.fifthPart.Overlays {
		aligned := clock &^ 3
		if overlay.CounterMinimum != 0 && aligned < overlay.CounterMinimum || overlay.CounterMaximum != 0 && aligned >= overlay.CounterMaximum {
			continue
		}
		index := aligned / overlay.CounterStride
		if overlay.ID == "mouth-inner" {
			index -= 5
		}
		if index < 0 || index >= len(overlay.Frames) {
			continue
		}
		y := overlay.PositiveOffsetY
		if state.Clock < 0 {
			y = overlay.NegativeOffsetY
		}
		// A mouth pose changes discretely; its placement follows the same
		// interpolated translation as the body without blending frame artwork.
		actor.Extras = append(actor.Extras, WorldSpriteAttachment{Atlas: "guardian-parts", Sprite: overlay.Frames[index], X: float64(state.X + overlay.OffsetX), Y: float64(state.Y + y), PreviousX: actor.PreviousX + float64(overlay.OffsetX), PreviousY: actor.PreviousY + float64(y), Interpolate: true})
	}
}

func (w *World) queueFifthSound(channel, value int) {
	family, index := "synthesized", value
	if value&128 != 0 {
		family, index = "sampled", value&127
	}
	w.SoundRequests[channel] = fmt.Sprintf("%s-effect-%02d", family, index)
}

func (w *World) prepareFifthArena(group *visualassets.GuardianGroup) {
	for row := 0; row < 39; row++ {
		for column := 0; column < 20; column++ {
			w.Level.Terrain.Map[row*20+column] = 0
		}
	}
	if len(group.Components[0].TileFrames) != 0 {
		patch := group.Components[0].TileFrames[0]
		for row := 0; row < patch.Rows; row++ {
			copy(w.Level.Terrain.Map[(row+6)*20+3:][:patch.Columns], patch.Tiles[row*patch.Columns:][:patch.Columns])
		}
	}
}

func (w *World) damageFifthGuardian(actor *WorldActor, amount uint16) {
	var event FifthGuardianEvents
	wasDestroyed := w.fifthPartState(actor).Destroyed
	if actor.fifthFinal {
		event = w.FifthFinal.DamagePart(w.fifthFinalArt, actor.fifthIndex-1, amount)
	} else {
		event = w.FifthMiddle.Damage(actor.fifthIndex-1, amount)
	}
	state := w.fifthPartState(actor)
	actor.Health = state.Health
	actor.Active = state.Active
	index := actor.fifthIndex - 1
	if actor.fifthFinal && index == 21 || !actor.fifthFinal && index == 5 {
		body := w.fifthMiddleActors[0]
		if actor.fifthFinal {
			body = w.fifthFinalActors[0]
			body.Health = w.FifthFinal.Parts[0].Health
			// The source core callback changes its health image immediately,
			// while its already-published collision rectangle remains intact.
			actor.Sprite = state.Sprite
		}
		if state.Active {
			// Core damage selects the owner's tile renderer, including the final
			// body normally drawn only as scenery. Mouth overlays resume later.
			if actor.fifthFinal {
				body.Patch = &body.fifthPart.TileFrames[0]
				body.Extras = body.Extras[:0]
			} else {
				body.Patch = w.fifthMiddleBodyRender(body).normal
			}
			body.Flash, body.fifthFlashPass = true, w.Frame
		}
		w.storeActorResidue(body)
	} else if !wasDestroyed && (actor.fifthFinal && index >= 3 && index < 21 || !actor.fifthFinal && index >= 1 && index <= 4) {
		// Mount callbacks select the individual flash renderer even when the
		// hit leaves a retained wreck. Harmless bodies, corners and bands do not.
		actor.Flash, actor.fifthFlashPass = state.Active, w.Frame
	}
	if state.Destroyed && state.Active {
		// Mount wrecks keep their actor and current image, but stop intercepting
		// later shots immediately, before the next moving-actor callback.
		actor.Collision.Left, actor.Collision.Right = 1000, 1000
	}
	w.Score += event.Score
	if event.Explosions == 1 {
		w.spawnActorDeathEffect(actor)
	}
	if event.Defeated {
		for _, member := range w.Actors {
			if member.Active && member.ActorList == "moving" {
				member.Active = false
				w.storeActorResidue(member)
			}
		}
		if actor.fifthFinal {
			for row := 0; row < 39; row++ {
				for column := 0; column < 20; column++ {
					w.Level.Terrain.Map[row*20+column] = 0
				}
			}
			w.LevelFinished = true
			w.spawnExitCash(event.Cash)
			w.spawnSecondRandomExplosions(event.Explosions, 0, 0, 320, 192)
		} else {
			for row := 141; row < 151; row++ {
				for column := 6; column < 14; column++ {
					w.Level.Terrain.Map[row*20+column] = 0
				}
			}
			w.spawnSecondRandomExplosions(event.Explosions, 112, 2256-w.ScrollY, 96, 160)
			w.spawnExitCash(event.Cash)
		}
	}
	w.storeActorResidue(actor)
}

func (w *World) spawnFifthShot(shot FifthGuardianShot, group *visualassets.GuardianGroup) {
	if shot.Animation == "middle-side-shot" {
		w.spawnFifthSide(shot)
		return
	}
	w.spawnEnemyShot(shot.X, shot.Y, EnemyShot{Speed: shot.Speed, Direction: shot.Heading})
}

func (w *World) spawnFifthSide(shot FifthGuardianShot) {
	clip := w.fifthMiddleArt.Components[0].HeadingAnimations[shot.Heading]
	state := FifthSeekingState{X: shot.X, Y: shot.Y, Heading: shot.Heading, Clock: 0, Active: true, Animation: NewAnimation(clip), Sprite: clip.Frames[0].Sprite}
	// The source constructor consumes this clock draw after allocating its slot.
	state.Clock = shot.InitialClock
	w.addFifthSeeking(state, false)
}
func (w *World) spawnFifthMouth(creature FifthMouthCreature) {
	clip := w.fifthFinalArt.Components[0].HeadingAnimations[0]
	state := FifthSeekingState{X: creature.X, Y: creature.Y, Clock: 1, Active: true, Animation: NewAnimation(clip), Sprite: clip.Frames[0].Sprite}
	w.addFifthSeeking(state, true)
}
func (w *World) addFifthSeeking(state FifthSeekingState, mouth bool) {
	actor := &WorldActor{X: float64(state.X), Y: float64(state.Y), PreviousX: float64(state.X), PreviousY: float64(state.Y), Active: true, Visible: false, ActorList: "moving", Atlas: "guardian-parts", fifthSeeking: &state, fifthMouth: mouth, Sprite: state.Sprite, Health: w.fifthMiddleArt.MotionParameters["side_health"], Score: 100, Collision: CollisionRect{Left: 1000, Right: 1000}, part: &visualassets.ActorPart{ResourceTag: 228, DamageMode: "individual"}}
	if mouth {
		actor.part.ResourceTag = 236
		actor.Health = w.fifthFinalArt.MotionParameters["mouth_creature_health"]
		actor.Score = 150
	}
	if mouth {
		binding, err := w.reserveWorldActor(236, ActorPoolMoving, true)
		if err != nil {
			w.poolError = err
			return
		}
		actor.ID, actor.Binding = binding.EntityID, binding
		w.poolActors[binding.Slot] = actor
		w.Actors = append(w.Actors, actor)
	} else {
		if err := w.bindWorldActor(actor); err != nil {
			w.poolError = err
			return
		}
		w.Actors = append([]*WorldActor{actor}, w.Actors...)
	}
	w.storeActorResidue(actor)
}
func (w *World) advanceFifthSeeking(actor *WorldActor) {
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	state := actor.fifthSeeking
	group, outer := w.fifthMiddleArt, 18
	if actor.fifthMouth {
		group, outer = w.fifthFinalArt, w.FifthFinal.OuterRemaining
	}
	event := state.AdvanceContact(group, actor.fifthMouth, w.fixedProjectileInputs(), outer, func(name string) visualassets.CollisionBox { return w.movingSpriteBoxes[name] }, func(name string) visualassets.SpriteRegion {
		if w.Level.GuardianParts != nil {
			for _, region := range w.Level.GuardianParts.Sprites {
				if region.Name == name {
					return region
				}
			}
		}
		return visualassets.SpriteRegion{}
	})
	actor.X, actor.Y = float64(state.X), float64(state.Y)
	actor.Active, actor.Visible = state.Active, state.Active
	actor.Sprite = state.Sprite
	actor.Collision = event.Collision
	if event.Explosion {
		w.spawnSecondExplosion(event.ExplosionX, event.ExplosionY)
	}
	if event.PlayerDamage != 0 {
		w.damagePlayer(event.PlayerDamage)
	}
}

func (w *World) storeFifthSeekingResidue(actor *WorldActor) {
	state := actor.fifthSeeking
	r := &actor.Binding.Residue
	r.X, r.Y = int16(state.X), int16(state.Y)
	r.Counter, r.Direction = int16(state.Clock), int16(state.Heading)
	r.Health, r.PowerOrScore = uint16(actor.Health), uint16(actor.Score)
	r.StrongHealth = false
	r.WaveBonusToken = 0
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}
func (w *World) spawnFifthColumn(event FifthGuardianLaser) {
	state := FifthLaserColumnState{X: event.X, Y: event.Y, Speed: event.Speed, Active: true}
	binding, err := w.reserveWorldActor(272, ActorPoolProjectile, true)
	if err != nil {
		w.poolError = err
		return
	}
	actor := &WorldActor{ID: binding.EntityID, Binding: binding, X: float64(state.X), Y: float64(state.Y), PreviousX: float64(state.X), PreviousY: float64(state.Y), Active: true, ActorList: "transient", Atlas: "guardian-parts", fifthColumn: &state, part: &visualassets.ActorPart{ResourceTag: 272, DamageMode: "block-shot"}}
	w.poolActors[binding.Slot] = actor
	w.Actors = append(w.Actors, actor)
	w.storeFifthColumnResidue(actor)
}

func (w *World) storeFifthColumnResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	s, r := actor.fifthColumn, &actor.Binding.Residue
	r.X, r.Y, r.Counter, r.Direction = int16(s.X), int16(s.Y), int16(s.Length), int16(s.Speed)
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}

func (w *World) advanceFifthColumn(actor *WorldActor) {
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	state := actor.fifthColumn
	state.Advance(w.ScrollDelta, w.fifthMiddleArt.MotionParameters["body_laser_speed"])
	actor.X, actor.Y = float64(state.X), float64(state.Y)
	actor.Active, actor.Visible = state.Active, state.Active
	actor.Collision = state.Collision
	actor.DrawKind = "fifth-column"
	actor.DrawLength = state.Length
	actor.DrawUp = state.Speed < 0
	if state.Active && w.PlayerAlive && w.Dive.Phase == 0 && state.Collision.Intersects(w.playerCollision) {
		w.damagePlayer(6)
		actor.Active = false
	}
}

func (w *World) restoreFifthGuardianActors(scrollChange int) {
	for index, actor := range w.fifthMiddleActors {
		if actor == nil || !actor.Active || w.FifthMiddle == nil || w.FifthMiddle.Defeated {
			continue
		}
		state := &w.FifthMiddle.Parts[index]
		state.Y += scrollChange
		actor.Y += float64(scrollChange)
		actor.PreviousY = actor.Y
		shiftFifthAttachments(actor, scrollChange)
		w.Actors = append(w.Actors, actor)
	}
	for index, actor := range w.fifthFinalActors {
		if actor == nil || !actor.Active || w.FifthFinal == nil || w.FifthFinal.Defeated {
			continue
		}
		state := &w.FifthFinal.Parts[index]
		state.Y += scrollChange
		actor.Y += float64(scrollChange)
		actor.PreviousY = actor.Y
		shiftFifthAttachments(actor, scrollChange)
		w.Actors = append(w.Actors, actor)
	}
}

func shiftFifthAttachments(actor *WorldActor, scrollChange int) {
	for i := range actor.Extras {
		actor.Extras[i].Y += float64(scrollChange)
		actor.Extras[i].PreviousX, actor.Extras[i].PreviousY = actor.Extras[i].X, actor.Extras[i].Y
	}
}
