package app

import (
	"testing"

	"xenon2/internal/engine"
	"xenon2/internal/visualassets"
)

// This arranges the public driver boundary exposed by native Supernova: the
// source frame has incremented, but its remaining callbacks wait31PALticks.
// Engine continuation tests separately prove the actual pickup/list boundary.
func TestDemoWaitsForCompletedSupernovaFrameBeforeSampling(t *testing.T) {
	for _, human := range []bool{false, true} {
		g := frontendGame(t)
		g.Screen = LevelScreen
		g.Config.Demo, g.Config.HumanDemo = true, human
		driver := g.Driver.(*worldDriver)
		w := driver.world
		w.Ready, w.MaterializationFrames = false, 0
		g.demoControls(inputFrame{})
		d := g.demo
		completedFrame, level := d.logicFrame, d.level
		held := inputFrame{gameMotion: engine.MotionInput{Left: true}, fire: true, divePressed: true}
		d.controls = held
		// The source continuation resumes this incremented frame onPAL31.
		w.Frame = completedFrame + 1
		w.ScreenClearFrames, w.ScreenClearPaletteMask = 31, 0x1234
		frame, camera, player := w.Frame, w.ScrollY, w.Player
		// Isolate one real PAL tick per display update and keep the next
		// ordinary logic boundary beyond this31-tick effect fixture.
		g.palClock = engine.NewFrameClock(1, 1)
		g.clock = engine.NewFrameClock(1, 100)
		held.divePressed = false
		for pal := 0; pal < 31; pal++ {
			advanceFrontend(t, g, inputFrame{})
			if got := d.controls; got != held {
				t.Fatalf("human%v PAL%d changed held controls or repeated Dive: %+v", human, pal, got)
			}
			if d.logicFrame != completedFrame || d.level != level {
				t.Fatalf("human%v PAL%d sampled the incomplete source frame", human, pal)
			}
			if w.Frame != frame || w.ScrollY != camera || w.Player != player {
				t.Fatal("public flash boundary advanced ordinary motion/frame")
			}
			if g.View.FreezeInterpolation != (pal < 30) || g.View.PaletteMask != w.ScreenClearPaletteMask {
				t.Fatalf("PAL%d view lost current strobe/freeze", pal+1)
			}
		}
		if w.ScreenClearFrames != 0 || g.View.PaletteMask != 0 || g.View.FreezeInterpolation {
			t.Fatal("PAL31 did not publish completed palette state")
		}
		advanceFrontend(t, g, inputFrame{})
		got := d.controls
		if d.logicFrame != frame || d.level != level || got == held || got.divePressed {
			t.Fatalf("human%v completed same-frame state retained partial controls: frame%d want%d controls%+v", human, d.logicFrame, frame, got)
		}
		if repeated := g.demoControls(inputFrame{}); repeated != got {
			t.Fatal("completed frame was sampled more than once")
		}
	}
}

func TestManualDemoTakeoverStillAppliesDuringSupernova(t *testing.T) {
	g := frontendGame(t)
	g.Screen = LevelScreen
	g.Config.Demo, g.Config.HumanDemo = true, true
	w := g.Driver.(*worldDriver).world
	w.ScreenClearFrames = 31
	manual := inputFrame{gameMotion: engine.MotionInput{Right: true}, fire: true, divePressed: true}
	if got := g.demoControls(manual); got != manual || g.DemoActive() || g.demo != nil {
		t.Fatal("Supernova demo sampling gate consumed the manual takeover action")
	}
}

