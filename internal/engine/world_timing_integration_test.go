package engine

import (
	"fmt"
	"testing"

	"xenon2/internal/visualassets"
)

// scheduledSession uses the app's PAL-before-logic dispatch order. Omitting
// updates while paused retains both clock phases; it does not test audio replay.
type scheduledSession struct {
	session              *Session
	logic, pal           FrameClock
	logicCalls, palTicks int
}

func newScheduledSession(t *testing.T, refreshes, display, players int) *scheduledSession {
	t.Helper()
	data := testWorld(t).Level
	data.Encounters = &visualassets.Encounters{}
	s, err := NewSession(data, players, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range s.Players {
		if w != nil {
			w.Ready = false
			w.MaterializationFrames = 0
		}
	}
	return &scheduledSession{session: s, logic: NewRationalFrameClock(50, refreshes, display), pal: NewFrameClock(50, display)}
}

func (s *scheduledSession) advance(t *testing.T, paused bool, input func(int) Input, after func(int)) {
	t.Helper()
	if paused {
		return
	}
	for ticks := s.pal.Advance(); ticks > 0; ticks-- {
		s.session.ActiveWorld().AdvancePALTick()
		s.palTicks++
	}
	for calls := s.logic.Advance(); calls > 0; calls-- {
		controls := Input{}
		if input != nil {
			controls = input(s.logicCalls)
		}
		if _, err := s.session.Advance(controls); err != nil {
			t.Fatal(err)
		}
		s.logicCalls++
		if after != nil {
			after(s.logicCalls)
		}
	}
}

func timingWorldEquivalent(a, b *World) bool {
	return a.Frame == b.Frame && a.Player == b.Player && a.ScrollY == b.ScrollY && a.Equipment == b.Equipment && a.Dive == b.Dive && a.MaterializationFrames == b.MaterializationFrames && a.ScreenClearFrames == b.ScreenClearFrames && a.ScreenClearPaletteMask == b.ScreenClearPaletteMask && a.RandomState() == b.RandomState()
}

func TestPALThreeRefreshSessionMatchesPrelogicGroupedSchedule(t *testing.T) {
	for _, display := range []int{50, 60, 120, 144} {
		t.Run(fmt.Sprint(display), func(t *testing.T) {
			clocked, grouped := newScheduledSession(t, 3, display, 1), newScheduledSession(t, 3, display, 1)
			for _, s := range []*scheduledSession{clocked, grouped} {
				w := s.session.ActiveWorld()
				w.Equipment.SuperFrames, w.Equipment.ShadesFrames = 12, 10
				w.Dive = DiveState{Phase: 4, Remaining: 8}
			}
			input := func(pass int) Input {
				return Input{Motion: MotionInput{Right: pass < 12, Left: pass >= 24 && pass < 36}}
			}
			for range display * 3 {
				clocked.advance(t, false, input, func(pass int) {
					for range 3 {
						grouped.session.ActiveWorld().AdvancePALTick()
					}
					if _, err := grouped.session.Advance(input(pass - 1)); err != nil {
						t.Fatal(err)
					}
					if pass == 5 {
						clocked.session.ActiveWorld().applyCarrierReward(18)
						grouped.session.ActiveWorld().applyCarrierReward(18)
					}
					if clocked.palTicks != pass*3 || !timingWorldEquivalent(clocked.session.ActiveWorld(), grouped.session.ActiveWorld()) {
						t.Fatalf("display%d logic%d failed exact three-refresh/session correspondence", display, pass)
					}
				})
			}
			if clocked.logicCalls != 50 || clocked.palTicks != 150 {
				t.Fatalf("three-second totals differ: %d/%d", clocked.logicCalls, clocked.palTicks)
			}
		})
	}
}

func TestPALThreeSupernovaResumesOnFirstEligibleLogicBoundary(t *testing.T) {
	s := newScheduledSession(t, 3, 60, 1)
	w := s.session.ActiveWorld()
	w.Equipment.SuperFrames, w.Equipment.ShadesFrames = 7, 9
	w.Dive.Remaining = 6
	w.applyCarrierReward(18)
	for update := 1; update <= 60; update++ {
		s.advance(t, false, nil, nil)
		if s.palTicks < 31 && (w.Frame != 0 || w.Equipment.SuperFrames != 7 || w.Equipment.ShadesFrames != 9 || w.Dive.Remaining != 6) {
			t.Fatal("palette-only refreshes advanced gameplay timers")
		}
		if s.palTicks == 31 && (w.ScreenClearFrames != 0 || w.ScreenClearPaletteMask != 0 || w.Frame != 0) {
			t.Fatal("31st PAL tick did not end the strobe independently of logic")
		}
		if w.Frame != 0 {
			if s.logicCalls != 11 || s.palTicks != 33 || update != 40 || w.Frame != 1 || w.Equipment.SuperFrames != 6 || w.Equipment.ShadesFrames != 8 || w.Dive.Remaining != 5 {
				t.Fatalf("resumption boundary: update%d calls%d PAL%d world%+v", update, s.logicCalls, s.palTicks, w.Equipment)
			}
			return
		}
	}
	t.Fatal("strobe prevented resumed simulation")
}

func TestPALThreePostlogicReplayGroupingDiffersAtStrobeBoundary(t *testing.T) {
	correct, legacy := newScheduledSession(t, 3, 60, 1), newScheduledSession(t, 3, 60, 1)
	correct.session.ActiveWorld().applyCarrierReward(18)
	legacy.session.ActiveWorld().applyCarrierReward(18)
	for range 40 {
		correct.advance(t, false, nil, nil)
	}
	for range 11 {
		if _, err := legacy.session.Advance(Input{}); err != nil {
			t.Fatal(err)
		}
		for range 3 {
			legacy.session.ActiveWorld().AdvancePALTick()
		}
	}
	if correct.session.ActiveWorld().Frame != 1 || legacy.session.ActiveWorld().Frame != 0 {
		t.Fatal("regression did not distinguish the app's PAL-before-logic boundary from a postlogic replay shortcut")
	}
}

func TestPALThreePausedScheduleRetainsEffectAndClockPhases(t *testing.T) {
	paused, reference := newScheduledSession(t, 3, 60, 1), newScheduledSession(t, 3, 60, 1)
	for _, s := range []*scheduledSession{paused, reference} {
		s.session.ActiveWorld().applyCarrierReward(18)
		for range 11 {
			s.advance(t, false, nil, nil)
		}
	}
	beforeLogic, beforePAL := paused.logic, paused.pal
	beforeRandom, beforeFrames := paused.session.ActiveWorld().RandomState(), paused.session.ActiveWorld().ScreenClearFrames
	for range 180 {
		paused.advance(t, true, nil, nil)
	}
	if paused.logic != beforeLogic || paused.pal != beforePAL || paused.session.ActiveWorld().RandomState() != beforeRandom || paused.session.ActiveWorld().ScreenClearFrames != beforeFrames {
		t.Fatal("paused display updates advanced either clock or the palette effect")
	}
	for range 120 {
		paused.advance(t, false, nil, nil)
		reference.advance(t, false, nil, nil)
	}
	if paused.logic != reference.logic || paused.pal != reference.pal || paused.logicCalls != reference.logicCalls || paused.palTicks != reference.palTicks || !timingWorldEquivalent(paused.session.ActiveWorld(), reference.session.ActiveWorld()) {
		t.Fatal("resume discarded fractional clock phases or shifted world timers")
	}
}

func TestPALThreeAlternatingDeathRetainsIncomingTimerUntilReady(t *testing.T) {
	s := newScheduledSession(t, 3, 60, 2)
	first, second := s.session.Players[0], s.session.Players[1]
	frames := make([]visualassets.AnimationFrame, 9)
	for index := range frames {
		frames[index] = visualassets.AnimationFrame{Sprite: fmt.Sprintf("death-%d", index), Duration: 1}
	}
	frames[8].Duration = 0
	first.commonAnimations["player-death"] = visualassets.NamedActorAnimation{Ending: "hold", Animation: visualassets.ActorAnimation{Frames: frames}}
	first.Checkpoint.PlayerX, first.Checkpoint.ScrollY, first.Checkpoint.Money = 100, 3000, 200
	second.Checkpoint.PlayerX, second.Checkpoint.ScrollY, second.Checkpoint.Money = 140, 3100, 300
	second.Equipment.ShadesFrames = 9
	first.damagePlayer(127)
	for update := 0; update < 180 && s.session.Current == 0; update++ {
		s.advance(t, false, nil, nil)
	}
	if s.session.Current != 1 || s.logicCalls != 17 || first.Equipment.Lives != 2 || !second.Ready || !second.PlayerAlive || second.Player.X != 140 || second.ScrollY != 3100 || second.Money != 300 || second.Equipment.ShadesFrames != 9 || s.palTicks != s.logicCalls*3 {
		t.Fatal("three-refresh death admission lost the incoming checkpoint/timer or PAL correspondence")
	}
	for range 60 {
		s.advance(t, false, nil, nil)
	}
	if second.Frame != 0 || second.Equipment.ShadesFrames != 9 {
		t.Fatal("waiting READY advanced the incoming gameplay timer")
	}
	s.advance(t, false, func(int) Input { return Input{Fire: true} }, nil)
	for attempts := 0; attempts < 12 && second.Ready; attempts++ {
		s.advance(t, false, func(int) Input { return Input{Fire: true} }, nil)
	}
	if second.Ready {
		t.Fatal("bounded READY acceptance failed")
	}
	for attempts := 0; attempts < 12 && second.Frame == 0; attempts++ {
		s.advance(t, false, nil, nil)
	}
	if second.Frame != 1 || second.Equipment.ShadesFrames != 8 || !second.PlayerAlive {
		t.Fatal("first resumed logic pass did not consume exactly one incoming timer step")
	}
}
