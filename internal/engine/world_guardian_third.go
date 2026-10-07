package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

func (w *World) initializeThirdStage() error {
	if w.Level.Number != 3 || len(w.Level.GuardianGroups) == 0 {
		return nil
	}
	for i := range w.Level.GuardianGroups {
		group := &w.Level.GuardianGroups[i]
		if group.ID == "middle-guardian" {
			w.thirdMiddleArt = group
		}
		if group.ID == "final-guardian" {
			w.thirdFinalArt = group
		}
	}
	if w.thirdMiddleArt == nil || w.thirdFinalArt == nil || w.Level.GuardianParts == nil {
		return fmt.Errorf("third stage guardian resources are incomplete")
	}
	final := NewThirdFinalState(w.thirdFinalArt.MotionParameters["initial_health"])
	w.ThirdFinal = &final
	for _, sprite := range w.Level.GuardianParts.Sprites {
		if sprite.Collision != nil {
			w.movingSpriteBoxes[sprite.Name] = *sprite.Collision
		}
	}
	return nil
}

// activateThirdMiddle is called by the source fixed kind3 record after traversal.
// The constructor appends its seventeen parts at the moving list's tail.
func (w *World) activateThirdMiddle() error {
	if w.ThirdMiddle != nil {
		return nil
	}
	state, err := NewThirdGuardianState(w.thirdMiddleArt)
	if err != nil {
		return err
	}
	w.ThirdMiddle = &state
	for index := range w.thirdMiddleArt.Components {
		descriptor := &w.thirdMiddleArt.Components[index]
		binding, err := w.reserveWorldActor(84, ActorPoolMoving, true)
		if err != nil {
			return err
		}
		actor := &WorldActor{ID: binding.EntityID, Binding: binding, X: float64(descriptor.InitialX), Y: float64(descriptor.InitialWorldY), PreviousX: float64(descriptor.InitialX), PreviousY: float64(descriptor.InitialWorldY), ActorList: "moving", Atlas: "guardian-parts", Active: true, Score: 1000, Health: int(binding.Residue.Health), Sprite: state.Parts[index].Sprite, thirdMiddlePart: index + 1, thirdPart: descriptor,
			part: &visualassets.ActorPart{ResourceTag: 84, StrongHealth: true, MotionMode: "third-middle", DamageMode: "block-shot"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
		actor.motion.X, actor.motion.Y = int32(descriptor.InitialX)<<16, int32(descriptor.InitialWorldY)<<16
		actor.Binding.Residue.SetFireState(0, binding.Residue.FireRate())
		if index == 0 {
			state.ResidualFireRate = binding.Residue.FireRate()
		}
		w.poolActors[binding.Slot] = actor
		w.storeActorResidue(actor)
		w.thirdMiddleActors[index] = actor
		w.Actors = append(w.Actors, actor)
	}
	w.ThirdStage.MiddleHeartbeat = true
	return nil
}

func (w *World) advanceThirdMiddle() error {
	event, err := w.ThirdMiddle.Advance(w.thirdMiddleArt, &w.Level.Paths.SineTable, w.Frame, w.Player.X, w.Player.Y, &w.random)
	if err != nil {
		return err
	}
	w.thirdMiddleUpdated = event.ActiveSignal
	for index, motion := range w.ThirdMiddle.Parts {
		actor := w.thirdMiddleActors[index]
		if actor == nil || !actor.Active {
			continue
		}
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		actor.X, actor.Y = float64(motion.X), float64(motion.Y)
		actor.Sprite = motion.Sprite
		actor.Visible = true
		actor.Flash = w.ThirdMiddle.Flash
		actor.Collision = CollisionRect{Right: -1, Bottom: -1}
		if motion.Collidable {
			w.updateSecondActorCollision(actor)
		}
		actor.motion.X, actor.motion.Y = motion.Arc.X, motion.Arc.Y
		if index == 0 {
			actor.motion.X, actor.motion.Y = w.ThirdMiddle.Flight.X, w.ThirdMiddle.Flight.Y
		}
		actor.Binding.Residue.SetFireState(motion.FireAccumulator, actor.Binding.Residue.FireRate())
		w.storeActorResidue(actor)
	}
	if event.Sound != "" {
		w.SoundRequests[2] = event.Sound
	}
	for _, shot := range event.Shots {
		w.spawnThirdShot(shot, w.thirdMiddleArt)
	}
	return nil
}

func (w *World) damageThirdMiddle(actor *WorldActor, amount uint16) {
	index := actor.thirdMiddlePart - 1
	if index != 3 && index != 4 {
		return
	}
	event := w.ThirdMiddle.StrikeEye(index-3, amount)
	if !event.Applied {
		return
	}
	for _, part := range w.thirdMiddleActors {
		if part != nil && part.Active {
			part.Flash = true
		}
	}
	if event.EyeDestroyed {
		actor.Collision = CollisionRect{Right: -1, Bottom: -1}
		x, y := w.thirdSpriteCenter(actor.Sprite, int(actor.X), int(actor.Y))
		w.spawnSecondExplosion(x, y)
	}
	if !event.Defeated {
		return
	}
	for _, part := range w.thirdMiddleActors {
		if part != nil {
			part.Active = false
			w.storeActorResidue(part)
		}
	}
	w.Score += 1000
	w.spawnSecondCashPairs(event.CashPairs, false)
}

func (w *World) thirdSpriteCenter(name string, x, y int) (int, int) {
	if w.Level.GuardianParts != nil {
		for _, sprite := range w.Level.GuardianParts.Sprites {
			if sprite.Name == name {
				return x - sprite.AnchorX + sprite.Width/2, y - sprite.AnchorY + (sprite.Height-1)/2
			}
		}
	}
	return x, y
}

func (w *World) advanceThirdStage() error {
	if w.Level.Number != 3 || w.ThirdFinal == nil {
		return nil
	}
	event := w.ThirdStage.Advance(ThirdStageInput{ScrollY: w.ScrollY, MinimumScrollY: w.MinimumScrollY, MaximumScrollY: w.MaximumScrollY, RequestedStep: w.Player.ScrollStep, MiddleUpdated: w.thirdMiddleUpdated, FinalUpdated: w.thirdFinalUpdated, FinalDefeated: w.ThirdFinal.Defeated})
	w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = event.MinimumScrollY, event.MaximumScrollY, event.MaximumScrollY
	if event.HoldRequestedStep {
		step := event.ExtraBackgroundStep
		offset := (step >> 1) + (int(w.Frame&1) & step)
		w.BackgroundY = (w.BackgroundY - offset + 192) % 192
		if w.BackgroundStars != nil {
			w.BackgroundStars.Advance(step)
		}
		w.Player.ScrollStep = 0
	}
	if event.LaunchFinal {
		if err := w.spawnThirdFinal(); err != nil {
			return err
		}
	}
	if event.FinalCash {
		w.spawnSecondCashPairs(10, true)
	}
	return nil
}

func (w *World) spawnThirdFinal() error {
	launch, err := w.ThirdFinal.SelectLaunch()
	if err != nil {
		return err
	}
	path := w.paths[w.thirdFinalArt.Launches[launch].PathID]
	if path == nil {
		return fmt.Errorf("third final launch path is absent")
	}
	for i := range w.thirdFinalArt.Components {
		descriptor := &w.thirdFinalArt.Components[i]
		binding, err := w.reserveWorldActor(80, ActorPoolMoving, false)
		if err != nil {
			return err
		}
		state, err := NewThirdFinalMember(path, *descriptor, w.ScrollY, w.Player.ScrollStep)
		if err != nil {
			return err
		}
		state.Motion.X |= int32(binding.Residue.XFraction)
		state.Motion.Y |= int32(binding.Residue.YFraction)
		state.Motion.AngleFixed |= int32(uint16(binding.Residue.Direction))
		state.ResidualFireRate = binding.Residue.FireRate()
		actor := &WorldActor{ID: binding.EntityID, Binding: binding, X: float64(state.Motion.X >> 16), Y: float64(state.Motion.Y >> 16), ActorList: "moving", Atlas: "guardian-parts", Active: true, Score: 2000, Health: int(binding.Residue.Health), Sprite: state.Sprite, thirdFinalMember: &state, thirdPart: descriptor, path: path, motion: state.Motion,
			part: &visualassets.ActorPart{ResourceTag: 80, MotionMode: "third-final", DamageMode: "block-shot"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		actor.Binding.Residue.SetFireState(0, state.ResidualFireRate)
		w.poolActors[binding.Slot] = actor
		w.storeActorResidue(actor)
		w.Actors = append([]*WorldActor{actor}, w.Actors...)
	}
	return nil
}

func (w *World) advanceThirdFinal(actor *WorldActor) error {
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	event, err := actor.thirdFinalMember.Advance(actor.path, *actor.thirdPart, &w.Level.Paths.SineTable, ThirdFinalInput{ScrollDelta: w.ScrollDelta, PlayerX: w.Player.X, PlayerY: w.Player.Y, FireRate: uint8(w.thirdFinalArt.MotionParameters["fire_rate"]), ShotSpeed: w.thirdFinalArt.MotionParameters["shot_speed"]}, &w.random)
	if err != nil {
		return err
	}
	if actor.thirdPart.Index == 10 && event.Updated {
		w.thirdFinalUpdated = true
	}
	state := actor.thirdFinalMember
	actor.Active = !state.Removed
	actor.Visible = actor.Active
	actor.X, actor.Y = float64(state.Motion.X>>16), float64(state.Motion.Y>>16)
	actor.Sprite = state.Sprite
	actor.motion = state.Motion
	actor.Flash = false
	actor.Collision = CollisionRect{Right: -1, Bottom: -1}
	if actor.Active {
		w.updateSecondActorCollision(actor)
	}
	actor.Binding.Residue.SetFireState(state.FireAccumulator, state.ResidualFireRate)
	w.storeActorResidue(actor)
	for _, shot := range event.Shots[:event.ShotCount] {
		w.spawnThirdShot(shot, w.thirdFinalArt)
	}
	return nil
}
func (w *World) spawnThirdShot(event ThirdGuardianShot, group *visualassets.GuardianGroup) {
	w.spawnEnemyShot(event.X, event.Y, EnemyShot{Speed: event.Speed, Direction: event.Direction})
	if event.Animation == "" || len(w.Projectiles) == 0 {
		return
	}
	for _, clip := range group.Animations {
		if clip.ID == event.Animation {
			shot := w.Projectiles[0]
			shot.Atlas, shot.animation = "guardian-parts", clip.Animation
			shot.animationState = NewAnimation(clip.Animation)
			shot.Sprite = shot.animationState.Sprite(clip.Animation)
			break
		}
	}
}
func (w *World) damageThirdFinal(actor *WorldActor, amount uint16) {
	if actor.thirdPart.Index != 0 {
		return
	}
	for _, member := range w.Actors {
		if member.Active && member.thirdFinalMember != nil {
			member.Flash = true
		}
	}
	if !w.ThirdFinal.Strike(amount) {
		return
	}
	for _, member := range w.Actors {
		if member.Active && member.ActorList == "moving" {
			member.Active = false
			w.storeActorResidue(member)
		}
	}
	w.spawnSecondRandomExplosions(20, 0, 0, 320, 192)
}

func (w *World) restoreThirdMiddleActors() {
	if w.ThirdMiddle == nil || w.ThirdMiddle.Defeated {
		return
	}
	for _, actor := range w.thirdMiddleActors {
		if actor != nil && actor.Active {
			actor.PreviousX, actor.PreviousY = actor.X, actor.Y
			w.Actors = append(w.Actors, actor)
		}
	}
}
