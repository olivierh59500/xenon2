package app

import (
	"math"
	"testing"

	"xenon2/internal/controls"
)

func TestMobileSystemGestureAndResumeKeepExpertInSameSession(t *testing.T) {
	g := menuAdmittedGame(t, 2, 1)
	awaitFrontendBoundary(t, g, 120, "gameplay fade completion", func() bool { return g.fade == nil || g.fade.Done })
	g.Config.Demo, g.Config.HumanDemo = true, true
	d := g.Driver.(*worldDriver)
	world, frame := d.world, d.world.Frame
	var guard controls.GestureGuard
	var pad controls.Pad
	pad.Place(controls.NewLayout(1344))
	areas := controls.GestureInsets{Left: 12, Top: 8, Right: 12, Bottom: 24}
	steps := [][]controls.Touch{
		{{ID: 7, X: 672, Y: 599, Pressed: true}},
		{{ID: 7, X: 672, Y: 400}},
		nil, // Android cancels the contact before switching applications.
		nil, // The synthetic cursor is reset when the view resumes.
	}
	for index, contacts := range steps {
		raw := inputFrame{mouseX: 50 + index*37, mouseY: 100 - index*25}
		advanceFrontend(t, g, mergeMobileInput(raw, pad.Update(guard.Filter(nil, contacts, 1344, controls.Height, areas))))
		if !g.DemoActive() || !g.Config.HumanDemo || g.demo == nil || d.world != world || d.diagnostic || world.Cheats.Enabled() {
			t.Fatal("system gesture or resume canceled the expert or replaced the active session")
		}
	}
	for range 60 {
		advanceFrontend(t, g, mergeMobileInput(inputFrame{}, pad.Update(nil)))
	}
	if d.world.Frame <= frame || !g.DemoActive() {
		t.Fatal("expert stopped advancing after application resume")
	}
	// A deliberate game control still takes over on its first press.
	r := pad.Layout.Buttons[controls.Fire].Bounds
	touch := controls.Touch{ID: 7, X: r.X + r.Width/2, Y: r.Y + r.Height/2, Pressed: true}
	advanceFrontend(t, g, mergeMobileInput(inputFrame{}, pad.Update(guard.Filter(nil, []controls.Touch{touch}, 1344, controls.Height, areas))))
	if g.DemoActive() || g.demo != nil || d.world != world {
		t.Fatal("real fire touch did not take over the same active session")
	}
}

func TestMobileDemoCanvasTapAndReleaseKeepExpertAdmission(t *testing.T) {
	g := frontendGame(t)
	var pad controls.Pad
	pad.Place(controls.NewLayout(1344))
	tap := controls.Touch{ID: 3, X: pad.Layout.SceneX + 160*3, Y: 170 * 3, Pressed: true}
	advanceFrontend(t, g, mergeMobileInput(inputFrame{}, pad.Update([]controls.Touch{tap})))
	if !g.DemoActive() || !g.Config.HumanDemo {
		t.Fatal("canvas tap did not select the ordinary expert demonstration")
	}
	for update := 0; update < 1800; update++ {
		// The touch pointer disappears and the platform cursor may change when
		// the view is recreated or returns from another application.
		raw := inputFrame{mouseX: update % 1344, mouseY: update % controls.Height}
		advanceFrontend(t, g, mergeMobileInput(raw, pad.Update(nil)))
		if !g.DemoActive() {
			t.Fatal("releasing the menu tap or resetting the synthetic cursor canceled the demo")
		}
		if d, ok := g.Driver.(*worldDriver); ok && d.session != nil && d.world.Frame > 20 && !d.world.Ready && g.Screen == LevelScreen {
			if d.diagnostic || d.world.Cheats.Enabled() || d.world.Level.Number != 1 {
				t.Fatal("mobile menu admission bypassed normal game rules")
			}
			return
		}
	}
	t.Fatal("mobile demo selection did not reach normal gameplay")
}

func TestMobileGestureInsetsUseReportedViewCoordinates(t *testing.T) {
	before := androidGestureInsets.Load()
	defer androidGestureInsets.Store(before)
	SetMobileGestureInsets(48, 18, 36, 72, 2400, 1080)
	g := MobileGame{width: 1344}
	got := g.gestureInsets()
	if math.Abs(got.Left-26.88) > 1e-9 || math.Abs(got.Top-10) > 1e-9 || math.Abs(got.Right-20.16) > 1e-9 || math.Abs(got.Bottom-40) > 1e-9 {
		t.Fatalf("platform insets were not scaled to the Go view: %+v", got)
	}
	SetMobileGestureInsets(0, 0, 0, 0, 0, 0)
	if after := g.gestureInsets(); after != got {
		t.Fatal("an unmeasured Android layout erased valid gesture bounds")
	}
	SetMobileGestureInsets(0, 0, 0, 0, 2400, 1080)
	if after := g.gestureInsets(); after != (controls.GestureInsets{Left: 12, Top: 8, Right: 12, Bottom: 24}) {
		t.Fatal("hidden system bars removed the reserved gesture edges")
	}
}
