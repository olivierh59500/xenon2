package engine

import (
	"fmt"
	"slices"
	"testing"
)

func originalSupernovaWorld(t testing.TB, level int) *World {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, level))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	return w
}

func supernovaThreat(t testing.TB, w *World) *WorldProjectile {
	t.Helper()
	w.updatePlayerCollision()
	sprite := w.Level.Rules.DefaultEnemyShot
	box, ok := w.shotSpriteBoxes[sprite]
	if !ok {
		t.Fatal("source default enemy projectile collision is absent")
	}
	if box.Width == 0 || box.Height == 0 {
		// Level2's default image has an empty collider. Use its original
		// collidable terrain-shot variant, as advanceTerrainCannon does.
		if len(w.Level.Rules.EnemyShotChoices) < 2 {
			t.Fatal("source collidable enemy projectile variant is absent")
		}
		sprite = w.Level.Rules.EnemyShotChoices[1]
		box = w.shotSpriteBoxes[sprite]
	}
	r := ActorCollisionRect(box, w.Player.X, w.Player.Y)
	// Align the source projectile rectangle with the cached ship rectangle
	// after its native down6+scroll1 update. Level2 has a different image anchor.
	x := w.Player.X + (w.playerCollision.Left+w.playerCollision.Right)/2 - (r.Left+r.Right)/2
	y := w.Player.Y + (w.playerCollision.Top+w.playerCollision.Bottom)/2 - (r.Top+r.Bottom)/2 - 7
	w.spawnEnemyShot(x, y, EnemyShot{Direction: 4, Speed: 6})
	w.Projectiles[0].Sprite = sprite
	return w.Projectiles[0]
}

// Original0x4926 waits31VBLs, then0x49c2 removes tag20 before returning to
// the projectile list. A newer shot already visited before collection may hit.
func TestOriginalSupernovaCollectionSuspendsPhysicalProjectileOrderOptional(t *testing.T) {
	for level := 1; level <= 5; level++ {
		for _, pickupFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("level%d/pickupFirst%v", level, pickupFirst), func(t *testing.T) {
				w := originalSupernovaWorld(t, level)
				if !pickupFirst {
					w.spawnPickup(18, w.Player.X, w.Player.Y)
				}
				shot := supernovaThreat(t, w)
				shotY := shot.Y
				if pickupFirst {
					w.spawnPickup(18, w.Player.X, w.Player.Y)
				}
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
				wantShield := 35
				if pickupFirst {
					wantShield = 39
				}
				if w.Equipment.Shield != wantShield || w.ScreenClearFrames != 31 || !w.stepContinuation.active || shot.Active != pickupFirst || pickupFirst && shot.Y != shotY {
					t.Fatalf("native collection boundary differs: HP%d strobe%d active%v player%+v prefix%+v invulnerable%d drops%d shot%+v box%+v", w.Equipment.Shield, w.ScreenClearFrames, w.stepContinuation.active, w.Player, w.playerCollision, w.InvulnerableFrames, w.PendingExitDrops, shot, w.shotSpriteBoxes[shot.Sprite])
				}
				frame, camera, player, fire := w.Frame, w.ScrollY, w.Player, w.fire
				terrain := slices.Clone(w.RenderTerrainMap)
				actorTerrain := slices.Clone(w.ActorRenderTerrainMap)
				random := w.RandomState()
				for tick := 1; tick <= 30; tick++ {
					if err := w.Step(Input{Fire: true, Motion: MotionInput{Up: true, Right: true}}); err != nil {
						t.Fatal(err)
					}
					w.AdvancePALTick()
					mask := uint16(random.Next())
					if w.Frame != frame || w.ScrollY != camera || w.Player != player || w.fire != fire || !slices.Equal(w.RenderTerrainMap, terrain) || !slices.Equal(w.ActorRenderTerrainMap, actorTerrain) || w.ScreenClearFrames != 31-tick || w.ScreenClearPaletteMask != mask || w.SoundRequests[2] != "synthesized-effect-02" || shot.Active != pickupFirst || pickupFirst && shot.Y != shotY {
						t.Fatalf("flash advanced a blocked pass or lost its sound atPAL%d", tick)
					}
				}
				w.AdvancePALTick()
				if w.Frame != frame || w.stepContinuation.active || w.ScreenClearFrames != 0 || w.ScreenClearPaletteMask != 0 || w.Equipment.Shield != wantShield || shot.Active {
					t.Fatal("PAL31 failed to sweep and finish only the suspended native pass")
				}
				if pickupFirst {
					slot := w.Pool.Slot(shot.Binding.Slot)
					if slot.allocated && slot.EntityID == shot.ID {
						t.Fatal("saved-next cleanup retained the swept older projectile slot")
					}
				}
			})
		}
	}
}

