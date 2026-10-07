package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func laserBlockerWorld(t *testing.T, blocker int) (*World, [3]*WorldActor, int) {
	t.Helper()
	w := testWorld(t)
	var targets [3]*WorldActor
	for index := 2; index >= 0; index-- {
		mode := "individual"
		if index == blocker || blocker == -1 && index == 0 {
			mode = "block-shot"
		}
		target := &WorldActor{Active: true, ActorList: "moving", Health: 10, Collision: CollisionRect{Left: 100, Top: 40, Right: 110, Bottom: 72}, part: &visualassets.ActorPart{ResourceTag: 200, DamageMode: mode}}
		if index == blocker {
			// First-guardian links use the verified consuming callback, whereas
			// the no-op first target deliberately has no such source identity.
			target.firstSegment = 1
		}
		if err := w.bindWorldActor(target); err != nil {
			t.Fatal(err)
		}
		targets[index] = target
		w.Actors = append([]*WorldActor{target}, w.Actors...)
	}
	runtime, err := NewWeaponRuntime(&visualassets.SpriteAtlas{})
	if err != nil {
		t.Fatal(err)
	}
	w.Weapons = runtime
	binding, err := w.reserveWorldActor(36, ActorPoolProjectile, false)
	if err != nil {
		t.Fatal(err)
	}
	binding.Residue.OwnerSlot = NoActorSlot
	runtime.mounts[0].X, runtime.mounts[0].Y = 107, 96
	runtime.projectiles = append(runtime.projectiles, runtimeWeaponProjectile{Binding: binding, Render: WeaponRenderItem{ID: binding.EntityID, Kind: "laser", Active: true}, Laser: LaserBeamState{Length: 0}, Owner: 0})
	return w, targets, binding.Slot
}

func advanceFixtureLaser(t *testing.T, w *World) {
	t.Helper()
	if err := w.Weapons.AdvanceProjectile(w.Weapons.projectiles[0].Render.ID, w.weaponContext(Input{}, false)); err != nil {
		t.Fatal(err)
	}
}

func TestLaserAbsorptionStopsLaterTargetsAndRetiresPhysicalBeam(t *testing.T) {
	for blocker := -1; blocker < 3; blocker++ {
		w, targets, slot := laserBlockerWorld(t, blocker)
		random := w.RandomState()
		advanceFixtureLaser(t, w)
		for index, target := range targets {
			want := 10
			if blocker == -1 && index > 0 || blocker >= 0 && index < blocker {
				want = 7
			}
			if target.Health != want {
				t.Fatalf("blocker%d target%d health%d, want%d", blocker, index, target.Health, want)
			}
		}
		wantTag, wantActive := int16(36), true
		if blocker >= 0 {
			wantTag, wantActive = 4, false
		}
		if w.Weapons.projectiles[0].Render.Active != wantActive || w.Pool.Slot(slot).ResourceTag != wantTag || w.Pool.Slot(slot).list != ActorPoolProjectile {
			t.Fatalf("blocker%d lost source beam retirement: active%v tag%d", blocker, w.Weapons.projectiles[0].Render.Active, w.Pool.Slot(slot).ResourceTag)
		}
		if w.RandomState() != random || w.Score != 0 {
			t.Fatal("nonlethal beam traversal consumed an unrelated reward or random value")
		}
		// World compacts the completed projectile phase before presentation.
		w.Weapons.Compact()
		if blocker >= 0 && len(w.Weapons.RenderState(nil)) != 0 {
			t.Fatal("consumed source laser remains in the rendered projectile state")
		}
	}
}

func TestLaserConsumptionCallbackWorksWithoutLegacyRectangleCallback(t *testing.T) {
	w, _, slot := laserBlockerWorld(t, 0)
	context := w.weaponContext(Input{}, false)
	context.HitRect = nil
	if err := w.Weapons.AdvanceProjectile(w.Weapons.projectiles[0].Render.ID, context); err != nil {
		t.Fatal(err)
	}
	if w.Weapons.projectiles[0].Render.Active || w.Pool.Slot(slot).ResourceTag != 4 {
		t.Fatal("dedicated laser callback depended on the unrelated legacy rectangle callback")
	}
}

