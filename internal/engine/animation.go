package engine

import "xenon2/internal/visualassets"

// AnimationState retains the original frame countdown. A zero-duration frame
// remains displayed; positive durations count gameplay passes, not draws.
type AnimationState struct {
	Frame     int
	Remaining int
}

func NewAnimation(animation visualassets.ActorAnimation) AnimationState {
	state := AnimationState{}
	if len(animation.Frames) != 0 {
		state.Remaining = animation.Frames[0].Duration
	}
	return state
}

// Advance applies the decrement-before-switch rule of the original animator.
func (s *AnimationState) Advance(animation visualassets.ActorAnimation) {
	if len(animation.Frames) == 0 || s.Remaining <= 0 {
		return
	}
	s.Remaining--
	if s.Remaining != 0 {
		return
	}
	s.Frame++
	if s.Frame == len(animation.Frames) {
		s.Frame = animation.LoopFrom
	}
	if s.Frame < 0 || s.Frame >= len(animation.Frames) {
		return
	}
	s.Remaining = animation.Frames[s.Frame].Duration
}

func (s AnimationState) Sprite(animation visualassets.ActorAnimation) string {
	if s.Frame < 0 || s.Frame >= len(animation.Frames) {
		return ""
	}
	return animation.Frames[s.Frame].Sprite
}

// AdvanceNamed preserves a finite sequence's removal ending instead of
// silently turning it into a loop. Held and looping frames remain active.
func (s *AnimationState) AdvanceNamed(animation visualassets.NamedActorAnimation) bool {
	if animation.Ending == "remove" && s.Frame == len(animation.Animation.Frames)-1 && s.Remaining == 1 {
		s.Remaining = 0
		return false
	}
	s.Advance(animation.Animation)
	return true
}
