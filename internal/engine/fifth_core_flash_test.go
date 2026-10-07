package engine

import (
	"fmt"
	"slices"
	"testing"

	"xenon2/internal/visualassets"
)

func fifthCoreFlashWorld(t *testing.T, final, exposed bool) (*World, *WorldActor, *WorldActor) {
	t.Helper()
	w := fifthResourceWorld(t)
	w.Level.Encounters = &visualassets.Encounters{}
	w.ScrollY = 2240
	if err := w.activateFifthGuardian(visualassets.FixedEncounter{Y: 2336}, final); err != nil {
		t.Fatal(err)
	}
	if final {
		// The source body enters with its lower weak point on screen.
		w.ScrollY, w.ScrollDelta = 256, 160
		w.advanceFifthGuardian(true)
		if exposed {
			for index := 3; index < 21; index++ {
				w.damageFifthGuardian(w.fifthFinalActors[index], 255)
			}
		}
	} else {
		w.ScrollDelta = 0
	}
	w.ScrollDelta = 0
	w.advanceFifthGuardian(final)
	w.Player = PlayerMotionState{X: 24, Y: 176}
	w.PreviousPlayer = w.Player
	w.MaterializationFrames = 0
	if final {
		return w, w.fifthFinalActors[0], w.fifthFinalActors[21]
	}
	return w, w.fifthMiddleActors[0], w.fifthMiddleActors[5]
}

func addFifthFlashShot(t *testing.T, w *World, cannon bool, x, y int, damage uint16) int {
	t.Helper()
	if cannon {
		w.Weapons.context = w.weaponContext(Input{}, false)
		animation := w.Weapons.animations["cannon-ball"]
		box := w.Weapons.boxes[animation.Animation.Frames[0].Sprite]
		x, y = x-box.X, y-box.Y
		shot := w.Weapons.add("cannon-ball", "cannon-ball", x, y+10, 0)
		if shot == nil || shot.Binding.EntityID == 0 {
			t.Fatal("source cannonball did not receive a physical projectile slot")
		}
		shot.Cannon = CannonBallMotion{X: x, Y: y + 10}
		return shot.Binding.Slot
	}
	binding, err := w.reserveWorldActor(16, ActorPoolProjectile, false)
	if err != nil {
		t.Fatal(err)
	}
	shot := &WorldSmallShot{ID: binding.EntityID, Binding: binding, Active: true,
		Shot: SmallShot{X: x, Y: y + 9, VelocityY: -9, Damage: damage}}
	w.poolSmallShots[binding.Slot] = shot
	w.SmallShots = append(w.SmallShots, shot)
	return binding.Slot
}

// Native core callbacks select the owner's tile flash renderer at 0x5652a
// and 0x562b2. Those renderers restore normal drawing at 0x56840 and 0x55d1c.
func TestOriginalFifthCoreHitsPublishOwnerTileFlashOptional(t *testing.T) {
	for _, final := range []bool{false, true} {
		for _, cannon := range []bool{false, true} {
			t.Run(fmt.Sprintf("final_%v/cannon_%v", final, cannon), func(t *testing.T) {
				w, body, core := fifthCoreFlashWorld(t, final, true)
				damage := uint16(1)
				if cannon {
					damage = CannonBallDamage
				}
				beforeHealth := core.Health
				if final {
					beforeHealth = int(w.FifthFinal.CoreHealth)
				}
				score, random := w.Score, w.RandomState()
				slot := addFifthFlashShot(t, w, cannon, int(core.X)+3, int(core.Y)+3, damage)
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
				health := core.Health
				if final {
					health = int(w.FifthFinal.CoreHealth)
				}
				if health != beforeHealth-int(damage) || w.Score != score || w.RandomState() != random || w.Pool.Slot(slot).ResourceTag != 4 {
					t.Fatalf("real core hit changed native damage, score, random or retirement: health %d -> %d, tag %d", beforeHealth, health, w.Pool.Slot(slot).ResourceTag)
				}
				if !body.Flash || body.Patch == nil || core.Flash {
					t.Fatalf("core hit did not select its owner's tile renderer: body flash %v patch %v, core flash %v", body.Flash, body.Patch != nil, core.Flash)
				}
				if final {
					want := w.fifthFinalArt.Components[21].HeadingFrames[health*11/w.fifthFinalArt.MotionParameters["core_health"]]
					if core.Sprite != want || body.Health != health {
						t.Fatalf("native core callback did not publish its immediate image/body health: sprite %s want %s, body health %d", core.Sprite, want, body.Health)
					}
				}
				for _, actor := range w.Actors {
					if actor.Active && actor.fifthIndex > 0 && actor != body && actor.Flash {
						t.Fatalf("core hit flashed unrelated part %d", actor.fifthIndex-1)
					}
				}
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
				if body.Flash || final && body.Patch != nil || !final && body.Patch == nil {
					t.Fatal("source body renderer did not resume normal drawing on the following pass")
				}
			})
		}
	}
}

