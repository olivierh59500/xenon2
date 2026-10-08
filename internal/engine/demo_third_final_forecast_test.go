package engine

import "testing"

// This arranges the original final checkpoint and advances its complete source
// controllers. Repositioning the ship below a selected native pose isolates an
// aiming or avoidance decision; it is not a connected campaign fixture.
func thirdFinalAdmittedSourceFixture(t testing.TB, passes int) *World {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 176, 152
	w.MinimumScrollY = 0
	w.RestartCheckpoint()
	w.Ready, w.MaterializationFrames = false, 0
	for range passes {
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if w.ThirdFinal.LaunchCount != 1 || w.ThirdFinal.Health != 80 || !w.PlayerAlive {
		t.Fatal("original final checkpoint did not admit an undamaged worm")
	}
	return w
}

// Follow the first ordinary primary's real point callbacks, rather than its
// predicted screen coordinates. Later volleys cannot claim this shot's damage.
func ordinaryThirdFirstPrimaryHitWithinAimWindow(t testing.TB, w *World) bool {
	t.Helper()
	if w.Equipment.Primary.Item != ItemForwardShot {
		t.Fatal("first-primary fixture requires its original single basic gun")
	}
	firstID, observed, queried := 0, 0, false
	var healthBefore uint16
	w.Weapons.context.ObservePointImpact = func(event WeaponPointImpact) {
		if event.Kind != "small-shot" || event.OwnerSlot != 0 {
			return
		}
		if firstID == 0 {
			firstID = event.ProjectileID
		}
		if event.ProjectileID == firstID {
			observed++
			queried, healthBefore = true, w.ThirdFinal.Health
		}
	}
	defer func() { w.Weapons.context.ObservePointImpact = nil }()
	for range 18 {
		queried = false
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(Input{Fire: true}); err != nil {
			t.Fatal(err)
		}
		if queried && w.ThirdFinal.Health < healthBefore {
			return true
		}
		if firstID != 0 {
			active := false
			for _, shot := range w.Weapons.projectiles {
				active = active || shot.Render.ID == firstID && shot.Render.Active
			}
			if !active {
				if observed == 0 {
					t.Fatal("ordinary primary retired without its point callback")
				}
				return false
			}
		}
	}
	if observed == 0 {
		t.Fatal("first primary never reached a point query in the unchanged aim window")
	}
	return false
}

func TestThirdFinalAimMatchesFirstNativePrimaryRatherThanLinearLeadOptional(t *testing.T) {
	for _, fixture := range []struct {
		name       string
		passes, x  int
		usefulShot bool
	}{
		{"curving-head-reachable", 20, 134, true},
		{"linear-lead-misses", 16, 143, false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			w := thirdFinalAdmittedSourceFixture(t, fixture.passes)
			w.Player.X, w.Player.Y = fixture.x, 176
			before := forecastIsolationDigest(w)
			if got := presentationShotOpportunityForMotion(w, MotionInput{}); got != fixture.usefulShot {
				t.Fatalf("primary aim disagrees with the original curved-head callback: useful%v want%v", got, fixture.usefulShot)
			}
			var forecast WorldForecast
			opportunity, supported := presentationGuardianShotOpportunity(w, MotionInput{}, &forecast, 3)
			if !supported || opportunity != fixture.usefulShot {
				t.Fatalf("native head prediction: supported%v useful%v want%v", supported, opportunity, fixture.usefulShot)
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("first-primary head prediction changed the live world")
			}
			if got := ordinaryThirdFirstPrimaryHitWithinAimWindow(t, w); got != fixture.usefulShot {
				t.Fatalf("actual ordinary point callback disagrees: useful%v want%v", got, fixture.usefulShot)
			}
		})
	}
}

func TestThirdFinalPointObserverRejectsBodyAndPreservesOtherScopesOptional(t *testing.T) {
	w := thirdFinalAdmittedSourceFixture(t, 20)
	for _, actor := range w.Actors {
		if !actor.Active || actor.thirdFinalMember == nil || actor.Collision.Empty() {
			continue
		}
		_, target := presentationTargetBounds(w, actor)
		if target != (actor.thirdPart.Index == 0) {
			t.Fatal("final point observer selected a blocking body or omitted its head")
		}
	}
	for _, fixture := range []struct {
		name   string
		modify func(*World)
	}{
		{"before-arena", func(w *World) { w.ScrollY = 209 }},
		{"not-launched", func(w *World) { w.ThirdFinal.LaunchCount = 0 }},
		{"defeated", func(w *World) { w.ThirdFinal.Defeated = true }},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			copy, final := *w, *w.ThirdFinal
			copy.ThirdFinal = &final
			fixture.modify(&copy)
			if presentationGuardianAimSupported(&copy) {
				t.Fatal("third-final point observer escaped its active source arena")
			}
		})
	}
}

