package engine

import "xenon2/internal/visualassets"

func (w *World) advanceFourthStage() error {
	if w.Level.Number != 4 || w.Level.FixedSprites == nil || w.Level.FixedSprites.FourthStage == nil {
		return nil
	}
	art := w.Level.FixedSprites.FourthStage
	decision := FourthStagePrelude(w.Frame, w.ScrollY, w.MaximumScrollY)
	w.MaximumScrollY, w.VisitedScrollY = decision.MaximumScrollY, decision.MaximumScrollY
	if !decision.Spawn {
		return nil
	}
	binding, err := w.reserveWorldActor(200, ActorPoolMoving, false)
	if err != nil {
		return err
	}
	variant := decision.FamilyOffset + int(w.random.Next()&1)
	w.Pool.Slot(binding.Slot).ResourceTag = int16(art.Variants[variant].ResourceTag)
	state := NewFourthFallingActor(variant, art, binding.Residue, w.random.Next())
	actor := &WorldActor{ID: binding.EntityID, Binding: binding, Active: true, Visible: true, ActorList: "moving", Atlas: "fixed", X: float64(state.X), Y: float64(state.Y), PreviousX: float64(state.X), PreviousY: float64(state.Y), Health: art.Health, Score: 300, fourthFalling: &state, part: &visualassets.ActorPart{ResourceTag: art.Variants[variant].ResourceTag, StrongHealth: true, DamageMode: "individual", MotionMode: "fourth-falling"}}
	actor.Sprite = state.Sprite(art)
	w.updateSecondActorCollision(actor)
	w.poolActors[binding.Slot] = actor
	w.Actors = append([]*WorldActor{actor}, w.Actors...)
	w.storeFourthStageResidue(actor)
	return nil
}

func (w *World) advanceFourthFalling(actor *WorldActor) {
	art, state := w.Level.FixedSprites.FourthStage, actor.fourthFalling
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	event := state.Advance(art, w.ScrollDelta, func(name string) visualassets.CollisionBox { return w.movingSpriteBoxes[name] }, w.random.Next)
	actor.X, actor.Y, actor.Sprite, actor.Collision, actor.Active = float64(state.X), float64(state.Y), state.Sprite(art), state.Collision, !state.Removed
	if event.SpawnPod {
		w.spawnFourthPod(event.PodSide, event.PodX, event.PodY)
	}
	if event.Shot {
		w.spawnEnemyShot(event.ShotX, event.ShotY, EnemyShot{Direction: event.ShotDirection, Speed: event.ShotSpeed})
		if len(w.Projectiles) != 0 {
			w.Projectiles[0].Atlas, w.Projectiles[0].Sprite = "fixed", art.ShotSprite
		}
	}
	w.storeFourthStageResidue(actor)
}

func (w *World) spawnFourthPod(side, x, y int) {
	art := w.Level.FixedSprites.FourthStage
	binding, err := w.reserveWorldActor(int16(art.Variants[side*2].PodTag), ActorPoolProjectile, false)
	if err != nil {
		w.poolError = err
		return
	}
	state := NewFourthPod(side, x, y, art)
	actor := &WorldActor{ID: binding.EntityID, Binding: binding, Active: true, Visible: true, ActorList: "transient", Atlas: "fixed", X: float64(x), Y: float64(y), PreviousX: float64(x), PreviousY: float64(y), fourthPod: &state, part: &visualassets.ActorPart{ResourceTag: art.Variants[side*2].PodTag, DamageMode: "block-shot", MotionMode: "fourth-pod"}}
	actor.Binding.Residue.Counter = 0
	actor.Sprite = state.Animation.Sprite(art.Variants[side*2].PodAnimation)
	w.poolActors[binding.Slot] = actor
	w.Actors = append([]*WorldActor{actor}, w.Actors...)
	w.storeFourthStageResidue(actor)
}

