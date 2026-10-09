package engine

import (
	"fmt"
	"testing"

	"xenon2/internal/visualassets"
)

func TestThirdGuardKeepsTerrainProtectionAfterOpeningCheckpoints(t *testing.T) {
	for _, checkpoint := range []int{4608, 3408, 3072} {
		t.Run(fmt.Sprint(checkpoint), func(t *testing.T) {
			w := testWorld(t)
			w.Level.Number, w.Checkpoint.ScrollY = 3, checkpoint
			w.Level.Encounters = &visualassets.Encounters{}
			w.Player = PlayerMotionState{X: 160, Y: 100}
			w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 1000, 1016, 1016
			w.Rewind = NewTerrainRewind(1000, 160, 100)
			w.Level.PlayerStencil = &visualassets.PlayerTerrainStencil{Width: 1, Height: 1, Rows: []uint32{0x80000000}}
			w.Coverage = &TerrainCoverage{Columns: 20, Rows: 300, Map: make([]uint16, 6000), coverage: make(map[uint16][16]uint16)}
			var tile [16]uint16
			tile[11] = 1 << 9
			w.Coverage.coverage[1] = tile
			w.Coverage.Map[68*20+10] = 1
			planned := Input{Motion: MotionInput{Right: true}}
			before := forecastDigest(w)
			pilot := PresentationPilot{}
			guarded := pilot.forecastOpeningGuard(w, planned)
			if forecastDigest(w) != before || guarded.Motion == planned.Motion {
				t.Fatal("pre-arena guard omitted terrain protection or changed the live world")
			}
			for range 6 {
				if err := w.Step(guarded); err != nil {
					t.Fatal(err)
				}
				if !w.PlayerAlive || w.Rewind.Timer != 0 || w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
					t.Fatal("guarded ordinary controls entered the pending terrain rewind")
				}
			}
		})
	}
}
