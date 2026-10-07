package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

// The source barrier updater at 0x55d3c writes inclusive rectangle edges.
// Its ADD.B at 0x55d54 changes only the low byte of the bottom edge.
func TestFifthFinalBarrierBoundsPreserveSourceByteAddition(t *testing.T) {
	group := &visualassets.GuardianGroup{Components: make([]visualassets.GuardianComponent, 22)}
	group.Components[1] = visualassets.GuardianComponent{Behavior: "barrier-band", Health: 240}
	for _, test := range []struct{ top, bottom uint16 }{{0x0000, 0x0010}, {0x00ef, 0x00ff}, {0x00f0, 0x0000}, {0x00ff, 0x000f}, {0x0100, 0x0110}, {0x01f0, 0x0100}, {0xffef, 0xffff}, {0xfff0, 0xff00}, {0xffff, 0xff0f}} {
		state, err := NewFifthFinalGuardianState(group)
		if err != nil {
			t.Fatal(err)
		}
		state.Parts[0].Y = int(int16(test.top))
		random := NewRandomState()
		state.Advance(group, 0, 0, 1, 416, 0, 160, 176, &random)
		want := CollisionRect{Left: 48, Top: int(int16(test.top)), Right: 288, Bottom: int(int16(test.bottom))}
		if got := state.Parts[1].Collision; got != want {
			t.Fatalf("top word %04x: barrier %+v, want source %+v", test.top, got, want)
		}
	}
}

func fifthFinalBarrierWorld(t *testing.T) *World {
	t.Helper()
	w := fifthResourceWorld(t)
	if err := w.activateFifthGuardian(visualassets.FixedEncounter{}, true); err != nil {
		t.Fatal(err)
	}
	// Admit the source arena with its lower band at screen rows 64 through 80.
	// This controls the camera fixture, never the collision or damage outcome.
	w.ScrollY, w.ScrollDelta = 256, 160
	w.advanceFifthGuardian(true)
	w.ScrollDelta = 0
	return w
}

func TestOriginalFifthFinalInvisibleBandsBlockOrdinaryShotsOptional(t *testing.T) {
	w := fifthResourceWorld(t)
	if err := w.activateFifthGuardian(visualassets.FixedEncounter{}, true); err != nil {
		t.Fatal(err)
	}
	if w.weaponHitPoint(60, 76, 1) {
		t.Fatal("hidden constructor consumed a shot before its first updater")
	}
	w.ScrollY, w.ScrollDelta = 256, 160
	w.advanceFifthGuardian(true)
	for _, index := range []int{1, 2} {
		actor, part := w.fifthFinalActors[index], w.FifthFinal.Parts[index]
		if actor.Collision != part.Collision || actor.Collision.Empty() || actor.Visible || actor.Sprite != "" || actor.fifthPart.RenderMode != "none" || actor.part.ResourceTag != 324 {
			t.Fatalf("band %d lost its nonvisual source collision: actor %+v state %+v", index, actor.Collision, part.Collision)
		}
	}
	health, core, outer, random := w.FifthFinal.Parts[1].Health, w.FifthFinal.CoreHealth, w.FifthFinal.OuterRemaining, w.RandomState()
	shot := &WorldSmallShot{Active: true, Shot: SmallShot{X: 60, Y: 85, VelocityY: -9, Damage: 1}}
	w.advanceSmallShot(shot)
	if shot.Active || w.FifthFinal.Parts[1].Health != health || w.FifthFinal.CoreHealth != core || w.FifthFinal.OuterRemaining != outer || w.Score != 0 || w.RandomState() != random {
		t.Fatal("original barrier must consume the ordinary shot without damage, rewards or random draws")
	}
}