func TestOriginalFifthNoOpContactsDoNotSelectFlashOptional(t *testing.T) {
	for _, family := range []string{"middle-corner", "final-band", "closed-core"} {
		t.Run(family, func(t *testing.T) {
			w, body, core := fifthCoreFlashWorld(t, family != "middle-corner", false)
			target := core
			if family == "middle-corner" {
				target = w.fifthMiddleActors[6]
			} else if family == "final-band" {
				target = w.fifthFinalActors[1]
			}
			health, score, random := target.Health, w.Score, w.RandomState()
			x, y := int(target.X)+3, int(target.Y)+3
			if family != "closed-core" {
				x = (target.Collision.Left + target.Collision.Right) / 2
				y = (target.Collision.Top + target.Collision.Bottom) / 2
			}
			slot := addFifthFlashShot(t, w, false, x, y, 1)
			if err := w.Step(Input{}); err != nil {
				t.Fatal(err)
			}
			if target.Health != health || w.Score != score || w.RandomState() != random || body.Flash || target.Flash {
				t.Fatal("source no-op or unavailable collider changed damage state or selected a flash renderer")
			}
			retired := w.Pool.Slot(slot).ResourceTag == 4
			if retired != (family == "final-band") {
				t.Fatalf("closed core eligibility or no-op shot consumption changed: retired %v, point %d/%d, bounds %+v", retired, x, y, target.Collision)
			}
		})
	}
}

func TestOriginalFifthMiddleCoreContactKeepsCurrentPassFlashOptional(t *testing.T) {
	w, body, core := fifthCoreFlashWorld(t, false, true)
	// The original ship prefix reaches the narrow core without touching either
	// adjacent mount. Contact runs before the moving-actor callbacks.
	w.Player = PlayerMotionState{X: int(core.X) + 3, Y: int(core.Y) + 3}
	w.PreviousPlayer = w.Player
	random := w.RandomState()
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if core.Health != 73 || w.Equipment.Shield != 31 || !w.PlayerAlive || !body.Flash || core.Flash || w.Score != 0 || w.RandomState() != random {
		t.Fatalf("source player contact lost core damage or same-pass body renderer: health %d shield %d body flash %v core flash %v", core.Health, w.Equipment.Shield, body.Flash, core.Flash)
	}
	w.Player = PlayerMotionState{X: 24, Y: 176}
	w.PreviousPlayer = w.Player
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if body.Flash || core.Health != 73 {
		t.Fatal("source body renderer did not clear after the contact pass")
	}
}

