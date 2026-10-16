package engine

// fourthMiddleCoastMotion compares canonical one-command-and-release endpoints.
// Legal left-flank stopping positions precede distance; idle wins equal scores.
// Other targets retain unrestricted native quantization.
func fourthMiddleCoastMotion(player PlayerMotionState, target int) (MotionInput, bool) {
	context := MotionContext{BaseScrollStep: 1}
	motions := [3]MotionInput{{}, {Left: true}, {Right: true}}
	best, bestDistance := motions[0], int(^uint(0)>>1)
	bestSet, bestLegal := false, false
	for _, motion := range motions {
		next := player
		next.Advance(motion, context)
		for next.Inertia != 0 {
			next.Advance(MotionInput{}, context)
		}
		distance := absDemo(next.X - target)
		legal := target != 80 || next.X <= 80
		if !bestSet || legal && !bestLegal || legal == bestLegal && distance < bestDistance {
			best, bestDistance, bestLegal, bestSet = motion, distance, legal, true
		}
	}
	return best, player.Inertia == 0 && best == (MotionInput{})
}

// The complete heading-frame intersection gives a point lane valid for every
// satellite heading. Every Rear Shot tier has zero horizontal spawn/movement offset.
func fourthMiddleStableAim(w *World, index int) (int, int, int, bool) {
	if w == nil || w.fourthMiddleArt == nil || index < 0 || index >= len(w.fourthMiddleArt.Components) {
		return 0, 0, 0, false
	}
	component := w.fourthMiddleArt.Components[index]
	if len(component.HeadingAnimations) != 8 {
		return 0, 0, 0, false
	}
	left, right := -10000, 10000
	for _, animation := range component.HeadingAnimations {
		if len(animation.Frames) == 0 {
			return 0, 0, 0, false
		}
		for _, frame := range animation.Frames {
			box, exists := w.movingSpriteBoxes[frame.Sprite]
			if !exists || box.Width <= 0 || box.Height <= 0 {
				return 0, 0, 0, false
			}
			bounds := ActorCollisionRect(box, component.InitialX, 0)
			left, right = max(left, bounds.Left), min(right, bounds.Right)
		}
	}
	if left > right {
		return 0, 0, 0, false
	}
	return (left + right) / 2, left, right, true
}