func TestOriginalFifthFinalBandConsumesPhysicalLaserBeforeLaterTargetOptional(t *testing.T) {
	w := fifthFinalBarrierWorld(t)
	// Append a diagnostic target behind the source band in the moving list.
	// A missing band would incorrectly damage it during the same beam traversal.
	target := &WorldActor{Active: true, ActorList: "moving", Health: 10, Collision: CollisionRect{Left: 55, Top: 64, Right: 65, Bottom: 80}, part: &visualassets.ActorPart{ResourceTag: 200, DamageMode: "individual"}}
	binding, err := w.reserveWorldActor(200, ActorPoolMoving, true)
	if err != nil {
		t.Fatal(err)
	}
	target.ID, target.Binding = binding.EntityID, binding
	w.poolActors[binding.Slot] = target
	w.Actors = append(w.Actors, target)
	runtime, err := NewWeaponRuntime(&visualassets.SpriteAtlas{})
	if err != nil {
		t.Fatal(err)
	}
	w.Weapons = runtime
	beam, err := w.reserveWorldActor(36, ActorPoolProjectile, false)
	if err != nil {
		t.Fatal(err)
	}
	beam.Residue.OwnerSlot = NoActorSlot
	runtime.mounts[0].X, runtime.mounts[0].Y = 62, 104
	runtime.projectiles = append(runtime.projectiles, runtimeWeaponProjectile{Binding: beam, Render: WeaponRenderItem{ID: beam.EntityID, Kind: "laser", Active: true}, Laser: LaserBeamState{}, Owner: 0})
	random := w.RandomState()
	if err := runtime.AdvanceProjectile(beam.EntityID, w.weaponContext(Input{}, false)); err != nil {
		t.Fatal(err)
	}
	if runtime.projectiles[0].Render.Active || w.Pool.Slot(beam.Slot).ResourceTag != 4 || target.Health != 10 || w.FifthFinal.Parts[1].Health != 240 || w.Score != 0 || w.RandomState() != random {
		t.Fatal("source barrier callback did not retire the physical beam before the following target")
	}
}

func TestOriginalFifthFactoriesRetainSourceContactStrengthOptional(t *testing.T) {
	for _, final := range []bool{false, true} {
		w := fifthResourceWorld(t)
		if err := w.activateFifthGuardian(visualassets.FixedEncounter{Y: 2336}, final); err != nil {
			t.Fatal(err)
		}
		group, actors := w.fifthMiddleArt, w.fifthMiddleActors[:]
		if final {
			group, actors = w.fifthFinalArt, w.fifthFinalActors[:]
		}
		for index, actor := range actors {
			if actor.part.StrongHealth != group.Components[index].StrongHealth {
				t.Fatalf("final %v part %d lost source strength", final, index)
			}
		}
	}
}

func TestOriginalFifthOrdinaryMountContactUsesEightPointDamageOptional(t *testing.T) {
	w := fifthFinalBarrierWorld(t)
	w.ScrollDelta = 160
	w.advanceFifthGuardian(true)
	w.ScrollDelta = 0
	w.Level.Encounters = &visualassets.Encounters{}
	w.Player = PlayerMotionState{X: 128, Y: 61}
	w.PreviousPlayer = w.Player
	w.MaterializationFrames = 0
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.Equipment.Shield != 31 || !w.PlayerAlive || !w.FifthFinal.Parts[3].Destroyed || w.FifthFinal.OuterRemaining != 17 || w.Score != 200 {
		t.Fatal("ordinary source mount contact must deal eight points before the surviving player's damage callback")
	}
}

