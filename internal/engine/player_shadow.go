package engine

import "fmt"

// PlayerShadowState is one of the four source thrust silhouettes. Its counter
// selects a fixed image and a side; no trail or animation countdown is used.
type PlayerShadowState struct {
	Counter int
	X, Y    int
	Visible bool
}

var shadowOffsets = [2][7]int{{0, -1, -1, -1, -2, -2, -2}, {0, 1, 1, 1, 2, 2, 2}}

func StepPlayerShadow(state *PlayerShadowState, player PlayerMotionState, input MotionInput, diving bool) error {
	inertia := player.Inertia
	if inertia < 0 {
		inertia = -inertia
	}
	if state.Counter < 0 || state.Counter > 3 || inertia > 6 {
		return fmt.Errorf("invalid player shadow state")
	}
	state.X, state.Y = player.X+shadowOffsets[state.Counter&1][inertia], player.Y
	state.Visible = !diving && (state.Counter < 2 && input.Up || state.Counter >= 2 && input.Down)
	return nil
}
