package engine

import (
	"reflect"
	"testing"
)

func TestNativeMotionPreviewIncludesOriginalTurnBeyondSixCommandsOptional(t *testing.T) {
	w := nativeMotionFixture(t, 251, 176, 1732)
	var planner nativeMotionPlanner
	input, ok := planner.command(w, 254, 1908, 254, 1908)
	if !ok {
		t.Fatal("original narrow route was not prepared")
	}
	at := planner.at
	commands, ok := planner.guardCommands(w, input)
	if !ok || len(commands) != 7 || planner.at != at {
		t.Fatal("preview omitted the original final alignment or consumed its route")
	}
	for _, motion := range commands {
		if err := w.Step(Input{Motion: motion}); err != nil {
			t.Fatal(err)
		}
		if w.Rewind.Timer != 0 || !w.PlayerAlive {
			t.Fatal("complete commitment entered terrain or lost its ship")
		}
	}
	if !nativeMotionMatches(w, planner.states[len(planner.states)-1]) {
		t.Fatal("full preview diverged from its actual source-motion endpoint")
	}
}

func nativeGuardCannonFixture(t testing.TB) (*World, nativeMotionPlanner, Input) {
	t.Helper()
	w := nativeMotionFixture(t, 93, 166, 690)
	w.MaterializationFrames = 0
	w.Player.SpeedTier = w.Equipment.SpeedTier
	// This source encounter is already behind its native admission row in the
	// observed corridor pose. Its original constructor writes the actual tiles.
	found := false
	for _, record := range w.Level.Encounters.Fixed {
		if record.EnemyKind == 4 && record.X == 152 && record.Y == 872 {
			w.spawnFixed(record)
			found = true
			break
		}
	}
	if !found || len(w.Actors) == 0 || w.Actors[0].fixedTileState == nil {
		t.Fatal("original corridor tile gun was not admitted")
	}
	w.advanceFixedTile(w.Actors[0])
	var planner nativeMotionPlanner
	motion, ok := planner.command(w, 102, 860, 160, 776)
	if !ok || len(planner.commands) != 1 || motion != (MotionInput{Down: true, Right: true}) {
		t.Fatalf("original one-command corner differs: found%v commands%+v", ok, planner.commands)
	}
	return w, planner, Input{Motion: motion}
}

func TestNativeMotionGuardSequenceFinalCommandIsReadOnlyOptional(t *testing.T) {
	w, planner, planned := nativeGuardCannonFixture(t)
	before := forecastDigest(w)
	commands := append([]MotionInput(nil), planner.commands...)
	states := append([]demoMotionForecast(nil), planner.states...)
	at := planner.at
	sequence, ok := planner.guardSequence(w, planned.Motion)
	if !ok || planner.at != len(planner.commands) || sequence[0] != planned.Motion {
		t.Fatal("final command was not previewed from its before-state")
	}
	for _, motion := range sequence[1:] {
		if motion != (MotionInput{}) {
			t.Fatal("completed route did not pad with idle controls")
		}
	}
	if planner.at != at || !reflect.DeepEqual(commands, planner.commands) || !reflect.DeepEqual(states, planner.states) || forecastDigest(w) != before {
		t.Fatal("reading a guard sequence changed the plan or live world")
	}

	for _, fixture := range []struct {
		name   string
		modify func(*nativeMotionPlanner) MotionInput
	}{
		{"different-world", func(p *nativeMotionPlanner) MotionInput { p.world = &World{}; return planned.Motion }},
		{"not-consumed", func(p *nativeMotionPlanner) MotionInput { p.at = 0; return planned.Motion }},
		{"past-end", func(p *nativeMotionPlanner) MotionInput { p.at = len(p.commands) + 1; return planned.Motion }},
		{"different-command", func(p *nativeMotionPlanner) MotionInput { return MotionInput{Up: true} }},
		{"missing-state", func(p *nativeMotionPlanner) MotionInput { p.states = nil; return planned.Motion }},
		{"different-before-state", func(p *nativeMotionPlanner) MotionInput {
			p.states = append([]demoMotionForecast(nil), p.states...)
			p.states[0].player.X++
			return planned.Motion
		}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			copy := planner
			motion := fixture.modify(&copy)
			before := copy
			if sequence, ok := copy.guardSequence(w, motion); ok || sequence != ([6]MotionInput{}) {
				t.Fatal("invalid retained plan supplied guard commands")
			}
			if !reflect.DeepEqual(copy, before) {
				t.Fatal("invalid guard preview consumed or invalidated its plan")
			}
		})
	}
}

func TestNativeMotionGuardSequenceKeepsRemainingTurnsOptional(t *testing.T) {
	w := nativeMotionFixture(t, 193, 176, 1896)
	var planner nativeMotionPlanner
	motion, ok := planner.command(w, 202, 2077, 256, 1832)
	if !ok || len(planner.commands) != 5 {
		t.Fatalf("original five-command rear corner differs: %+v", planner.commands)
	}
	sequence, ok := planner.guardSequence(w, motion)
	if !ok {
		t.Fatal("committed rear-corner sequence was rejected")
	}
	for index, command := range planner.commands {
		if sequence[index] != command {
			t.Fatalf("guard changed retained turn %d", index)
		}
	}
	if sequence[5] != (MotionInput{}) {
		t.Fatal("guard did not idle after the completed corner")
	}
}

func TestExpertWorldGuardUsesCommittedCornerRatherThanHeldDirectionOptional(t *testing.T) {
	w, planner, planned := nativeGuardCannonFixture(t)
	sequence, ok := planner.guardSequence(w, planned.Motion)
	if !ok {
		t.Fatal("original corner sequence was not available")
	}
	advance := func(useSequence bool) ForecastResult {
		var forecast WorldForecast
		if err := forecast.Load(w); err != nil {
			t.Fatal(err)
		}
		var result ForecastResult
		for pass := 0; pass < 6; pass++ {
			for range 3 {
				forecast.AdvancePALTick()
			}
			input := planned
			if useSequence {
				input.Motion = sequence[pass]
			}
			var err error
			result, err = forecast.Advance(input)
			if err != nil {
				t.Fatal(err)
			}
			if result.Boundary != ForecastRunning {
				break
			}
		}
		return result
	}
	held, committed := advance(false), advance(true)
	if held.Alive && held.Shield >= w.Equipment.Shield {
		t.Fatalf("holding the first direction did not meet the source gun: %+v", held)
	}
	if !committed.Alive || committed.Shield != w.Equipment.Shield {
		t.Fatalf("actual corner plus idle was not safe: %+v", committed)
	}
	before := forecastDigest(w)
	pilot := PresentationPilot{planner: DemoPilot{nativeMotion: planner}}
	if got := pilot.forecastOpeningGuard(w, planned); got != planned {
		t.Fatalf("guard replaced a safe retained corner: planned%+v got%+v", planned, got)
	}
	if pilot.planner.nativeMotion.at != planner.at || forecastDigest(w) != before {
		t.Fatal("guard consumed the native plan or changed the live world")
	}
	pilot.planner.nativeMotion.world = nil
	if got := pilot.forecastOpeningGuard(w, planned); got.Motion == planned.Motion {
		t.Fatal("an invalid retained plan bypassed ordinary held-direction protection")
	}
	// The opening still evaluates its held action. Retained corridor plans must
	// not change that independently verified policy, even if one is attached.
	w.ThirdMiddle = nil
	pilot.planner.nativeMotion = planner
	if got := pilot.forecastOpeningGuard(w, planned); got.Motion == planned.Motion {
		t.Fatal("corridor sequence evaluation leaked into the opening guard")
	}
}
