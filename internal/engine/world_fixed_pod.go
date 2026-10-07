package engine

import "xenon2/internal/visualassets"

func (w *World) spawnFixedPod(record visualassets.FixedEncounter) bool {
	if w.Level.Number != 2 || w.Level.FixedTiles == nil {
		return false
	}
	for i := range w.Level.FixedTiles.Kinds {
		kind := &w.Level.FixedTiles.Kinds[i]
		if kind.Kind != record.EnemyKind || kind.Behavior != "second-pod" {
			continue
		}
		variant := &kind.Variants[0]
		state := FixedPodState{X: record.X + variant.OriginOffsetX, WorldY: record.Y + variant.OriginOffsetY, Repeats: 1, Large: kind.Kind == 5}
		actor := &WorldActor{Active: true, ActorList: "scenery", fixedPod: &state, fixedTileVariant: variant, part: &visualassets.ActorPart{ResourceTag: variant.ResourceTag, DamageMode: "block-shot"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
		if err := w.bindWorldActor(actor); err != nil {
			w.poolError = err
			return true
		}
		w.initializeSecondEmitterResidue(actor)
		if !state.Large {
			w.setSecondMapPatch(state.X/16, state.WorldY/16, variant.Initial)
		}
		w.Actors = append([]*WorldActor{actor}, w.Actors...)
		return true
	}
	return false
}

func (w *World) advanceFixedPod(actor *WorldActor) {
	state := actor.fixedPod
	event := state.Advance(w.Frame, w.ScrollY, w.MaximumScrollY)
	actor.Active, actor.Visible = !state.Removed, false
	if event.WriteTiles {
		w.setSecondMapPatch(state.X/16, state.WorldY/16, actor.fixedTileVariant.Frames[event.Frame])
	}
	if event.Spawn {
		variant := 1
		if state.Large {
			variant = 0
		}
		w.spawnPodCreature(event.SpawnX, event.SpawnY, variant)
	}
}

func (w *World) spawnPodCreature(x, y, variant int) {
	if w.Level.FixedSprites == nil || w.Level.FixedSprites.PodCreatures == nil {
		return
	}
	art := w.Level.FixedSprites.PodCreatures
	clip := art.Idle[variant]
	state := PodCreatureState{X: x, Y: y, Variant: variant, Clip: clip, Animation: NewAnimation(clip)}
	actor := &WorldActor{Active: true, Visible: true, ActorList: "moving", Atlas: "fixed", podCreature: &state, Health: art.Health[variant], Score: art.Score[variant],
		Sprite: state.Animation.Sprite(clip), X: float64(x), Y: float64(y), PreviousX: float64(x), PreviousY: float64(y), part: &visualassets.ActorPart{ResourceTag: 224 + variant*4, StrongHealth: variant == 0, DamageMode: "individual"}}
	if err := w.bindWorldActor(actor); err != nil {
		w.poolError = err
		return
	}
	state.AllocationPhase = int(actor.Binding.AllocationPhase)
	w.initializeSecondCreatureResidue(actor)
	w.updateSecondActorCollision(actor)
	w.Actors = append([]*WorldActor{actor}, w.Actors...)
}

func (w *World) advancePodCreature(actor *WorldActor) {
	actor.Flash = false
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	state := actor.podCreature
	state.Advance(w.Frame, w.Player.X, w.Level.FixedSprites.PodCreatures)
	actor.X, actor.Y = float64(state.X), float64(state.Y)
	actor.Sprite = state.Animation.Sprite(state.Clip)
	actor.Visible = true
	w.updateSecondActorCollision(actor)
}
