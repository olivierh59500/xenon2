package engine

import (
	"fmt"
	"testing"
)

func fifthFinalPointScene(t *testing.T, bodyY int) *World {
	t.Helper()
	w := originalGuardianTargetWorld(t, 5, true)
	// Replay the original controller into a particular reachable body pose.
	// Its earlier events are outside this isolated point-callback fixture;
	// replaying births without their projectile phases would accumulate shots.
	for int(w.FifthFinal.Parts[0].Y) < bodyY-2 {
		w.Frame++
		w.ScrollY--
		w.FifthFinal.Advance(w.fifthFinalArt, w.ScrollY, w.ScrollDelta, w.BaseScrollStep, w.MaximumScrollY,
			int(w.Frame), w.Player.X, w.Player.Y, &w.random)
	}
	// Publish two consecutive source poses, retaining the actual previous
	// position for the earlier linear opportunity implementation as well.
	for pass := 0; pass < 2; pass++ {
		w.Frame++
		w.ScrollY--
		w.advanceFifthGuardian(true)
	}
	return w
}

func TestGuardianFirstPointImpactMatchesOrdinaryBulletCallbacksOptional(t *testing.T) {
	for _, bodyY := range []int{-16, 0} {
		w := fifthFinalPointScene(t, bodyY)
		band := w.fifthFinalActors[2]
		x, y := (band.Collision.Left+band.Collision.Right)/2, (band.Collision.Top+band.Collision.Bottom)/2
		before := forecastDigest(w)
		hit, useful := presentationFirstPointImpact(w, x, y)
		if !hit || useful || forecastDigest(w) != before {
			t.Fatal("invisible native band was missed, targeted or mutated")
		}
		state := *w.FifthFinal
		bullet := WorldSmallShot{Active: true, Shot: SmallShot{X: x, Y: y + 9, VelocityY: -9, Damage: 1}}
		w.advanceSmallShot(&bullet)
		if bullet.Active || *w.FifthFinal != state {
			t.Fatal("source ordinary bullet did not stop harmlessly at the armor band")
		}

		w = fifthFinalPointScene(t, bodyY)
		mount := w.fifthFinalActors[3]
		x, y = (mount.Collision.Left+mount.Collision.Right)/2, (mount.Collision.Top+mount.Collision.Bottom)/2
		hit, useful = presentationFirstPointImpact(w, x, y)
		if !hit || !useful {
			t.Fatal("clear source mount strip was rejected")
		}
		health := w.FifthFinal.Parts[3].Health
		bullet = WorldSmallShot{Active: true, Shot: SmallShot{X: x, Y: y + 9, VelocityY: -9, Damage: 1}}
		w.advanceSmallShot(&bullet)
		if bullet.Active || w.FifthFinal.Parts[3].Health != health-1 {
			t.Fatal("source ordinary bullet failed to damage the first eligible mount")
		}
	}
}

