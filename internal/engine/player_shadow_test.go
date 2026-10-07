package engine

import "testing"

func TestPlayerShadowNativeTraceOptional(t *testing.T) {
	cases := 0
	nativeCombatRows(t, "player-shadow-trace.csv", func(v []int64) {
		state := PlayerShadowState{Counter: int(v[0])}
		player := PlayerMotionState{X: int(v[4]), Y: int(v[5]), Inertia: int(v[1])}
		if err := StepPlayerShadow(&state, player, MotionInput{Up: v[2]&1 != 0, Down: v[2]&2 != 0}, v[3] != 0); err != nil {
			t.Fatal(err)
		}
		if state.X != int(v[6]) || state.Y != int(v[7]) || state.Visible != (v[8] != 0) {
			t.Fatalf("shadow %v: got %+v", v, state)
		}
		cases++
	})
	if cases != 416 {
		t.Fatalf("shadow coverage: %d cases", cases)
	}
}

func TestPlayerShadowDoesNotFollowTrail(t *testing.T) {
	state := PlayerShadowState{Counter: 3}
	if err := StepPlayerShadow(&state, PlayerMotionState{X: 200, Y: 90, Inertia: -6}, MotionInput{Down: true}, false); err != nil {
		t.Fatal(err)
	}
	if state.X != 202 || state.Y != 90 || !state.Visible {
		t.Fatalf("current-position thrust silhouette: %+v", state)
	}
}
