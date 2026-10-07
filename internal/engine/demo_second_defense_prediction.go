package engine

import "xenon2/internal/visualassets"

type demoSecondDefenseView struct {
	ID, PartIndex, Stream int
	Materializing         bool
	demoActorView
}

type demoSecondDefenseMember struct {
	state SecondDefenseSegment
	path  *visualassets.Path
	part  *visualassets.GuardianComponent
	view  demoSecondDefenseView
}

// demoSecondDefenseForecast predicts the currently active defense members for
// one through eight actor phases. Each output row contains count entries in
// physical moving-list order, starting at dst[future*count]. Retired members
// remain in their original row position with Active false.
//
// scrollDeltas supplies the preceding pass's actual camera displacement, as
// SecondDefenseSegment.Advance uses it before its path substeps. These twelve
// staggered members follow independent paths, including hidden colliders;
// they do not copy a preceding member's coordinates.
//
// This copies controllers, gates and their shared random stream. It forecasts
// no scheduler births, allocator evictions, combat deaths or unrelated random
// callbacks. All sixteen original defense paths contain only curves and End,
// so their original geometry is independent of unrelated random consumption.
func demoSecondDefenseForecast(w *World, horizon int, scrollDeltas [8]int, dst []demoSecondDefenseView) (count int, ok bool) {
	if w == nil || w.Level.Number != 2 || w.Level.Paths == nil || horizon < 1 || horizon > 8 {
		return 0, false
	}
	var ordered [ActorPoolCapacity]*WorldActor
	var members [ActorPoolCapacity]demoSecondDefenseMember
	for _, actor := range w.orderedMovingActors(&ordered) {
		if !actor.Active || actor.secondSegment == nil {
			continue
		}
		if actor.secondPart == nil || len(actor.secondPart.HeadingFrames) != 8 || ValidatePath(actor.path) != nil || actor.path.ID != actor.secondSegment.Motion.PathID {
			return 0, false
		}
		for _, sprite := range actor.secondPart.HeadingFrames {
			if _, exists := w.movingSpriteBoxes[sprite]; !exists {
				return 0, false
			}
		}
		members[count] = demoSecondDefenseMember{state: *actor.secondSegment, path: actor.path, part: actor.secondPart,
			view: demoSecondDefenseView{ID: actor.ID, PartIndex: actor.secondPart.Index, Stream: actor.secondSegment.Stream, Materializing: actor.Materializing,
				demoActorView: demoActorView{X: int(actor.X), Y: int(actor.Y), Sprite: actor.Sprite, Bounds: actor.Collision, Active: actor.Active, Visible: actor.Visible}}}
		count++
	}
	if len(dst) < count*horizon {
		return 0, false
	}
	gates, random := w.secondGateCounters, w.RandomState()
	for future := 0; future < horizon; future++ {
		for index := 0; index < count; index++ {
			member := &members[index]
			if member.view.Active {
				event, err := member.state.Advance(member.path, &w.Level.Paths.SineTable, scrollDeltas[future], &gates, &random)
				if err != nil {
					return 0, false
				}
				member.view.X, member.view.Y = int(member.state.Motion.X>>16), int(member.state.Motion.Y>>16)
				member.view.Sprite = member.part.HeadingFrames[event.HeadingFrame]
				member.view.Active = !member.state.Removed
				member.view.Visible = member.view.Active && member.state.Presentation != "hidden"
				member.view.Materializing = member.state.Presentation == "materializing"
				member.view.Bounds = CollisionRect{Right: -1, Bottom: -1}
				if member.view.Active {
					member.view.Bounds = ActorCollisionRect(w.movingSpriteBoxes[member.view.Sprite], member.view.X, member.view.Y)
				}
			}
			dst[future*count+index] = member.view
		}
	}
	return count, true
}
