package engine

import "testing"

func TestCheckpointRestoresRecordedGunsAndWallet(t *testing.T) {
	w := testWorld(t)
	w.Equipment.ApplyItem(ItemSpeedup)
	w.Equipment.ApplyItem(ItemDoubleShot)
	w.Equipment.Primary.Tier = 2
	w.Equipment.ApplyItem(ItemCannon)
	w.Money = 1500
	w.captureCheckpoint(3200, 80, 3376)
	w.Equipment.ApplyItem(ItemRearShot)
	w.Equipment.ApplyItem(ItemAutofire)
	retainedAdvance := w.Equipment.FireAdvance
	w.Money = 2000
	w.PlayerAlive = false
	w.ScrollY = 3000
	w.RestartCheckpoint()
	if !w.PlayerAlive || w.Player.X != 80 || w.Player.Y != 176 || w.ScrollY != 3200 || w.Money != 1500 {
		t.Fatalf("restart location/wallet differ: %+v", w)
	}
	if w.Equipment.Primary.Item != ItemDoubleShot || w.Equipment.Primary.Tier != 2 || w.Equipment.Mounts[0].Item != ItemCannon || w.Equipment.Rear.Item != ItemNone {
		t.Fatalf("checkpoint guns differ: %+v", w.Equipment)
	}
	if w.Equipment.SpeedTier != 1 || w.Equipment.Shield != 39 || w.Equipment.FireAdvance != retainedAdvance || w.MaximumScrollY != 3200 || w.cursor.FixedHighWater != 3392 {
		t.Fatal("checkpoint admission changed persistent upgrades or restart limits")
	}
}

func TestNashwanDoesNotReplaceCheckpointInventory(t *testing.T) {
	w := testWorld(t)
	w.Equipment.ApplyItem(ItemDoubleShot)
	w.captureCheckpoint(4000, 100, 4176)
	w.Equipment.ApplyItem(ItemSuperNashwan)
	w.Equipment.BeginSuperLoadout()
	w.captureCheckpoint(3500, 160, 3676)
	if w.Checkpoint.Loadout.Primary.Item != ItemDoubleShot || w.Checkpoint.Loadout.Mounts[0].Item != ItemNone {
		t.Fatal("temporary suite must not overwrite the saved checkpoint guns")
	}
	w.RestartCheckpoint()
	if !w.Equipment.SuperLoadoutActive || w.Equipment.SavedLoadout.Primary.Item != ItemDoubleShot || w.Equipment.SuperFrames != 170 {
		t.Fatal("restart must reinstall the temporary suite with the saved prior guns")
	}
}