func TestGuardianObservedFirstImpactPrecedesOlderProjectileDamageOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	if err := w.activateFourthGuardian(false); err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	// Replay an early pose covered by the original 24,000-state native trace.
	for frame := 0; frame <= 5; frame++ {
		w.Frame, w.ScrollY, w.ScrollDelta = uint64(frame), 2480-frame, 1
		if frame == 0 {
			w.ScrollDelta = 0
		}
		w.Player.X, w.Player.Y = 30+3*frame, 20+7*frame
		for _, actor := range w.fourthMiddleActors {
			if err := w.advanceFourthPart(actor); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, index := range []int{15, 16, 17, 19} {
		actor := w.fourthMiddleActors[index]
		w.damageFourthGuardian(actor, targetPoint(actor.Collision), w.FourthMiddle.Parts[index].Health)
	}
	last := w.fourthMiddleActors[18]
	w.damageFourthGuardian(last, targetPoint(last.Collision), w.FourthMiddle.Parts[18].Health-1)
	if w.FourthMiddle.OuterTargets != 1 || w.FourthMiddle.Parts[18].Health != 1 {
		t.Fatal("ordinary damage callbacks did not leave the last satellite")
	}
	// An ordinary earlier primary emission misses once and reaches the satellite
	// after the next pass's new primary has already struck the locked core.
	w.Player.X, w.Player.Y = 200, 25
	if err := w.Weapons.AdvanceEquipment(w.weaponContext(Input{Fire: true}, true)); err != nil {
		t.Fatal(err)
	}
	if err := w.advancePooledProjectiles(Input{}); err != nil {
		t.Fatal(err)
	}
	oldID := w.nextActorID
	w.Player.X, w.Player.Y = 135, 64
	before := forecastIsolationDigest(w)
	var forecast WorldForecast
	if err := forecast.Load(w); err != nil {
		t.Fatal(err)
	}
	var events []WeaponPointImpact
	var useful []bool
	if _, err := forecast.AdvanceObserved(Input{Fire: true}, func(event WeaponPointImpact) {
		events = append(events, event)
		_, target := presentationFirstPointImpact(forecast.State(), event.X, event.Y)
		useful = append(useful, target)
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].ProjectileID <= oldID || events[1].ProjectileID != oldID ||
		events[0].X != 135 || events[0].Y != 49 || events[1].X != 200 || events[1].Y != 1 || useful[0] || !useful[1] {
		t.Fatalf("native newest-first pre-hit order: events%+v useful%v", events, useful)
	}
	state := forecast.State()
	if state.FourthMiddle.OuterTargets != 0 || state.FourthMiddle.Parts[4].Health != w.FourthMiddle.Parts[4].Health {
		t.Fatal("old shot did not expose the core after the new shot was harmlessly consumed")
	}
	if _, target := presentationTargetBounds(state, state.fourthMiddleActors[4]); !target {
		t.Fatal("fixture does not distinguish the misleading post-projectile target snapshot")
	}
	if opportunity, supported := presentationGuardianShotOpportunity(w, MotionInput{}, &forecast, 3); !supported || opportunity {
		t.Fatal("old projectile's later damage was attributed to the absorbed new primary")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("observing or aiming changed the live world")
	}
}

func TestGuardianPrimaryAimUsesNativeClearAndBlockedStripsOptional(t *testing.T) {
	for _, bodyY := range []int{-16, 0} {
		for _, index := range []int{2, 3} {
			t.Run(fmt.Sprintf("bodyY%d/part%d", bodyY, index), func(t *testing.T) {
				w := fifthFinalPointScene(t, bodyY)
				w.Ready, w.MaterializationFrames = false, 0
				actor := w.fifthFinalActors[index]
				mount := w.fifthFinalActors[3]
				w.Player.X, w.Player.Y = (mount.Collision.Left+mount.Collision.Right)/2, actor.Collision.Bottom+15
				if index == 3 {
					// The clear firing strip lies above the lower armor band. This
					// point-query fixture permits ordinary band contact damage;
					// it does not assert that the ship's placement is a safe route.
				}
				before := forecastIsolationDigest(w)
				var forecast WorldForecast
				opportunity, supported := presentationGuardianShotOpportunity(w, MotionInput{}, &forecast, 3)
				if !supported || opportunity != (index == 3) {
					t.Fatalf("part%d: opportunity%v supported%v bounds%+v ship%+v", index, opportunity, supported, actor.Collision, w.Player)
				}
				if presentationShotOpportunityForMotion(w, MotionInput{}) != opportunity {
					t.Fatal("presentation fire did not use the native first-impact opportunity")
				}
				pilot := PresentationPilot{}
				if pilot.selectiveFireForMotion(w, MotionInput{}) != opportunity {
					t.Fatal("runtime presentation fire did not use the first-impact opportunity")
				}
				if index == 3 && forecast.State().FifthFinal.Parts[3].Health != w.FifthFinal.Parts[3].Health-1 {
					t.Fatal("accepted primary did not cause native mount damage")
				}
				if index == 2 && forecast.State().FifthFinal.Parts[3].Health != w.FifthFinal.Parts[3].Health {
					t.Fatal("armor band failed to consume the primary ahead of the mount")
				}
				if index == 2 {
					var oracle WorldForecast
					if err := oracle.Load(w); err != nil {
						t.Fatal(err)
					}
					firstCollider := 0
					for pass := 0; pass < 18 && firstCollider == 0; pass++ {
						for tick := 0; tick < 3; tick++ {
							oracle.AdvancePALTick()
						}
						if _, err := oracle.AdvanceObserved(Input{Fire: true}, func(event WeaponPointImpact) {
							if event.Kind != "small-shot" || event.OwnerSlot != 0 || event.ProjectileID <= w.nextActorID || firstCollider != 0 {
								return
							}
							var ordered [ActorPoolCapacity]*WorldActor
							for _, candidate := range oracle.State().orderedMovingActors(&ordered) {
								if candidate.Active && candidate.ActorList == "moving" && candidate.Collision.Contains(event.X, event.Y) {
									firstCollider = candidate.ID
									break
								}
							}
						}); err != nil {
							t.Fatal(err)
						}
					}
					if firstCollider != actor.ID {
						t.Fatalf("ordinary primary first hit actor%d, expected armor band%d", firstCollider, actor.ID)
					}
				}
				if forecastIsolationDigest(w) != before {
					t.Fatal("first-impact aiming changed live source state")
				}
			})
		}
	}
}