func TestOriginalSupernovaResumesOriginalInputAndFriendlyOnceOptional(t *testing.T) {
	w := originalSupernovaWorld(t, 1)
	if err := w.Step(Input{Fire: true}); err != nil {
		t.Fatal(err)
	}
	var friendly *runtimeWeaponProjectile
	for i := range w.Weapons.projectiles {
		if p := &w.Weapons.projectiles[i]; p.Render.Kind == "small-shot" && p.Render.Active {
			friendly = p
			break
		}
	}
	if friendly == nil {
		t.Fatal("ordinary first trigger did not create its native friendly projectile")
	}
	id, y := friendly.Render.ID, friendly.Small.Y
	w.Equipment.ApplyItem(ItemBitmapShades)
	w.spawnEnemyShot(w.Player.X, w.Player.Y, EnemyShot{Direction: 4, Speed: 6})
	w.spawnPickup(18, w.Player.X, w.Player.Y)
	original := Input{Motion: MotionInput{Down: true}}
	if err := w.Step(original); err != nil {
		t.Fatal(err)
	}
	frame, player, camera := w.Frame, w.Player, w.ScrollY
	if w.Equipment.ShadesFrames != 220 || friendly.Small.Y != y || w.stepContinuation.input != original {
		t.Fatal("collection ran the remaining friendly/timer phase before its wait")
	}
	for range 30 {
		if err := w.Step(Input{Fire: true, Dive: true, Motion: MotionInput{Up: true, Right: true}}); err != nil {
			t.Fatal(err)
		}
		w.AdvancePALTick()
	}
	w.AdvancePALTick()
	var resumed *runtimeWeaponProjectile
	for i := range w.Weapons.projectiles {
		if p := &w.Weapons.projectiles[i]; p.Render.ID == id {
			resumed = p
		}
	}
	if resumed == nil || resumed.Small.Y != y-9 || w.Frame != frame || w.Player != player || w.Equipment.ShadesFrames != 219 || w.fire.Remaining != w.Equipment.FirePeriod || w.previousFire || w.Equipment.Shield != 39 || w.stepContinuation.active {
		t.Fatal("resume repeated movement, lost a friendly shot, or used blocked input/timer ticks")
	}
	if w.PreviousScrollY != camera || w.RenderScrollY != camera {
		t.Fatal("resume replaced the original pass's terrain drawing origin")
	}
}