func TestOriginalFifthMountWreckStopsBlockingWithinTheProjectilePhaseOptional(t *testing.T) {
	for _, final := range []bool{false, true} {
		w := fifthResourceWorld(t)
		w.ScrollY = 2240
		if err := w.activateFifthGuardian(visualassets.FixedEncounter{Y: 2336}, final); err != nil {
			t.Fatal(err)
		}
		w.ScrollDelta = 0
		if final {
			w.ScrollY, w.ScrollDelta = 96, 320
		}
		w.advanceFifthGuardian(final)
		mount := w.fifthMiddleActors[1]
		if final {
			mount = w.fifthFinalActors[3]
		}
		pointX, pointY := mount.Collision.Left, mount.Collision.Top
		following := &WorldActor{Active: true, ActorList: "moving", Health: 10, Collision: mount.Collision, part: &visualassets.ActorPart{ResourceTag: 200, DamageMode: "individual"}}
		binding, err := w.reserveWorldActor(200, ActorPoolMoving, true)
		if err != nil {
			t.Fatal(err)
		}
		following.ID, following.Binding = binding.EntityID, binding
		w.poolActors[binding.Slot] = following
		w.Actors = append(w.Actors, following)
		fire := func() {
			shot := &WorldSmallShot{Active: true, Shot: SmallShot{X: pointX, Y: pointY + 9, VelocityY: -9, Damage: 1}}
			w.advanceSmallShot(shot)
			if shot.Active {
				t.Fatal("ordinary point shot failed to reach the source target")
			}
		}
		beforeSprite, beforeSlot, random := mount.Sprite, mount.Binding.Slot, w.RandomState()
		for range mount.Health {
			fire()
		}
		if !mount.Active || !mount.Visible || mount.Sprite != beforeSprite || mount.Binding.Slot != beforeSlot || w.Pool.Slot(beforeSlot).ResourceTag != 276 || w.Score != 200 || w.RandomState() != random || following.Health != 10 {
			t.Fatal("mount death changed its retained actor, displayed pose, source reward or pool identity")
		}
		// No actor update occurs between destruction and this second target hit.
		// The native callbacks at 0x56690 and 0x56260 disable the wreck now.
		fire()
		if following.Health != 9 || mount.Collision.Left != 1000 || mount.Collision.Right != 1000 || w.Score != 200 || w.RandomState() != random {
			t.Fatalf("final %v: wreck consumed the later shot or repeated its reward", final)
		}
		w.advanceFifthGuardian(final)
		if mount.Sprite != mount.fifthPart.DestroyedSprite || !mount.Active {
			t.Fatal("the following source callback did not display the retained wreck")
		}
	}
}

func TestOriginalFifthBarrierPlayerContactUsesPriorCallbackAndSourceStrengthOptional(t *testing.T) {
	for _, test := range []struct {
		name                              string
		shield, want                      int
		protect, invulnerable, dive, dead bool
	}{{name: "heavy", shield: 39, want: 23}, {name: "protected", shield: 39, want: 31, protect: true}, {name: "invulnerable", shield: 39, want: 39, invulnerable: true}, {name: "diving", shield: 39, want: 39, dive: true}, {name: "lethal", shield: 16, want: 0, dead: true}} {
		t.Run(test.name, func(t *testing.T) {
			w := fifthFinalBarrierWorld(t)
			w.Level.Encounters = &visualassets.Encounters{}
			w.Player = PlayerMotionState{X: 60, Y: 72}
			w.PreviousPlayer = w.Player
			w.MaterializationFrames = 0
			w.Equipment.Shield, w.Equipment.Protection = test.shield, test.protect
			if test.invulnerable {
				w.InvulnerableFrames = 80
			}
			if test.dive {
				w.Dive = DiveState{Phase: 4, Remaining: 80}
			}
			// Move the controller's next pose away. Contact must still use the
			// rectangle published by the preceding callback, before actor updates.
			w.ScrollDelta = 100
			before := w.Player
			if err := w.Step(Input{Motion: MotionInput{Right: true}}); err != nil {
				t.Fatal(err)
			}
			if w.Equipment.Shield != test.want || w.PlayerAlive == test.dead || w.FifthFinal.Parts[1].Health != 240 || w.Score != 0 {
				t.Fatalf("contact shield %d alive %v, want %d alive %v", w.Equipment.Shield, w.PlayerAlive, test.want, !test.dead)
			}
			if test.dead && (w.Player.X != before.X || w.Player.Y != before.Y) {
				t.Fatal("lethal barrier contact continued player movement")
			}
			if w.fifthFinalActors[1].Collision.Top != 164 {
				t.Fatal("player contact prevented the later actor callback")
			}
		})
	}
}