func TestOriginalFifthSupernovaKeepsNoOpRenderersOptional(t *testing.T) {
	w, body, core := fifthCoreFlashWorld(t, false, true)
	random := w.RandomState()
	w.finishSupernova()
	if core.Health != 73 || !body.Flash || core.Flash || w.Score != 800 || w.RandomState() != random {
		t.Fatal("source screen-clear traversal lost its damage callbacks or owner tile renderer")
	}
	for index := 1; index < len(w.fifthMiddleActors); index++ {
		actor := w.fifthMiddleActors[index]
		if actor.Flash != (index <= 4) {
			t.Fatalf("screen-clear callback selected the wrong individual renderer for part %d", index)
		}
	}
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	score := w.Score
	// Retained wrecks use the source no-op callback after their destruction.
	w.damageFifthGuardian(w.fifthMiddleActors[1], 1)
	if w.fifthMiddleActors[1].Flash || w.Score != score || body.Flash {
		t.Fatal("the retained wreck's no-op callback selected another flash or reward")
	}
}

func TestOriginalFifthLethalMountHitRetainsIndividualFlashOptional(t *testing.T) {
	for _, final := range []bool{false, true} {
		t.Run(fmt.Sprint(final), func(t *testing.T) {
			w, body, _ := fifthCoreFlashWorld(t, final, false)
			mount := w.fifthMiddleActors[1]
			if final {
				mount = w.fifthFinalActors[9]
			}
			w.damageFifthGuardian(mount, uint16(mount.Health-1))
			w.advanceFifthGuardian(final)
			oldSprite, oldSlot, score, random := mount.Sprite, mount.Binding.Slot, w.Score, w.RandomState()
			addFifthFlashShot(t, w, false, mount.Collision.Left+3, mount.Collision.Top+3, 1)
			if err := w.Step(Input{}); err != nil {
				t.Fatal(err)
			}
			if !mount.Active || !mount.Flash || body.Flash || mount.Sprite != oldSprite || mount.Binding.Slot != oldSlot || w.Pool.Slot(oldSlot).ResourceTag != 276 || w.Score != score+200 || w.RandomState() != random {
				t.Fatal("lethal mount hit lost its source one-frame individual renderer, retained slot or reward")
			}
			if err := w.Step(Input{}); err != nil {
				t.Fatal(err)
			}
			if mount.Flash || mount.Sprite != mount.fifthPart.DestroyedSprite || !mount.Collision.Empty() {
				t.Fatal("following source callback did not publish the ordinary noncollidable wreck")
			}
		})
	}
}

// The normal renderer at 0x5685a writes six muzzle tile words from the
// clock-selected table. Its replacement at 0x56840 draws that table without
// those writes, so consecutive flashes retain the last normal muzzle pose.
func TestOriginalFifthMiddleFlashRetainsNormalMuzzleTilesOptional(t *testing.T) {
	w, body, core := fifthCoreFlashWorld(t, false, true)
	w.FifthMiddle.Parts[0].FireAccumulator = 246 // The native +10 starts clock 1.
	wantHealth := 200
	for _, pass := range []struct {
		hit         bool
		clock, pose int
	}{{true, 1, 0}, {true, 2, 0}, {false, 3, 3}, {true, 4, 3}, {false, 5, 5}} {
		if pass.hit {
			addFifthFlashShot(t, w, false, int(core.X)+3, int(core.Y)+3, 1)
			wantHealth--
		}
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		want := w.fifthMiddleArt.Components[0].TileFrames[pass.pose]
		if w.FifthMiddle.Parts[0].Clock != pass.clock || body.Patch == nil || !slices.Equal(body.Patch.Tiles, want.Tiles) || body.Flash != pass.hit || core.Health != wantHealth || w.Score != 0 {
			t.Fatalf("clock %d: flash %v must use last normal pose %d, got clock %d flash %v health %d", pass.clock, pass.hit, pass.pose, w.FifthMiddle.Parts[0].Clock, body.Flash, core.Health)
		}
	}
}

