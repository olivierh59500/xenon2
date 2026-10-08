package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func diveTerrainWorld(t *testing.T) *World {
	t.Helper()
	w := testWorld(t)
	w.Level.Encounters = &visualassets.Encounters{}
	w.Level.PlayerStencil = &visualassets.PlayerTerrainStencil{Width: 1, Height: 1, Rows: []uint32{0x80000000}}
	w.Coverage = &TerrainCoverage{Columns: 20, Rows: 300, Map: make([]uint16, 6000), coverage: make(map[uint16][16]uint16)}
	w.Player = PlayerMotionState{X: 160, Y: 100}
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 4000, 4032, 4032
	w.Rewind = NewTerrainRewind(4000, 160, 100)
	return w
}

func TestDiveRequestedOnTerrainContactStillRestoresPendingHistory(t *testing.T) {
	w := diveTerrainWorld(t)
	// A single covered pixel lies three pixels ahead. The ordinary input pass
	// touches it before the end-of-pass dive request is admitted.
	var tile [16]uint16
	tile[1] = 0x8000
	w.Coverage.coverage[1] = tile
	w.Coverage.Map[256*20+10] = 1
	w.Equipment.DiveCharges = 1
	if w.Coverage.Touches(160, 100, 4000, *w.Level.PlayerStencil) || !w.Coverage.Touches(160, 97, 4000, *w.Level.PlayerStencil) {
		t.Fatal("test must enter its covered pixel through ordinary movement")
	}
	if err := w.Step(Input{Motion: MotionInput{Up: true}, Dive: true}); err != nil {
		t.Fatal(err)
	}
	if w.Player.Y != 97 || w.Rewind.Timer != 1 || w.Dive.Phase != 1 || w.Equipment.DiveCharges != 0 {
		t.Fatal("ordinary contact followed by dive did not retain its pending history")
	}
	history, shield := w.Rewind, w.Equipment.Shield
	forecast := newDemoMotionForecast(w)
	input := MotionInput{Right: true, Up: true}
	if !forecast.advance(w, input) || w.Rewind != history {
		t.Fatal("pending underwater history must be forecast without changing the live game")
	}
	if err := w.Step(Input{Motion: input}); err != nil {
		t.Fatal(err)
	}
	if w.Player.X != 160 || w.Player.Y != 101 || w.Player.Inertia != 0 || w.Rewind.Timer != -1 || w.Dive.Phase != 2 || !w.PlayerAlive || w.Equipment.Shield != shield {
		t.Fatalf("underwater pending rewind skipped the source history: player%+v timer%d dive%+v", w.Player, w.Rewind.Timer, w.Dive)
	}
	if forecast.player != w.Player || forecast.scroll.Y != w.ScrollY || forecast.scroll.Maximum != w.MaximumScrollY || forecast.rewind != w.Rewind {
		t.Fatal("movement forecast diverged from the real underwater rewind")
	}
}

func TestUnderwaterPendingRewindRetainsOriginalCrushBoundary(t *testing.T) {
	w := diveTerrainWorld(t)
	var solid [16]uint16
	for row := range solid {
		solid[row] = 0xffff
	}
	w.Coverage.coverage[1] = solid
	for index := range w.Coverage.Map {
		w.Coverage.Map[index] = 1
	}
	w.Dive = DiveState{Phase: 4, Remaining: 100}
	w.Rewind.Timer = -16
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if !w.PlayerAlive || w.Rewind.Timer != -17 || w.Dive.Phase != 4 {
		t.Fatal("pending underwater rewind must retain its final reverse pass")
	}
	forecast := newDemoMotionForecast(w)
	if forecast.advance(w, MotionInput{Right: true}) {
		t.Fatal("forecast accepted the original crushing boundary underwater")
	}
	if err := w.Step(Input{Motion: MotionInput{Right: true}}); err != nil {
		t.Fatal(err)
	}
	if w.PlayerAlive || w.Equipment.Shield != 0 || w.Rewind.Timer != -17 {
		t.Fatal("diving incorrectly cancelled crushing from an existing rewind")
	}
}

func TestUnderwaterMotionCreatesNoNewRewindAndClearsRecoveredHistory(t *testing.T) {
	for _, timer := range []int{0, -1} {
		w := diveTerrainWorld(t)
		w.Dive = DiveState{Phase: 4, Remaining: 100}
		w.Rewind.Timer = timer
		input := MotionInput{Right: true, Up: true}
		forecast := newDemoMotionForecast(w)
		if !forecast.advance(w, input) {
			t.Fatal("clear underwater motion was rejected")
		}
		if err := w.Step(Input{Motion: input}); err != nil {
			t.Fatal(err)
		}
		if w.Player.X != 163 || w.Player.Y != 97 || w.Rewind.Timer != 0 || !w.PlayerAlive || forecast.player != w.Player || forecast.rewind != w.Rewind {
			t.Fatal("recovered underwater history prevented ordinary controls")
		}
	}
	w := diveTerrainWorld(t)
	w.Dive = DiveState{Phase: 4, Remaining: 100}
	var tile [16]uint16
	tile[1] = 0x8000
	w.Coverage.coverage[1] = tile
	w.Coverage.Map[256*20+10] = 1
	if err := w.Step(Input{Motion: MotionInput{Up: true}}); err != nil {
		t.Fatal(err)
	}
	if w.Rewind.Timer != 0 || !w.Coverage.Touches(w.Player.X, w.Player.Y, w.RenderScrollY, *w.Level.PlayerStencil) || !w.PlayerAlive {
		t.Fatal("a new underwater terrain contact incorrectly started a rewind")
	}
}
