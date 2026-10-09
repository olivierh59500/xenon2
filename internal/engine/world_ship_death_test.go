package engine

import (
	"fmt"
	"testing"

	"xenon2/internal/visualassets"
)

func shipDeathWeaponWorld(t *testing.T, items ...Item) *World {
	t.Helper()
	w := testWorld(t)
	w.Level.Encounters = &visualassets.Encounters{}
	var err error
	w.Weapons, err = NewWeaponRuntime(&visualassets.SpriteAtlas{})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.initializeWorldPool(); err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		w.Equipment.ApplyItem(item)
		if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

func TestShipDeathRetiresOnlyNewestMountBeforeOwningPhase(t *testing.T) {
	for _, items := range [][]Item{{ItemCannon, ItemRearShot}, {ItemRearShot, ItemCannon}} {
		t.Run(fmt.Sprint(items), func(t *testing.T) {
			w := shipDeathWeaponWorld(t, items...)
			cannon, rear := w.Weapons.mounts[1], w.Weapons.mounts[5]
			checkpoint, saved, random, free := w.Checkpoint, w.Equipment.SavedLoadout, w.RandomState(), w.Pool.FreeFirst()
			w.destroyPlayer()
			if w.poolError != nil {
				t.Fatal(w.poolError)
			}
			selected, retained := rear.Binding, cannon.Binding
			selectedPosition := 5
			if items[1] == ItemCannon {
				selected, retained, selectedPosition = cannon.Binding, rear.Binding, 1
			}
			if !w.Pool.Slot(selected.Slot).allocated || w.Pool.Slot(selected.Slot).ResourceTag != 4 || w.Equipment.slots()[selectedPosition].Item != ItemNone {
				t.Fatal("death did not retire just the physical head and its catalogue entry")
			}
			if !w.Pool.Slot(retained.Slot).allocated || w.Pool.Slot(retained.Slot).ResourceTag == 4 || w.Equipment.Primary.Item != ItemForwardShot || w.Equipment.Primary.Tier != 0 {
				t.Fatal("death also removed an older weapon or the unmodified basic gun")
			}
			if w.Checkpoint != checkpoint || w.Equipment.SavedLoadout != saved || w.RandomState() != random || w.Pool.FreeFirst() != free {
				t.Fatal("initial death released a slot or changed saved gameplay state")
			}
			if items[1] == ItemCannon {
				support := w.Pool.Slot(cannon.SupportBinding.Slot)
				if !support.allocated || support.ResourceTag != 4 || w.Weapons.mounts[1].SupportActive {
					t.Fatal("selected cannon retained its support callback")
				}
				w.releaseDeadPoolEntries(ActorPoolEquipment)
				if w.Pool.Slot(selected.Slot).allocated || !support.allocated {
					t.Fatal("equipment cleanup released a support owned by another phase")
				}
				if err := w.advancePooledProjectiles(Input{}); err != nil {
					t.Fatal(err)
				}
				if support.allocated {
					t.Fatal("the projectile phase did not release the retired support")
				}
			} else if w.Pool.Slot(cannon.SupportBinding.Slot).ResourceTag != 64 || !w.Weapons.mounts[1].SupportActive {
				t.Fatal("removing the rear gun retired an older cannon's support")
			}
		})
	}
}

func TestShipDeathRestoresBasicBeforeReleasingRemovedPrimary(t *testing.T) {
	w := shipDeathWeaponWorld(t, ItemFlamer)
	w.Checkpoint.Loadout = w.Equipment.WeaponLoadout
	old, free, checkpoint := w.Weapons.mounts[0].Binding, w.Pool.FreeFirst(), w.Checkpoint
	w.destroyPlayer()
	current := w.Weapons.mounts[0].Binding
	if w.poolError != nil || w.Equipment.Primary.Item != ItemForwardShot || w.Equipment.Primary.Tier != 0 || w.Equipment.FirePeriod != 8 || current.Slot != free || current.EntityID == old.EntityID {
		t.Fatal("death did not reconstruct the original basic gun immediately")
	}
	if !w.Pool.Slot(old.Slot).allocated || w.Pool.Slot(old.Slot).ResourceTag != 4 || w.Pool.Slot(current.Slot).ResourceTag != 148 || w.Checkpoint != checkpoint {
		t.Fatal("primary replacement released old storage early or changed its checkpoint")
	}
	w.releaseDeadPoolEntries(ActorPoolEquipment)
	if w.Pool.Slot(old.Slot).allocated || !w.Pool.Slot(current.Slot).allocated {
		t.Fatal("owning-phase cleanup removed the new basic gun")
	}
}

func TestLethalContactConsumesPendingFireWithoutGhostAllocation(t *testing.T) {
	w, enemy := contactDeathWorld(t, 16, true, false, false, false)
	var err error
	w.Weapons, err = NewWeaponRuntime(&visualassets.SpriteAtlas{})
	if err != nil {
		t.Fatal(err)
	}
	w.Actors = nil
	if err := w.initializeWorldPool(); err != nil {
		t.Fatal(err)
	}
	enemy.Binding, enemy.ID = ActorPoolBinding{}, 0
	if err := w.bindWorldActor(enemy); err != nil {
		t.Fatal(err)
	}
	w.Actors = []*WorldActor{enemy}
	w.fire.Pending = true
	before, random := w.nextActorID, w.RandomState()
	if err := w.Step(Input{Fire: true}); err != nil {
		t.Fatal(err)
	}
	if w.PlayerAlive || w.Dive.Phase != 1 || w.Dive.Direction != 0 || w.nextActorID != before || len(w.Weapons.projectiles) != 0 || w.RandomState() != random || w.fire.Pending {
		t.Fatal("lethal contact emitted a ghost basic shot or lost the source death phase")
	}
}

func TestDeadShipDoesNotAdvanceLivingTrailOrResurface(t *testing.T) {
	w, _ := contactDeathWorld(t, 16, true, false, false, false)
	for index := range w.shipTrail {
		w.shipTrail[index] = PlayerMotionState{X: 90 + index*20, Y: 130 + index}
	}
	trail := w.shipTrail
	input := Input{Motion: MotionInput{Up: true, Down: true}}
	if err := w.Step(input); err != nil {
		t.Fatal(err)
	}
	if w.PlayerAlive || w.Dive.Phase != 1 || w.Dive.Direction != 0 || w.shipTrail != trail {
		t.Fatal("death did not freeze the living ship's phase and history")
	}
	w.Dive.Remaining = 1
	for range 2 {
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		if w.PlayerAlive || w.Dive.Phase != 1 || w.Rewind.Timer != 0 || w.shipTrail != trail {
			t.Fatal("death animation executed living movement or resurfacing")
		}
		for _, shadow := range w.Shadows {
			if shadow.Visible {
				t.Fatal("held controls exposed thrust on the dead ship")
			}
		}
	}
	if w.Dive.Remaining != 0 || w.Dive.Direction != -1 {
		t.Fatal("the separate equipment countdown did not expire normally")
	}
}