func TestLaserCallbackConsumptionUsesVerifiedGuardianPolicies(t *testing.T) {
	w := testWorld(t)
	w.fourthFinalArt = &visualassets.GuardianGroup{Components: []visualassets.GuardianComponent{{DamageBehavior: "final-core-damage"}, {DamageBehavior: "arm-contact"}}}
	area := CollisionRect{Left: 100, Top: 32, Right: 119, Bottom: 51}
	for _, fixture := range []struct {
		actor WorldActor
		want  bool
	}{
		{WorldActor{firstSegment: 1}, true},
		{WorldActor{part: &visualassets.ActorPart{DamageMode: "block-shot"}}, false},
		{WorldActor{thirdFinalMember: &ThirdFinalMember{}, thirdPart: &visualassets.GuardianComponent{DamageBehavior: "block-shot"}}, true},
		{WorldActor{thirdFinalMember: &ThirdFinalMember{}, thirdPart: &visualassets.GuardianComponent{DamageBehavior: "worm-head-health"}}, false},
		{WorldActor{fifthFinal: true, fifthPart: &visualassets.GuardianComponent{DamageBehavior: "barrier-contact"}}, true},
		{WorldActor{fifthFinal: true, fifthPart: &visualassets.GuardianComponent{DamageBehavior: "none"}}, false},
		{WorldActor{fourthIndex: 1, fourthFinal: true}, false},
		{WorldActor{fourthIndex: 2, fourthFinal: true}, true},
	} {
		if got := w.laserCallbackConsumes(&fixture.actor, area); got != fixture.want {
			t.Fatalf("guardian callback absorption%v, want%v for%+v", got, fixture.want, fixture.actor)
		}
	}
	w.FourthMiddle = &FourthMiddleGuardian{}
	w.FourthMiddle.Parts[16].Collision = CollisionRect{Left: 110, Top: 20, Right: 140, Bottom: 68}
	satellite := &WorldActor{fourthIndex: 17}
	if w.laserCallbackConsumes(satellite, area) || !w.laserCallbackConsumes(satellite, CollisionRect{Left: 100, Top: 1, Right: 119, Bottom: 20}) {
		t.Fatal("satellite armor confused an interior hit with the consuming border")
	}
}

func TestLaserSatelliteArmorRejectsDamageAndConsumesBeamBeforeLaterTargets(t *testing.T) {
	w, targets, slot := laserBlockerWorld(t, 0)
	actor := targets[0]
	actor.firstSegment, actor.fourthIndex = 0, 17
	actor.part.ResourceTag, actor.part.DamageMode = 84, "fourth-guardian"
	w.FourthMiddle = &FourthMiddleGuardian{OuterTargets: 5}
	w.FourthMiddle.Parts[16].Health = 20
	w.FourthMiddle.Parts[16].Collision = actor.Collision
	w.fourthMiddleActors[16] = actor
	advanceFixtureLaser(t, w)
	if w.FourthMiddle.Parts[16].Health != 20 || w.FourthMiddle.OuterTargets != 5 || w.Score != 0 || targets[1].Health != 10 || targets[2].Health != 10 {
		t.Fatal("satellite armored callback damaged its plate or later targets")
	}
	if w.Weapons.projectiles[0].Render.Active || w.Pool.Slot(slot).ResourceTag != 4 {
		t.Fatal("satellite armored callback retained the original beam")
	}
}

func TestLaserBlockedTraversalNativeCallbackOptional(t *testing.T) {
	comparisons := 0
	nativeCombatRows(t, "laser-blocker-trace.csv", func(v []int64) {
		w, targets, slot := laserBlockerWorld(t, int(v[0]))
		advanceFixtureLaser(t, w)
		if w.Pool.Slot(slot).ResourceTag != int16(v[1]) {
			t.Fatalf("source beam callback retirement differs%v: tag%d", v, w.Pool.Slot(slot).ResourceTag)
		}
		for index, target := range targets {
			if target.Health != int(v[index+2]) {
				t.Fatalf("source target order differs%v: target%d health%d", v, index, target.Health)
			}
		}
		if w.Score != int(v[5]) || w.RandomState().A != uint32(v[6]) || w.RandomState().B != uint32(v[7]) {
			t.Fatal("source beam traversal changed score or random consumption")
		}
		comparisons++
	})
	if comparisons != 4 {
		t.Fatalf("incomplete source laser traversal comparison: %d", comparisons)
	}
}
