package engine

import "testing"

// This arranges the recorded right-pocket pose to isolate route recovery. The
// original terrain, stage camera bounds and World.Step own all subsequent motion.
func TestPresentationPilotRetreatsFromOriginalFirstStageRightPocketOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 3006, 3022, 3022
	w.Player.X, w.Player.Y, w.Player.SpeedTier = 250, 176, 2
	w.Equipment.SpeedTier = 2
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.cursor = RestartEncounterCursor(w.ScrollY)
	// This is a terrain/controller fixture, not a survival or campaign proof.
	w.InvulnerableFrames = 2000
	p := PresentationPilot{}
	backward, crossed := 0, false
	maximum := w.ScrollY
	for pass := 0; pass < 900; pass++ {
		input := p.NormalInput(w)
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		if w.ScrollDelta < 0 && input.Motion.Down {
			backward++
		}
		maximum = max(maximum, w.ScrollY)
		crossed = crossed || w.Player.X < 140
		if crossed && p.planner.retreatGoal == 0 && w.ScrollY+w.Player.Y <= 3060 {
			if backward < 10 || maximum < 3100 || !w.PlayerAlive {
				t.Fatal("right-pocket route bypassed genuine reverse motion or terrain collision")
			}
			t.Logf("Retreated to camera%d, crossed left and resumed forward after%d ordinary inputs (%d reverse passes)", maximum, pass+1, backward)
			return
		}
	}
	t.Fatalf("right pocket not escaped: camera%d max%d xy%d,%d reverse%d crossed%v rewind%d", w.ScrollY, maximum, w.Player.X, w.Player.Y, backward, crossed, w.Rewind.Timer)
}