func TestOriginalSupernovaPendingPassForecastIsolationOptional(t *testing.T) {
	for _, unbound := range []bool{false, true} {
		t.Run(fmt.Sprintf("unbound%v", unbound), func(t *testing.T) {
			w := originalSupernovaWorld(t, 1)
			shots, err := AppendSmallWeaponShots(nil, WeaponSlot{Item: ItemForwardShot, Tier: 1}, 280, 160)
			if err != nil {
				t.Fatal(err)
			}
			binding, err := w.reserveWorldActor(16, ActorPoolProjectile, false)
			if err != nil {
				t.Fatal(err)
			}
			friendly := &WorldSmallShot{ID: binding.EntityID, Binding: binding, Shot: shots[0], Active: true}
			w.SmallShots = append(w.SmallShots, friendly)
			w.poolSmallShots[binding.Slot] = friendly
			w.spawnEnemyShot(w.Player.X, w.Player.Y, EnemyShot{Direction: 4, Speed: 6})
			enemy := w.Projectiles[0]
			w.spawnPickup(18, w.Player.X, w.Player.Y)
			pickup := w.Collectibles[0]
			if unbound {
				for _, b := range []ActorPoolBinding{friendly.Binding, enemy.Binding, pickup.Binding} {
					if err := w.Pool.Release(b.Slot); err != nil {
						t.Fatal(err)
					}
					delete(w.poolBindings, b.EntityID)
					w.clearPoolReferences(b.Slot)
				}
				friendly.Binding, enemy.Binding, pickup.Binding = ActorPoolBinding{}, ActorPoolBinding{}, ActorPoolBinding{}
			}
			if err := w.Step(Input{}); err != nil {
				t.Fatal(err)
			}
			if w.ScreenClearFrames != 31 || w.Equipment.Shield != 39 || !enemy.Active || !w.stepContinuation.active || (w.stepContinuation.unbound != nil) != unbound {
				t.Fatal("fixture did not suspend the expected native/diagnostic phase")
			}
			y := friendly.Shot.Y
			before := forecastDigest(w)
			var f WorldForecast
			if err := f.Load(w); err != nil {
				t.Fatal(err)
			}
			if unbound {
				s := f.State().stepContinuation.unbound
				if s == w.stepContinuation.unbound || s.projectiles[0] == enemy || s.smallShots[0] == friendly || s.projectiles[0] != f.State().Projectiles[0] || s.smallShots[0] != f.State().SmallShots[0] {
					t.Fatal("pending diagnostic lists lost isolation or shared entity identity")
				}
			}
			for range 31 {
				f.AdvancePALTick()
			}
			if forecastDigest(w) != before || f.State().stepContinuation.active || f.State().Equipment.Shield != 39 || f.State().Frame != w.Frame {
				t.Fatal("forecast resume modified the source or failed the suspended pass")
			}
			for range 31 {
				w.AdvancePALTick()
			}
			if friendly.Shot.Y != y-9 || forecastDigest(f.State()) != forecastDigest(w) {
				t.Fatal("live and copied PAL31 continuation differ or lost the friendly callback")
			}
			// Reload must discard the old pending graph before ordinary play.
			if err := f.Load(w); err != nil || f.State().stepContinuation.active || f.State().stepContinuation.unbound != nil {
				t.Fatal("reload retained a stale suspended cursor")
			}
		})
	}
}

func TestOriginalConsecutiveSupernovasResumeOneNativePassOptional(t *testing.T) {
	w := originalSupernovaWorld(t, 1)
	shot := supernovaThreat(t, w)
	w.spawnPickup(18, w.Player.X, w.Player.Y)
	older := w.Collectibles[0]
	w.spawnPickup(18, w.Player.X, w.Player.Y)
	newer := w.Collectibles[0]
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	frame := w.Frame
	if newer.Active || !older.Active || !shot.Active || w.ScreenClearFrames != 31 {
		t.Fatal("first original pickup did not suspend its older successor")
	}
	for range 31 {
		w.AdvancePALTick()
	}
	if older.Active || shot.Active || w.ScreenClearFrames != 31 || !w.stepContinuation.active || w.Frame != frame || w.Equipment.Shield != 39 {
		t.Fatal("first sweep did not resume into the second native blocking callback")
	}
	for range 31 {
		w.AdvancePALTick()
	}
	if w.ScreenClearFrames != 0 || w.stepContinuation.active || w.Frame != frame || w.Equipment.Shield != 39 {
		t.Fatal("second sweep repeated or abandoned its original pass")
	}
}
