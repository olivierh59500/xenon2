package engine

// demoThirdChainPrediction copies one articulated chain and its random stream.
// Each row contains its eight source members after that actor phase. Inputs are
// candidate-dependent: playerYs is the ship height after that pass's movement,
// and scrollDeltas is the preceding pass's actual camera displacement.
//
// The copy includes hidden body colliders and the tail's held/released poses.
// It predicts no combat removal, births or unrelated random callbacks. Active
// sweeps need no random value; an idle activation uses the supplied stream. A
// caller forecasting several groups must account for their source RNG order.
func demoThirdChainPrediction(w *World, actor *WorldActor, horizon int, scrollDeltas, playerYs [8]int, random RandomState, dst [][8]demoActorView) (nextRandom RandomState, ok bool) {
	if w == nil || w.Level.Number != 3 || w.Level.FixedSprites == nil || w.Level.FixedSprites.Third == nil || actor == nil || horizon < 1 || horizon > 8 || len(dst) < horizon {
		return random, false
	}
	leader := actor
	if leader.thirdChainPart != 1 {
		leader = actor.leader
	}
	if leader == nil || !leader.Active || leader.thirdChain == nil || leader.thirdChainPart != 1 {
		return random, false
	}
	state := *leader.thirdChain
	art := w.Level.FixedSprites.Third.Chain
	if state.Variant < 0 || state.Variant >= len(art.Bodies) || state.Variant >= len(art.Tails) {
		return random, false
	}
	var views [8]demoActorView
	for index, member := range leader.thirdChainMembers {
		if member == nil {
			return random, false
		}
		views[index] = demoActorView{X: int(member.X), Y: int(member.Y), Sprite: member.Sprite, Bounds: member.Collision, Active: member.Active, Visible: member.Visible}
	}
	for future := 0; future < horizon; future++ {
		if views[0].Active {
			state.Advance(ThirdChainInput{ScrollDelta: scrollDeltas[future], PlayerY: playerYs[future]}, art, &random)
			for index, part := range state.Parts {
				if !views[index].Active {
					continue
				}
				sprite := art.Bodies[state.Variant]
				if index == 7 {
					sprite = part.Animation.Sprite(art.Tails[state.Variant])
				}
				box, exists := w.movingSpriteBoxes[sprite]
				if !exists {
					return random, false
				}
				views[index] = demoActorView{X: part.X, Y: part.Y, Sprite: sprite, Bounds: ActorCollisionRect(box, part.X, part.Y), Active: !part.Removed, Visible: part.Visible}
			}
		}
		dst[future] = views
	}
	return random, true
}