// The carrier and reward use native constructors; only a single ordinary
// point-hit bullet and the pickup contact pose are arranged for this boundary.
func pendingDemoSupernova(t *testing.T) *Game {
	t.Helper()
	g := frontendGame(t)
	data := g.Driver.(*worldDriver).world.Level
	data.Encounters = &visualassets.Encounters{Moving: []visualassets.Wave{{TriggerY: 4608, EnemyKind: 0, Count: 1, PathID: 32, MotionBudget: 18}}}
	w, err := engine.NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	var carrier *engine.WorldActor
	for pass := 0; pass < 80; pass++ {
		if err = w.Step(engine.Input{}); err != nil {
			t.Fatal(err)
		}
		for _, actor := range w.Actors {
			if actor.Active && actor.CarriedReward == 18 && actor.Collision.Top >= 80 && actor.Collision.Bottom <= 150 {
				carrier = actor
				break
			}
		}
		if carrier != nil {
			break
		}
	}
	if carrier == nil {
		t.Fatal("original carrier did not enter the native fixture")
	}
	var next engine.WorldForecast
	if err = next.Load(w); err != nil {
		t.Fatal(err)
	}
	if _, err = next.Advance(engine.Input{}); err != nil {
		t.Fatal(err)
	}
	var box engine.CollisionRect
	for _, actor := range next.State().Actors {
		if actor.ID == carrier.ID {
			box = actor.Collision
		}
	}
	x, y := (box.Left+box.Right)/2, (box.Top+box.Bottom)/2
	shots, err := engine.AppendSmallWeaponShots(nil, engine.WeaponSlot{Item: engine.ItemForwardShot}, x, y+15)
	if err != nil {
		t.Fatal(err)
	}
	w.SmallShots = append(w.SmallShots, &engine.WorldSmallShot{ID: 999999, Active: true, Shot: shots[0]})
	if err = w.Step(engine.Input{}); err != nil {
		t.Fatal(err)
	}
	var pickup *engine.WorldCollectible
	for _, item := range w.Collectibles {
		if item.Active && item.Reward == 18 {
			pickup = item
			break
		}
	}
	if pickup == nil {
		t.Fatal("native carrier hit did not create Supernova")
	}
	w.Player = engine.PlayerMotionState{X: int(pickup.X), Y: int(pickup.Y)}
	w.PreviousPlayer = w.Player
	w.Rewind = engine.NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.Equipment.ApplyItem(engine.ItemBitmapShades)
	if err = w.Step(engine.Input{}); err != nil {
		t.Fatal(err)
	}
	if w.ScreenClearFrames != 31 || w.Equipment.ShadesFrames != 220 {
		t.Fatal("actual collection did not suspend the source remainder")
	}
	g.Screen = LevelScreen
	g.Config.Demo = true
	g.Config.LogicPALRefreshes = 1
	g.SetDriver(&worldDriver{world: w})
	g.demo = &demoDirector{screen: LevelScreen, phase: g.director.Phase, logicFrame: w.Frame - 1, level: 1, controls: inputFrame{gameMotion: engine.MotionInput{Left: true}, divePressed: true}}
	for range 30 {
		g.Driver.(*worldDriver).AdvancePALTick()
	}
	if w.ScreenClearFrames != 1 || w.Equipment.ShadesFrames != 220 {
		t.Fatal("source PAL wait advanced remaining timers early")
	}
	g.palClock = engine.NewFrameClock(1, 1)
	g.clock = engine.NewFrameClock(1, 1)
	return g
}

func TestDemoResamplesCompletedSupernovaBeforeSameUpdatePAL1Step(t *testing.T) {
	g := pendingDemoSupernova(t)
	w := g.Driver.(*worldDriver).world
	frame := w.Frame
	advanceFrontend(t, g, inputFrame{})
	if w.Frame != frame+1 || w.ScreenClearFrames != 0 || w.Equipment.ShadesFrames != 218 || g.demo.logicFrame != frame || g.demo.controls.divePressed {
		t.Fatalf("PAL1 new pass missed completed-frame sample: world%d cache%d want%d strobe%d shades%d", w.Frame, g.demo.logicFrame, frame, w.ScreenClearFrames, w.Equipment.ShadesFrames)
	}
}

func TestManualTakeoverOnSupernovaResumeUpdatePreservesAction(t *testing.T) {
	g := pendingDemoSupernova(t)
	w := g.Driver.(*worldDriver).world
	frame, x := w.Frame, w.Player.X
	manual := inputFrame{gameMotion: engine.MotionInput{Right: true}, fire: true, divePressed: true}
	advanceFrontend(t, g, manual)
	if g.DemoActive() || g.demo != nil || w.Frame != frame+1 || w.Player.X <= x || g.pendingDive {
		t.Fatal("resume update replaced/repeated the human takeover command")
	}
}
