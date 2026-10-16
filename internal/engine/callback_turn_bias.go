package engine

import "xenon2/internal/visualassets"

// callbackTurnBias is the direction retained by consecutive updater callbacks
// for the second guardian's crowded branch. It carries no native machine state.
type callbackTurnBias uint8

const (
	preserveCallbackTurn callbackTurnBias = iota
	forwardCallbackTurn
	reverseCallbackTurn
)

func callbackTurnFromFraction(fraction uint16) callbackTurnBias {
	if fraction&0x4000 != 0 {
		return reverseCallbackTurn
	}
	return forwardCallbackTurn
}

func callbackTurnFromWhole(value int) callbackTurnBias {
	return callbackTurnFromFraction(uint16(value))
}

func (bias callbackTurnBias) apply(previous bool) bool {
	switch bias {
	case reverseCallbackTurn:
		return true
	case forwardCallbackTurn:
		return false
	default:
		return previous
	}
}

func animationCallbackTurn(state AnimationState, clip visualassets.ActorAnimation) callbackTurnBias {
	if state.Remaining != 1 || len(clip.Frames) == 0 {
		return preserveCallbackTurn
	}
	next := state.Frame + 1
	if next >= len(clip.Frames) {
		return forwardCallbackTurn
	}
	if clip.Frames[next].ReverseWhenAdvanced {
		return reverseCallbackTurn
	}
	return forwardCallbackTurn
}