func (w *World) advanceFourthPod(actor *WorldActor) error {
	art, state := w.Level.FixedSprites.FourthStage, actor.fourthPod
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	event := state.Advance(art, w.ScrollDelta, func(name string) visualassets.SpriteRegion {
		for _, region := range w.Level.FixedSprites.Atlas.Sprites {
			if region.Name == name {
				return region
			}
		}
		return visualassets.SpriteRegion{}
	})
	actor.X, actor.Y, actor.Sprite = float64(state.X), float64(state.Y), state.Animation.Sprite(art.Variants[state.Side*2].PodAnimation)
	if event.SpawnChild {
		if err := w.spawnFourthPodChild(state.Side, event.ChildX, event.ChildY); err != nil {
			return err
		}
	}
	if event.ConvertToExplosion {
		actor.X, actor.Y = float64(event.ExplosionX), float64(event.ExplosionY)
		actor.fourthPod = nil
		actor.part.ResourceTag, actor.part.MotionMode = 12, "finite-effect"
		clip := w.commonAnimations["explosion-small"]
		clip.Animation.Ending = clip.Ending
		actor.Atlas, actor.animation, actor.animationState = "common", clip.Animation, NewAnimation(clip.Animation)
		actor.Sprite = actor.animationState.Sprite(actor.animation)
		w.Pool.Slot(actor.Binding.Slot).ResourceTag = 12
	}
	w.storeFourthStageResidue(actor)
	return nil
}

func (w *World) spawnFourthPodChild(side, x, y int) error {
	art := w.Level.FixedSprites.FourthStage
	binding, err := w.reserveWorldActor(248, ActorPoolMoving, false)
	if err != nil {
		return err
	}
	state, err := NewFourthPodChild(side, x, y, art, binding.Residue)
	if err != nil {
		return err
	}
	actor := &WorldActor{ID: binding.EntityID, Binding: binding, Active: true, Visible: true, ActorList: "moving", Atlas: "fixed", X: float64(x), Y: float64(y), PreviousX: float64(x), PreviousY: float64(y), Health: art.PodChildHealth, Score: 100, Collision: state.Collision, fourthChild: &state, part: &visualassets.ActorPart{ResourceTag: 248, DamageMode: "individual", MotionMode: "fourth-pod-child"}}
	actor.Sprite = state.Sprite(art)
	w.poolActors[binding.Slot] = actor
	w.Actors = append([]*WorldActor{actor}, w.Actors...)
	actor.WaveToken = w.WaveBonuses.RegisterIndependent()
	w.storeFourthStageResidue(actor)
	return nil
}

func (w *World) advanceFourthChild(actor *WorldActor) error {
	state, art := actor.fourthChild, w.Level.FixedSprites.FourthStage
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	if err := state.Advance(art, &w.Level.Paths.SineTable, w.random.Next, func(name string) visualassets.CollisionBox { return w.movingSpriteBoxes[name] }); err != nil {
		return err
	}
	if state.Rerouted {
		w.WaveBonuses.Escape(actor.WaveToken)
	}
	actor.X, actor.Y, actor.Sprite, actor.Collision = float64(state.Motion.X>>16), float64(state.Motion.Y>>16), state.Sprite(art), state.Collision
	w.storeFourthStageResidue(actor)
	return nil
}

func (w *World) storeFourthStageResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	r := &actor.Binding.Residue
	r.X, r.Y = int16(actor.X), int16(actor.Y)
	if actor.fourthFalling != nil || actor.fourthChild != nil {
		r.Health, r.PowerOrScore = uint16(actor.Health), uint16(actor.Score)
		r.WaveBonusToken = actor.WaveToken
	}
	if s := actor.fourthFalling; s != nil {
		r.Counter = int16(s.Phase)
		r.SetFireState(s.PrimaryClock, s.SecondaryClock)
	}
	if s := actor.fourthChild; s != nil {
		r.XFraction, r.YFraction = uint16(s.Motion.X), uint16(s.Motion.Y)
		r.Counter, r.MotionBudget = int16(s.Motion.Remaining), int16(s.Motion.Budget)
		r.Direction, r.HorizontalDriftRemainder = int16(uint16(s.Motion.AngleFixed)), uint16(uint32(s.Motion.AngleFixed)>>16)
	}
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}
