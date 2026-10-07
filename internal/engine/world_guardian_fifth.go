package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

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
	for index, descriptor := range group.Components {
		binding, err := w.reserveWorldActor(int16(descriptor.ResourceTag), ActorPoolMoving, true)
		if err != nil {
			return err
		}
		actor := &WorldActor{ID: binding.EntityID, Binding: binding, Active: true, Visible: false, ActorList: "moving", Atlas: "guardian-parts", fifthIndex: index + 1, fifthFinal: final, fifthPart: &group.Components[index], Health: descriptor.Health, Collision: CollisionRect{Right: -1, Bottom: -1}, part: &visualassets.ActorPart{ResourceTag: descriptor.ResourceTag, DamageMode: "fifth-guardian"}}
		state := w.fifthPartState(actor)
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
		actor.Active, actor.Visible = state.Active, state.Active
		actor.Sprite, actor.Health = state.Sprite, state.Health
		actor.Collision = CollisionRect{Right: -1, Bottom: -1}
		if actor.fifthPart.RenderMode == "sprite" && state.Active && !state.Destroyed {
			w.updateSecondActorCollision(actor)
		}
		if final && index == 21 && w.FifthFinal.OuterRemaining > 0 {
			actor.Collision = CollisionRect{Right: -1, Bottom: -1}
		}
		actor.Flash = state.Destroyed == false && (final && w.FifthFinal.Flash || !final && w.FifthMiddle.Flash)
		actor.Extras = actor.Extras[:0]
		if index == 0 {
			if !final && len(actor.fifthPart.TileFrames) > state.Clock {
				actor.Patch = &actor.fifthPart.TileFrames[state.Clock]
			}
			if final {
				actor.Patch = nil
				w.setFifthMouthOverlays(actor, state)
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
		actor.Extras = append(actor.Extras, WorldSpriteAttachment{Atlas: "guardian-parts", Sprite: overlay.Frames[index], X: float64(state.X + overlay.OffsetX), Y: float64(state.Y + y)})
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
	if actor.fifthFinal {
		event = w.FifthFinal.DamagePart(w.fifthFinalArt, actor.fifthIndex-1, amount)
	} else {
		event = w.FifthMiddle.Damage(actor.fifthIndex-1, amount)
	}
	state := w.fifthPartState(actor)
	actor.Health = state.Health
	actor.Flash = !state.Destroyed
	actor.Active = state.Active
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
			w.spawnSecondRandomExplosions(event.Explosions, 0, 0, 320, 192)
		} else {
			for row := 141; row < 151; row++ {
				for column := 6; column < 14; column++ {
					w.Level.Terrain.Map[row*20+column] = 0
				}
			}
			w.spawnSecondRandomExplosions(event.Explosions, 112, 2256-w.ScrollY, 96, 160)
		}
		w.spawnExitCash(event.Cash)
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
	actor := &WorldActor{X: float64(state.X), Y: float64(state.Y), PreviousX: float64(state.X), PreviousY: float64(state.Y), Active: true, Visible: false, ActorList: "moving", Atlas: "guardian-parts", fifthSeeking: &state, fifthMouth: mouth, Sprite: state.Sprite, Health: w.fifthMiddleArt.MotionParameters["side_health"], Score: 100, part: &visualassets.ActorPart{ResourceTag: 228, DamageMode: "individual"}}
	if mouth {
		actor.part.ResourceTag = 236
		actor.Health = w.fifthFinalArt.MotionParameters["mouth_creature_health"]
		actor.Score = 150
	}
	if err := w.bindWorldActor(actor); err != nil {
		w.poolError = err
		return
	}
	w.Actors = append([]*WorldActor{actor}, w.Actors...)
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
		w.Actors = append(w.Actors, actor)
	}
}
