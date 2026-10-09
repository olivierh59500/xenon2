package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// Constructors and the complete original death callback supply this local
// boundary trace. No owning-list cleanup or artificial release is inserted.
func TestShipDeathEquipmentMatchesOriginalCallbackOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to local original comparisons")
	}
	file, err := os.Open(filepath.Join(root, "ship-death-weapons-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	data := originalWorldData(t, 1)
	// The original common initializer has not admitted the level guardian.
	// Its independent slot would change the exact free-list comparison.
	data.Guardians, data.GuardianGroups, data.GuardianParts = nil, nil, nil
	comparisons := 0
	for _, row := range rows[1:] {
		if len(row) != 49 {
			t.Fatalf("native death record has %d fields, want 49", len(row))
		}
		var value [49]int
		for index, text := range row {
			if index == 13 {
				continue
			}
			value[index], err = strconv.Atoi(text)
			if err != nil {
				t.Fatal(err)
			}
		}
		w, err := NewWorld(data)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range value[3:6] {
			if item == 0 || Item(item) == ItemForwardShot {
				continue
			}
			w.Equipment.ApplyItem(Item(item))
			if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
				t.Fatal(err)
			}
		}
		if value[5] == int(ItemSuperNashwan) {
			w.Equipment.BeginSuperLoadout()
			if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
				t.Fatal(err)
			}
		}
		if value[2] != 0 {
			w.Equipment.ApplyItem(ItemPowerup)
			if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
				t.Fatal(err)
			}
		}
		var previous [7]ActorPoolBinding
		for index, mount := range w.Weapons.mounts {
			previous[index] = mount.Binding
		}
		checkpoint, saved, random := w.Checkpoint, w.Equipment.SavedLoadout, w.RandomState()
		w.PendingExitDrops = value[1]
		w.destroyPlayer()
		if w.poolError != nil {
			t.Fatal(w.poolError)
		}
		got := [7]int{0, w.Dive.Phase, w.Dive.Direction, w.Equipment.Shield, w.Equipment.FirePeriod, w.Pool.FreeFirst(), w.Pool.First(ActorPoolEquipment)}
		if !w.PlayerAlive {
			got[0] = 1
		}
		want := [7]int{value[6], value[7], value[8], value[9], value[10], value[11], value[12]}
		if got != want {
			t.Fatalf("case%d initial death differs: got%v want%v", value[0], got, want)
		}
		hash := uint64(0xcbf29ce484222325)
		for index := range ActorPoolCapacity {
			hash ^= uint64(uint16(w.Pool.Slot(index).ResourceTag))
			hash *= 0x100000001b3
		}
		wantHash, err := strconv.ParseUint(row[13], 10, 64)
		if err != nil || hash != wantHash {
			t.Fatalf("case%d physical type history differs: got%d want%d error%v", value[0], hash, wantHash, err)
		}
		for index, slot := range w.Equipment.slots() {
			owner := NoActorSlot
			if binding := w.Weapons.mounts[index].Binding; binding.EntityID != 0 {
				owner = binding.Slot
			}
			got := [4]int{int(slot.Item), slot.Tier, slot.MaxTier, owner}
			at := 14 + index*4
			want := [4]int{value[at], value[at+1], value[at+2], value[at+3]}
			if got != want {
				t.Fatalf("case%d equipment%d differs: got%v want%v", value[0], index, got, want)
			}
			tag := -1
			if binding := previous[index]; binding.EntityID != 0 {
				tag = int(w.Pool.Slot(binding.Slot).ResourceTag)
			}
			if tag != value[42+index] {
				t.Fatalf("case%d previous equipment%d retired differently: got%d want%d", value[0], index, tag, value[42+index])
			}
		}
		if w.Checkpoint != checkpoint || w.Equipment.SavedLoadout != saved || w.RandomState() != random {
			t.Fatal("initial death altered checkpoint loadout, dormant equipment or randomness")
		}
		comparisons++
	}
	if comparisons != 72 {
		t.Fatalf("incomplete initial death comparison: %d", comparisons)
	}
	t.Logf("%d complete original ship-death/equipment boundaries", comparisons)
}
