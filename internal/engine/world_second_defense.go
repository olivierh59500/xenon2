package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

func (w *World) initializeSecondArena() error {
	if w.Level.Number != 2 || w.SecondGuardian == nil || len(w.Level.GuardianGroups) == 0 {
		return nil
	}
	for i := range w.Level.GuardianGroups {
		group := &w.Level.GuardianGroups[i]
		switch group.ID {
		case "middle-defense-wave":
			w.secondWaveArt = group
		case "middle-defense-nodes":
			w.secondNodeArt = group
		case "final-terrain-guardian":
			w.secondTerrainCells = NewSecondTerrainCells(group.DestructibleCells)
		}
	}
	if w.secondWaveArt == nil || w.secondNodeArt == nil || len(w.secondWaveArt.Components) != 12 || len(w.secondNodeArt.Components) != 3 || len(w.secondWaveArt.Launches) != 16 || w.Level.GuardianParts == nil {
		return fmt.Errorf("second arena resources are incomplete")
	}
	for _, bank := range []*visualassets.SpriteAtlas{w.Level.GuardianParts, &w.Level.Guardians.Atlas} {
		for _, sprite := range bank.Sprites {
			if sprite.Collision != nil {
				w.movingSpriteBoxes[sprite.Name] = *sprite.Collision
			}
		}
	}
	config, err := SecondMinionConfigFromVisual(*w.secondGuardianArt)
	if err != nil {
		return err
	}
	w.secondMinionConfig = &config
	scheduler := NewSecondDefenseScheduler()
	w.secondScheduler = &scheduler
	w.secondDefenseRemaining = 3
	var nodes []*WorldActor
	for i := range w.secondNodeArt.Components {
		descriptor := &w.secondNodeArt.Components[i]
		state := NewSecondDefenseNodeState(descriptor.Index, descriptor.InitialX/16, descriptor.InitialWorldY/16, descriptor.Health)
		w.nextActorID++
		actor := &WorldActor{ID: w.nextActorID, Active: true, ActorList: "moving", Atlas: "guardian-parts", Health: descriptor.Health, part: &visualassets.ActorPart{ResourceTag: 84, DamageMode: "second-defense-node"}, secondNode: &state, secondPart: descriptor, Collision: state.Collision}
		actor.X, actor.Y = float64(descriptor.InitialX), float64(descriptor.InitialWorldY-w.ScrollY)
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		w.secondNodes[descriptor.Index] = actor
		nodes = append(nodes, actor)
	}
	w.Actors = append(nodes, w.Actors...)
	return nil
}