func sixThirdFinalGuardPasses(t testing.TB, w *World, input Input) (ForecastResult, int) {
	t.Helper()
	var forecast WorldForecast
	if err := forecast.Load(w); err != nil {
		t.Fatal(err)
	}
	var result ForecastResult
	contacts := 0
	for range 6 {
		state := forecast.State()
		// The source player callback rebuilds this rectangle before movement.
		// A value copy provides exactly that geometry without changing State.
		view := *state
		view.updatePlayerCollision()
		for _, actor := range state.Actors {
			if actor.Active && actor.ActorList == "moving" && actor.Collision.Intersects(view.playerCollision) {
				contacts++
				break
			}
		}
		for range 3 {
			forecast.AdvancePALTick()
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
	return result, contacts
}

// Admit the second original selector at an arena pose. Its prior-pass heartbeat
// prevents the stage callback from admitting another group on the first pass.
// All subsequent positions, shots and shield changes come from ordinary Step.
func thirdFinalSecondLaunchSourceFixture(t testing.TB) *World {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 76, 256
	w.MinimumScrollY = 0
	w.RestartCheckpoint()
	w.Ready, w.MaterializationFrames = false, 0
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("second-launch checkpoint starts in terrain")
	}
	if launch, err := w.ThirdFinal.SelectLaunch(); err != nil || launch != 0 {
		t.Fatal("original launch selector did not begin at zero")
	}
	if err := w.spawnThirdFinal(); err != nil {
		t.Fatal(err)
	}
	w.thirdFinalUpdated = true
	for range 45 {
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if !w.PlayerAlive || w.Equipment.Shield != 27 || w.ThirdFinal.Health != 80 {
		t.Fatal("source second-launch approach changed")
	}
	return w
}

func TestThirdFinalGuardAnticipatesNativeFanAndPreMovementContactOptional(t *testing.T) {
	for _, fixture := range []struct {
		name                   string
		second                 bool
		heldShield, safeShield int
		contact                bool
	}{
		{"eight-way-fan", false, 35, 39, false},
		{"incoming-head-before-movement", true, 3, 27, true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			var w *World
			planned := Input{}
			if fixture.second {
				w = thirdFinalSecondLaunchSourceFixture(t)
				var head *WorldActor
				for _, actor := range w.Actors {
					if actor.Active && actor.thirdFinalMember != nil && actor.thirdPart.Index == 0 {
						head = actor
						break
					}
				}
				if head == nil || head.thirdFinalMember.Motion.PathID != 65 || int(head.X) != 256 || int(head.Y) != 138 {
					t.Fatal("original second head did not reach the selected native pose")
				}
				w.Player.X, w.Player.Y = int(head.X)+3, 176
				planned.Motion = MotionInput{Right: true}
			} else {
				w = thirdFinalAdmittedSourceFixture(t, 10)
				w.Player.X, w.Player.Y = 72, 176
			}
			w.Equipment.ApplyItem(ItemSpeedup)
			w.Equipment.ApplyItem(ItemSpeedup)
			w.Player.SpeedTier = w.Equipment.SpeedTier
			before := forecastIsolationDigest(w)
			pilot := PresentationPilot{}
			guarded := pilot.forecastOpeningGuard(w, planned)
			if guarded.Motion == planned.Motion || guarded.Fire != planned.Fire || guarded.Dive != planned.Dive {
				t.Fatal("active final guard did not replace the unsafe ordinary movement alone")
			}
			held, heldContacts := sixThirdFinalGuardPasses(t, w, planned)
			safe, safeContacts := sixThirdFinalGuardPasses(t, w, guarded)
			if !held.Alive || !safe.Alive || held.Shield != fixture.heldShield || safe.Shield != fixture.safeShield || safeContacts != 0 || (heldContacts > 0) != fixture.contact {
				t.Fatalf("native avoidance differs: held%+v/%d contacts guarded%+v/%d contacts", held, heldContacts, safe, safeContacts)
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("six-pass final guard or validation changed the live source world")
			}
		})
	}
}
