package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

func (w *World) initializeFirstMiddle() error {
	if w.Level.Number != 1 {
		return nil
	}
	for i := range w.Level.GuardianGroups {
		if w.Level.GuardianGroups[i].ID != "middle-defense-streams" {
			continue
		}
		w.firstMiddleArt = &w.Level.GuardianGroups[i]
		if len(w.firstMiddleArt.Components) != 11 || len(w.firstMiddleArt.Launches) != 16 || len(w.firstMiddleArt.Gates) != 16 || w.Level.GuardianParts == nil {
			return fmt.Errorf("first middle arena resources are incomplete")
		}
		state := NewFirstMiddleState()
		w.FirstMiddle = &state
		for _, sprite := range w.Level.GuardianParts.Sprites {
			if sprite.Collision != nil {
				w.movingSpriteBoxes[sprite.Name] = *sprite.Collision
			}
		}
		break
	}
	return nil
}

func (w *World) advanceFirstMiddleStage() error {
	if w.FirstMiddle == nil {
		return nil
	}
	var gates [16]int
	for i, launch := range w.firstMiddleArt.Launches {
		gates[i] = launch.GateID
	}
	updated := w.FirstMiddle.Updated
	event := w.FirstMiddle.Advance(w.ScrollY, w.MaximumScrollY, w.ScrollY+w.Player.Y, gates)
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = event.Scroll, event.Maximum, event.Maximum
	if event.Crossed {
		w.ShopReady = true
		w.ScrollDeviationPasses = 0
	}
	if event.Scroll >= 2624 && event.Scroll <= 3344 {
		for stream := range updated {
			if !updated[stream] {
				if err := w.spawnFirstMiddleStream(stream, event.Launches[stream]); err != nil {
					return err
				}
			}
		}
		for i, changed := range event.GateChanged {
			if changed {
				gate := w.firstMiddleArt.Gates[i]
				w.setSecondMapPatch(gate.Column, gate.Row, gate.Frames[event.GateFrames[i]])
			}
		}
	}
	return nil
}