func (w *World) setSecondMapCell(column, row int, tile uint16) {
	if column < 0 || column >= w.Level.Terrain.Columns || row < 0 || row >= w.Level.Terrain.Rows {
		return
	}
	w.Level.Terrain.Map[row*w.Level.Terrain.Columns+column] = tile
}
func (w *World) setSecondMapPatch(column, row int, patch visualassets.TilePatch) {
	for y := range patch.Rows {
		for x := range patch.Columns {
			w.setSecondMapCell(column+x, row+y, patch.Tiles[y*patch.Columns+x])
		}
	}
}
func (w *World) advanceSecondNode(actor *WorldActor) {
	state := actor.secondNode
	event := state.Advance(SecondDefenseNodeInput{Frame: w.Frame, PlayerX: w.Player.X, ScrollY: w.ScrollY, MaximumScrollY: w.MaximumScrollY, Remaining: w.secondDefenseRemaining, DefenseFlags: w.secondScheduler.DefenseFlags, BackwardScroll: w.secondBackward}, &w.secondGateCounters)
	w.MaximumScrollY, w.VisitedScrollY = event.MaximumScrollY, event.MaximumScrollY
	actor.X, actor.Y = float64(state.TileX*16), float64(state.TileY*16-w.ScrollY)
	actor.Collision = state.Collision
	actor.Patch = nil
	actor.Visible = false
	actor.Flash = false
	w.setSecondMapPatch(state.TileX, state.TileY, actor.secondPart.TileFrames[event.TileFrame])
	for _, gate := range w.secondNodeArt.Gates {
		index := gate.ID - 1
		if index >= 0 && index < 8 && event.GateChanged[index] {
			w.setSecondMapPatch(gate.Column, gate.Row, gate.Frames[event.GateFrame[index]])
		}
	}
}
func (w *World) advanceSecondDefenseWaves() error {
	if w.secondScheduler == nil {
		return nil
	}
	if w.secondMiddleReleased {
		if w.ScrollY > 2544 {
			w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2544, 2544, 2544
		}
		return nil
	}
	if w.ScrollY < 2512 || w.ScrollY > 2896 {
		return nil
	}
	for _, choice := range w.secondScheduler.LaunchIdleStreams(w.secondStreamsUpdated, w.secondDefenseRemaining) {
		launch := &w.secondWaveArt.Launches[choice.LaunchIndex]
		if launch.GateID > 0 && launch.GateID <= 8 && w.secondGateCounters[launch.GateID-1] == 0 {
			w.secondGateCounters[launch.GateID-1] = 24
		}
		for i := range w.secondWaveArt.Components {
			descriptor := &w.secondWaveArt.Components[i]
			segment, err := NewSecondDefenseSegment(*launch, *descriptor, choice.Stream, w.ScrollY, w.Player.ScrollStep)
			if err != nil {
				return err
			}
			w.nextActorID++
			actor := &WorldActor{ID: w.nextActorID, Active: true, ActorList: "moving", Atlas: "guardian-parts", Health: descriptor.Health, Score: descriptor.Score, Sprite: descriptor.Sprite, secondSegment: &segment, secondPart: descriptor, path: &launch.Path,
				part: &visualassets.ActorPart{ResourceTag: descriptor.ResourceTag, DamageMode: "second-defense-segment"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
			actor.X, actor.Y = float64(segment.Motion.X>>16), float64(segment.Motion.Y>>16)
			actor.PreviousX, actor.PreviousY = actor.X, actor.Y
			w.Actors = append([]*WorldActor{actor}, w.Actors...)
		}
	}
	return nil
}
func (w *World) advanceSecondSegment(actor *WorldActor) error {
	event, err := actor.secondSegment.Advance(actor.path, &w.Level.Paths.SineTable, w.ScrollDelta, &w.secondGateCounters, &w.random)
	if err != nil {
		return err
	}
	if event.StreamUpdated {
		w.secondStreamsUpdated[actor.secondSegment.Stream] = true
	}
	actor.Active = !actor.secondSegment.Removed
	actor.Visible = actor.Active && actor.secondSegment.Presentation != "hidden"
	actor.Materializing = actor.secondSegment.Presentation == "materializing"
	actor.X, actor.Y = float64(actor.secondSegment.Motion.X>>16), float64(actor.secondSegment.Motion.Y>>16)
	actor.Sprite = actor.secondPart.HeadingFrames[event.HeadingFrame]
	actor.Flash = false
	actor.Collision = CollisionRect{Right: -1, Bottom: -1}
	if actor.Active {
		w.updateSecondActorCollision(actor)
	}
	return nil
}
func (w *World) damageSecondSegment(actor *WorldActor, amount uint16) {
	result := DamageSecondDefenseSegment(uint16(actor.Health), amount, actor.secondPart.Index == 0, actor.secondSegment.Stream, w.secondScheduler.DefenseFlags, &w.random)
	actor.Health, actor.Flash = int(result.Health), true
	w.secondScheduler.DefenseFlags = result.DefenseFlags
	if !result.Destroyed {
		return
	}
	actor.Active = false
	w.Score += actor.Score
	state := NewSecondDefenseFragment(int(actor.X), int(actor.Y), result.FragmentHeading, actor.secondPart.DeathAnimation)
	w.nextActorID++
	fragment := &WorldActor{ID: w.nextActorID, X: actor.X, Y: actor.Y, PreviousX: actor.X, PreviousY: actor.Y, Atlas: "guardian-parts", ActorList: "transient", Active: true, Visible: true,
		animation: actor.secondPart.DeathAnimation, animationState: state.Animation, secondFragment: &state, part: &visualassets.ActorPart{ResourceTag: actor.part.ResourceTag, DamageMode: "block-shot"}, Collision: CollisionRect{Right: -1, Bottom: -1}}
	fragment.Sprite = state.Animation.Sprite(fragment.animation)
	w.Actors = append([]*WorldActor{fragment}, w.Actors...)
}
func (w *World) advanceSecondFragment(actor *WorldActor) {
	actor.secondFragment.Advance(actor.animation, &w.Level.Paths.SineTable)
	actor.Active = !actor.secondFragment.Removed
	actor.X, actor.Y = float64(actor.secondFragment.X), float64(actor.secondFragment.Y)
	actor.Sprite = actor.secondFragment.Animation.Sprite(actor.animation)
	actor.Visible = actor.Active
}
func (w *World) damageSecondNode(actor *WorldActor, amount uint16) {
	event := actor.secondNode.Strike(amount, w.secondDefenseRemaining, w.secondScheduler.DefenseFlags)
	if !event.Applied {
		return
	}
	actor.Health = int(actor.secondNode.Health)
	actor.Patch = actor.secondPart.DamageFlash
	actor.X += float64(actor.secondPart.FlashOffsetX)
	actor.Y += float64(actor.secondPart.FlashOffsetY)
	actor.Flash, actor.Visible = true, true
	if !event.Destroyed {
		return
	}
	actor.Active = false
	w.secondDefenseRemaining = event.Remaining
	w.secondScheduler.DefenseFlags = event.DefenseFlags
	w.setSecondMapPatch(actor.secondNode.TileX, actor.secondNode.TileY, actor.secondPart.TileFrames[5])
	w.spawnSecondExplosion(int(actor.Collision.Left)+8, int(actor.Collision.Top)+8)
	if event.ReverseScroll {
		w.secondBackward = true
		w.BaseScrollStep = -1
		w.secondScheduler.UseFinalLaunches()
	}
	if !event.ReleaseMinimum {
		return
	}
	w.secondMiddleReleased = true
	w.secondBackward = false
	w.BaseScrollStep = 1
	w.MinimumScrollY = 0
	for _, candidate := range w.Actors {
		if candidate.Active && candidate.ActorList == "moving" && !candidate.secondGuardian {
			candidate.Active = false
		}
	}
	for i := 162; i < 185; i++ {
		for x := range 20 {
			w.setSecondMapCell(x, i, 0)
		}
	}
	w.spawnSecondRandomExplosions(event.ExplosionCount, 0, 0, 320, 192)
	w.spawnSecondCashPairs(event.CashPairs, false)
}
func (w *World) restoreSecondArenaActors() {
	if w.secondScheduler == nil {
		return
	}
	w.secondScheduler.DefenseFlags = 0
	for i := len(w.secondNodes) - 1; i >= 0; i-- {
		actor := w.secondNodes[i]
		if actor != nil && actor.Active && !actor.secondNode.Destroyed {
			actor.Collision = CollisionRect{Right: -1, Bottom: -1}
			w.Actors = append([]*WorldActor{actor}, w.Actors...)
		}
	}
}
