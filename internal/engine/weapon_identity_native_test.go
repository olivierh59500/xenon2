package engine

import "testing"

// Original constructor records distinguish installed weapons from the separate
// projectiles they emit. The local comparison covers every installed family.
func TestInstalledWeaponIdentitiesMatchOriginalConstructorsOptional(t *testing.T) {
	data := originalWorldData(t, 1)
	seen := make(map[Item]bool)
	nativeCombatRows(t, "weapon-identity-trace.csv", func(v []int64) {
		w, err := NewWorld(data)
		if err != nil {
			t.Fatal(err)
		}
		item, index := Item(v[0]), int(v[1])
		if item != ItemForwardShot {
			w.Equipment.ApplyItem(item)
		}
		if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
			t.Fatal(err)
		}
		if index < 0 || index >= len(w.Weapons.mounts) {
			t.Fatalf("original weapon slot leaves equipment: %v", v)
		}
		mount, equipment := w.Weapons.mounts[index], w.Equipment.slots()[index]
		slot := w.Pool.Slot(mount.Binding.Slot)
		if slot == nil || slot.ResourceTag != int16(v[2]) || slot.list != ActorPoolEquipment || equipment.Item != Item(v[5]) || equipment.Tier != int(v[3]) || equipment.MaxTier != int(v[4]) {
			t.Fatalf("installed weapon differs from original constructor: native%v equipment%+v pool%+v", v, equipment, slot)
		}
		if slot.Residue.WaveBonusToken != uint16(v[5]) || slot.Residue.PowerOrScore != uint16(v[3]) || slot.Residue.Health != uint16(v[4]) {
			t.Fatalf("original installed item/power/maximum fields differ: %v", v)
		}
		seen[item] = true
	})
	if len(seen) != 14 {
		t.Fatalf("incomplete original equipment identity coverage: %d", len(seen))
	}
}