func (w *World) spawnFirstMiddleStream(stream, launchIndex int) error {
	launch := &w.firstMiddleArt.Launches[launchIndex]
	state, err := NewFirstMiddleAnchor(&launch.Path, *launch, 0, w.firstMiddleArt.MotionParameters["motion_budget"], w.ScrollY, w.Player.ScrollStep)
	if err != nil {
		return err
	}
	markers := &[2]*WorldActor{}
	// One trailing marker is reserved before the anchor; its type changes to188
	// only after the chain and leading184 marker have been constructed.
	for _, tag := range []int{188, 268} {
		actor := &WorldActor{Active: true, ActorList: "moving", Visible: false, firstMiddleSentinel: tag == 188, firstMiddleStream: stream, part: &visualassets.ActorPart{ResourceTag: tag, DamageMode: "block-shot"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
		if tag == 268 {
			actor.firstMiddleAnchor = &state
			actor.path = &launch.Path
			actor.X, actor.Y = float64(state.Motion.X>>16), float64(state.Motion.Y>>16)
			actor.firstMiddleStream, actor.firstMiddleGate = stream, launch.GateState
		}
		if err := w.bindWorldActor(actor); err != nil {
			return err
		}
		if actor.firstMiddleAnchor != nil {
			// The constructor replaces whole pixels and retains each physical
			// slot's fractions, just like ordinary directional projectiles.
			state.Motion.X += int32(actor.Binding.Residue.XFraction)
			state.Motion.Y += int32(actor.Binding.Residue.YFraction)
			state.Motion.AngleFixed = int32(actor.Binding.Residue.HorizontalDriftRemainder)<<16 | int32(uint16(actor.Binding.Residue.Direction))
			actor.Binding.Residue.Counter, actor.Binding.Residue.MotionBudget = 0, int16(state.Motion.Budget)
			actor.Binding.Residue.PowerOrScore, actor.Binding.Residue.EmitterClock = 0, 0
			actor.Binding.Residue.VerticalVelocity = int16(launch.GateState)
			w.storeWorldResidue(actor.Binding)
		}
		actor.firstMiddleMarkers = markers
		if tag == 188 {
			markers[1] = actor
		}
		w.Actors = append([]*WorldActor{actor}, w.Actors...)
	}
	anchor := w.Actors[0]
	previous := anchor
	for i := range w.firstMiddleArt.Components {
		descriptor := &w.firstMiddleArt.Components[i]
		follower := FirstMiddleFollower{X: state.Motion.X, Y: state.Motion.Y, Remaining: descriptor.InitialDelay}
		actor := &WorldActor{Active: true, ActorList: "moving", Atlas: "guardian-parts", Health: 1, Score: descriptor.Score, Sprite: descriptor.Sprite,
			firstMiddleFollower: &follower, firstMiddleStream: stream, firstMiddleGate: launch.GateState, leader: previous,
			part: &visualassets.ActorPart{ResourceTag: descriptor.ResourceTag, DamageMode: "first-defense"}, firstMiddlePart: descriptor,
			X: float64(follower.X >> 16), Y: float64(follower.Y >> 16), Collision: CollisionRect{Right: -1, Bottom: -1}}
		if err := w.bindWorldActor(actor); err != nil {
			return err
		}
		follower.X = follower.X&^0xffff | int32(actor.Binding.Residue.XFraction)
		follower.Y = follower.Y&^0xffff | int32(actor.Binding.Residue.YFraction)
		follower.AngleFixed = int32(actor.Binding.Residue.HorizontalDriftRemainder)<<16 | int32(uint16(actor.Binding.Residue.Direction))
		actor.Binding.Residue.Counter, actor.Binding.Residue.MotionBudget = int16(follower.Remaining), 4
		actor.Binding.Residue.Health, actor.Binding.Residue.PowerOrScore = uint16(actor.Health), uint16(actor.Score)
		actor.Binding.Residue.WaveBonusToken, actor.Binding.Residue.EmitterClock, actor.Binding.Residue.StrongHealth = 0, 0, false
		actor.Binding.Residue.VerticalVelocity = int16(launch.GateState)
		w.storeWorldResidue(actor.Binding)
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		// The original constructor publishes a collision prefix even while
		// the delayed follower's renderer is hidden.
		w.updateSecondActorCollision(actor)
		actor.firstMiddleMarkers = markers
		previous = actor
		w.Actors = append([]*WorldActor{actor}, w.Actors...)
	}
	sentinel := &WorldActor{Active: true, ActorList: "moving", firstMiddleSentinel: true, firstMiddleStream: stream, part: &visualassets.ActorPart{ResourceTag: 184, DamageMode: "block-shot"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
	if err := w.bindWorldActor(sentinel); err != nil {
		return err
	}
	markers[0] = sentinel
	sentinel.firstMiddleMarkers = markers
	w.Actors = append([]*WorldActor{sentinel}, w.Actors...)
	return nil
}

func (w *World) advanceFirstMiddleAnchor(actor *WorldActor) error {
	w.FirstMiddle.Updated[actor.firstMiddleStream] = true
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	if err := actor.firstMiddleAnchor.Advance(actor.path, &w.Level.Paths.SineTable, w.ScrollDelta, &w.random); err != nil {
		return err
	}
	actor.Active = !actor.firstMiddleAnchor.Removed
	actor.X, actor.Y = float64(actor.firstMiddleAnchor.Motion.X>>16), float64(actor.firstMiddleAnchor.Motion.Y>>16)
	if !actor.Active {
		w.FirstMiddle.GateCounters[actor.firstMiddleGate] = max(w.FirstMiddle.GateCounters[actor.firstMiddleGate], 20)
		w.releaseWorldBinding(&actor.Binding)
	}
	return nil
}

func (w *World) advanceFirstMiddleFollower(actor *WorldActor) {
	w.FirstMiddle.Updated[actor.firstMiddleStream] = true
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	next := w.Pool.Next(actor.Binding.Slot)
	parent := actor.leader
	if next != NoActorSlot {
		if candidate := w.poolActors[next]; candidate != nil {
			parent = candidate
		}
	}
	if parent == nil || parent.firstMiddleSentinel {
		actor.Active = false
		w.releaseWorldBinding(&actor.Binding)
		if w.FirstMiddle.GateCounters[actor.firstMiddleGate] == 0 {
			w.FirstMiddle.GateCounters[actor.firstMiddleGate] = 20
		}
		return
	}
	var anchor FirstMiddleAnchor
	if parent.firstMiddleAnchor != nil {
		anchor = *parent.firstMiddleAnchor
	} else {
		s := parent.firstMiddleFollower
		if s == nil {
			actor.Active = false
			return
		}
		anchor = FirstMiddleAnchor{Motion: PathMotionState{X: s.X, Y: s.Y, AngleFixed: s.AngleFixed, Remaining: s.Remaining}, Removed: s.Removed}
	}
	actor.firstMiddleFollower.Advance(anchor, w.ScrollDelta)
	s := actor.firstMiddleFollower
	actor.Active, actor.Visible = !s.Removed, s.Visible && !s.Removed
	actor.X, actor.Y = float64(s.X>>16), float64(s.Y>>16)
	heading := (int(uint32(s.AngleFixed)>>16) + 16) >> 5 & 7
	actor.Sprite = actor.firstMiddlePart.HeadingFrames[heading]
	if actor.Active {
		w.updateSecondActorCollision(actor)
	}
	if !actor.Active && w.FirstMiddle.GateCounters[actor.firstMiddleGate] == 0 {
		w.FirstMiddle.GateCounters[actor.firstMiddleGate] = 20
	}
}

func (w *World) advanceFirstMiddleMarker(actor *WorldActor) {
	if actor.part.ResourceTag != 188 || actor.firstMiddleMarkers == nil {
		return
	}
	head := actor.firstMiddleMarkers[0]
	if head == nil || head.Binding.EntityID == 0 || w.Pool.Next(head.Binding.Slot) != actor.Binding.Slot {
		return
	}
	// The original trailing marker removes the empty pair when its own
	// callback is reached. Releasing it from an earlier follower invalidates
	// the traversal's saved-next slot and skips later groups in this pass.
	actor.Active, head.Active = false, false
	w.releaseWorldBinding(&actor.Binding)
	w.releaseWorldBinding(&head.Binding)
}

func (w *World) damageFirstMiddleFollower(actor *WorldActor, amount uint16) {
	result := ApplyEnemyDamage(uint16(actor.Health), amount)
	actor.Health = int(result.Health)
	if !result.Destroyed {
		return
	}
	actor.Active = false
	w.storeActorResidue(actor)
	state := FirstMiddleFragment{X: int(actor.X), Y: int(actor.Y), Heading: uint8(w.random.Next())}
	fragment := &WorldActor{Active: true, Visible: true, ActorList: "transient", Atlas: "guardian-parts", firstMiddleFragment: &state,
		X: actor.X, Y: actor.Y, PreviousX: actor.X, PreviousY: actor.Y, animation: actor.firstMiddlePart.DeathAnimation, animationState: NewAnimation(actor.firstMiddlePart.DeathAnimation), part: &visualassets.ActorPart{ResourceTag: actor.part.ResourceTag, DamageMode: "block-shot"}}
	if err := w.bindWorldActor(fragment); err != nil {
		w.poolError = err
		return
	}
	fragment.Sprite = fragment.animationState.Sprite(fragment.animation)
	w.Actors = append([]*WorldActor{fragment}, w.Actors...)
	w.Score += actor.Score
}

func (w *World) advanceFirstMiddleFragment(actor *WorldActor) {
	actor.firstMiddleFragment.Advance(&w.Level.Paths.SineTable)
	actor.animationState.Advance(actor.animation)
	actor.Active = !actor.firstMiddleFragment.Removed
	actor.X, actor.Y = float64(actor.firstMiddleFragment.X), float64(actor.firstMiddleFragment.Y)
	actor.selectSprite()
}