func TestOriginalFifthMiddleContactFlashRetainsPreUpdateMuzzleTilesOptional(t *testing.T) {
	w, body, core := fifthCoreFlashWorld(t, false, true)
	w.FifthMiddle.Parts[0].FireAccumulator = 246
	w.Player = PlayerMotionState{X: int(core.X) + 3, Y: int(core.Y) + 3}
	w.PreviousPlayer = w.Player
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.FifthMiddle.Parts[0].Clock != 1 || core.Health != 73 || !body.Flash || !slices.Equal(body.Patch.Tiles, w.fifthMiddleArt.Components[0].TileFrames[0].Tiles) {
		t.Fatal("contact before the updater advanced the normal muzzle table during a flash")
	}
	w.Player = PlayerMotionState{X: 24, Y: 176}
	w.PreviousPlayer = w.Player
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if body.Flash || !slices.Equal(body.Patch.Tiles, w.fifthMiddleArt.Components[0].TileFrames[2].Tiles) {
		t.Fatal("normal muzzle selection did not resume after the contact flash")
	}
}

func TestOriginalFifthMiddleFlashRetainsMuzzleAcrossClockResetOptional(t *testing.T) {
	w, body, core := fifthCoreFlashWorld(t, false, true)
	w.FifthMiddle.Parts[0].Clock = 7
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(body.Patch.Tiles, w.fifthMiddleArt.Components[0].TileFrames[8].Tiles) {
		t.Fatal("fixture did not run the normal clock-eight renderer")
	}
	addFifthFlashShot(t, w, false, int(core.X)+3, int(core.Y)+3, 1)
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.FifthMiddle.Parts[0].Clock != 0 || core.Health != 199 || !body.Flash || !slices.Equal(body.Patch.Tiles, w.fifthMiddleArt.Components[0].TileFrames[8].Tiles) {
		t.Fatal("flash rewrote the last normal muzzle table when the source clock reset")
	}
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if body.Flash || !slices.Equal(body.Patch.Tiles, w.fifthMiddleArt.Components[0].TileFrames[0].Tiles) {
		t.Fatal("normal clock-zero muzzle table did not return after the flash")
	}
}

func TestOriginalFifthMiddleContactAfterNormalOrFlashRetainsMuzzleTilesOptional(t *testing.T) {
	for _, priorFlash := range []bool{false, true} {
		t.Run(fmt.Sprint(priorFlash), func(t *testing.T) {
			w, body, core := fifthCoreFlashWorld(t, false, true)
			w.FifthMiddle.Parts[0].FireAccumulator = 246
			wantHealth, wantPose := 73, 1
			if priorFlash {
				addFifthFlashShot(t, w, false, int(core.X)+3, int(core.Y)+3, 1)
				wantHealth, wantPose = 72, 0
			}
			if err := w.Step(Input{}); err != nil {
				t.Fatal(err)
			}
			w.Player = PlayerMotionState{X: int(core.X) + 3, Y: int(core.Y) + 3}
			w.PreviousPlayer = w.Player
			if err := w.Step(Input{}); err != nil {
				t.Fatal(err)
			}
			if core.Health != wantHealth || w.Equipment.Shield != 31 || !body.Flash || !slices.Equal(body.Patch.Tiles, w.fifthMiddleArt.Components[0].TileFrames[wantPose].Tiles) {
				t.Fatalf("contact after prior flash %v must retain normal pose %d and source health %d", priorFlash, wantPose, wantHealth)
			}
		})
	}
}

func TestOriginalFifthMiddleConsecutiveContactsDoNotCommitSkippedMuzzlePoseOptional(t *testing.T) {
	w, body, core := fifthCoreFlashWorld(t, false, true)
	w.FifthMiddle.Parts[0].FireAccumulator = 246
	for range 2 {
		w.Player = PlayerMotionState{X: int(core.X) + 3, Y: int(core.Y) + 3}
		w.PreviousPlayer = w.Player
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(body.Patch.Tiles, w.fifthMiddleArt.Components[0].TileFrames[0].Tiles) {
			t.Fatal("consecutive contact committed a muzzle pose skipped by the flash renderer")
		}
	}
	// The source 200 HP core necessarily dies on its second 127-point contact.
	if !w.FifthMiddle.Defeated || body.Active || core.Active || w.Score != 1500 || w.PendingExitDrops != 10 || w.Equipment.Shield != 23 {
		t.Fatal("consecutive contacts changed the source core death, reward or surviving ship")
	}
}
