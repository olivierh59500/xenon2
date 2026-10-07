package engine

import "testing"

func originalNashwanWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(originalWorldData(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready = false
	// Isolate weapon lifetime from unrelated enemy contact in this short source
	// comparison. This is not a campaign completion or survival assertion.
	w.InvulnerableFrames = 1000
	return w
}

func beginNashwan(t *testing.T, w *World) {
	t.Helper()
	w.Equipment.ApplyItem(ItemSuperNashwan)
	(ShopRules{Level: 1}).Leave(&w.Equipment)
}

func TestNashwanPreservesSavedWeaponActorsAndCursorOptional(t *testing.T) {
	w := originalNashwanWorld(t)
	w.Equipment.ApplyItem(ItemLaser)
	w.Equipment.ApplyItem(ItemCannon)
	w.Equipment.ApplyItem(ItemMineSmall)
	for frame := range 34 {
		if err := w.Step(Input{Fire: true, Motion: MotionInput{Right: frame >= 30}}); err != nil {
			t.Fatal(err)
		}
	}
	original := w.Weapons.mounts
	list := w.Pool.EntityIDs(ActorPoolEquipment, nil)
	if original[1].Laser.Cooldown < 0 || original[5].Mine.Mode != 2 || w.Weapons.MineContext.Count == 0 {
		t.Fatal("ordinary gameplay did not establish a running laser timer and mine cursor")
	}
	context := w.Weapons.MineContext
	beginNashwan(t, w)
	for frame := range 170 {
		if err := w.Step(Input{Motion: MotionInput{Right: frame == 0}}); err != nil {
			t.Fatal(err)
		}
		if !w.PlayerAlive || w.Ready {
			t.Fatal("weapon lifetime comparison lost its live gameplay turn")
		}
		if frame == 169 {
			break
		}
		for index, mount := range original {
			if mount.Binding.EntityID == 0 {
				continue
			}
			slot := w.Pool.Slot(mount.Binding.Slot)
			if !slot.allocated || slot.EntityID != mount.Binding.EntityID || slot.ResourceTag != equipmentResourceTag(mount.Item) || slot.list != ActorPoolDormantEquipment {
				t.Fatalf("pass %d discarded saved weapon %d: %+v", frame, index, slot)
			}
		}
		if w.Weapons.savedMounts[1].Laser != original[1].Laser || w.Weapons.savedMounts[5].Mine != original[5].Mine || w.Weapons.savedMounts[2].Animation != original[2].Animation {
			t.Fatal("the source's detached equipment list must pause its weapon state")
		}
	}
	if w.Equipment.SuperFrames != 0 || w.Weapons.savedMountsActive || w.Pool.First(ActorPoolDormantEquipment) != NoActorSlot {
		t.Fatal("normal 170-pass expiry did not restore the saved equipment list")
	}
	for index, mount := range original {
		if w.Weapons.mounts[index].Binding.EntityID != mount.Binding.EntityID {
			t.Fatalf("restored weapon %d was reconstructed instead of restored", index)
		}
	}
	actual := w.Pool.EntityIDs(ActorPoolEquipment, nil)
	if len(actual) != len(list) {
		t.Fatalf("restored list length %d, want %d", len(actual), len(list))
	}
	for i := range list {
		if actual[i] != list[i] {
			t.Fatalf("restored equipment order %v, want %v", actual, list)
		}
	}
	if w.Weapons.mounts[1].Laser != original[1].Laser || w.Weapons.mounts[5].Mine != original[5].Mine || w.Weapons.MineContext.LastX != context.LastX || w.Weapons.MineContext.LastY != context.LastY {
		t.Fatal("expiry reset the paused timer, mine cursor or planting history")
	}
	if w.Weapons.MineContext.Count != 0 {
		t.Fatal("independent planted mines did not finish while their cursor was saved")
	}
	support := w.Weapons.mounts[2]
	if support.SupportBinding.EntityID != original[2].SupportBinding.EntityID || !support.SupportActive || support.SupportX != w.Player.X+WeaponMountOffset[1].X || support.SupportY != w.Player.Y+WeaponMountOffset[1].Y {
		t.Fatal("saved cannon support did not retain its projectile actor and follow the ship")
	}
	if support.SupportAnimation == original[2].SupportAnimation {
		t.Fatal("saved cannon support stopped advancing with the projectile list")
	}
}

func TestNashwanExpiryLaserReadsReleasedPhysicalOwnerOptional(t *testing.T) {
	w := originalNashwanWorld(t)
	beginNashwan(t, w)
	for frame := range 170 {
		if err := w.Step(Input{Fire: frame == 169}); err != nil {
			t.Fatal(err)
		}
	}
	id, ownerSlot := 0, NoActorSlot
	for _, p := range w.Weapons.projectiles {
		if p.Render.Kind == "laser" && p.Laser.Length == 0 && p.Owner == 3 {
			id, ownerSlot = p.Render.ID, p.Binding.Residue.OwnerSlot
		}
	}
	if id == 0 {
		t.Fatal("the final Nashwan pass did not emit its left laser")
	}
	owner := w.Pool.Slot(ownerSlot)
	saved := owner.Residue
	if owner.allocated || owner.ResourceTag != 0 {
		t.Fatal("source expiry must release temporary equipment in the timer phase")
	}
	if _, ok := w.readWorldResidue(ownerSlot); ok {
		t.Fatal("ordinary entity reads accepted a released allocation")
	}
	if residue, ok := w.readWeaponOwnerResidue(ownerSlot); !ok || residue != saved {
		t.Fatal("the source's physical owner read lost released memory")
	}
	for range 2 {
		if err := w.Step(Input{Motion: MotionInput{Right: true}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range w.Weapons.projectiles {
		if p.Render.ID != id {
			continue
		}
		if w.Pool.Slot(ownerSlot).allocated {
			t.Fatal("the released-owner comparison unexpectedly reused its owner slot")
		}
		if p.Laser.Length != 64 || p.Laser.X != int(saved.X)-7 || p.Laser.Y != int(saved.Y)-24 {
			t.Fatalf("source 0x3fea reads owner (%d,%d): beam %+v", saved.X, saved.Y, p.Laser)
		}
		return
	}
	t.Fatal("expiry removed a laser that still belonged to the projectile list")
}

func TestNashwanExpiryKeepsCompletedWeaponPresentationOptional(t *testing.T) {
	w := originalNashwanWorld(t)
	w.Equipment.ApplyItem(ItemMineSmall)
	beginNashwan(t, w)
	for range 169 {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	oldDrone := w.Weapons.mounts[5].Binding.EntityID
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.Weapons.mounts[5].Item != ItemMineSmall {
		t.Fatal("expiry did not restore ordinary simulation state")
	}
	foundDrone := false
	for _, item := range w.Weapons.RenderState(nil) {
		foundDrone = foundDrone || item.Kind == "attachment" && item.ID == oldDrone
	}
	if !foundDrone {
		t.Fatal("timer expiry changed the already completed Nashwan drawing")
	}
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	for _, item := range w.Weapons.RenderState(nil) {
		if item.Kind == "attachment" && item.ID == oldDrone {
			t.Fatal("the expired suite remained visible on the next gameplay pass")
		}
	}
}

func TestNashwanTurnCleanupReleasesBothWeaponListsOptional(t *testing.T) {
	w := originalNashwanWorld(t)
	w.Equipment.ApplyItem(ItemLaser)
	w.Equipment.ApplyItem(ItemCannon)
	w.Equipment.ApplyItem(ItemMineSmall)
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	w.captureCheckpoint(w.ScrollY, w.Player.X, w.Player.Y)
	beginNashwan(t, w)
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	lastEquipment := w.Pool.Last(ActorPoolDormantEquipment)
	var oldIDs []int
	for _, mounts := range []*[7]runtimeMount{&w.Weapons.mounts, &w.Weapons.savedMounts} {
		for _, m := range mounts {
			oldIDs = append(oldIDs, m.Binding.EntityID, m.SupportBinding.EntityID)
		}
	}
	w.suspendTurn()
	if w.poolError != nil || w.Weapons.savedMountsActive || w.Pool.First(ActorPoolEquipment) != NoActorSlot || w.Pool.First(ActorPoolDormantEquipment) != NoActorSlot {
		t.Fatal("outgoing turn retained temporary or saved equipment allocations")
	}
	if w.Pool.FreeFirst() != lastEquipment {
		t.Fatal("source 0x6d52 must release the equipment tail last onto the free stack")
	}
	for _, id := range oldIDs {
		if _, ok := w.poolBindings[id]; id != 0 && ok {
			t.Fatalf("outgoing turn retained equipment/support identity %d", id)
		}
	}
	allocated := -1
	for range 4 {
		w.RestartCheckpoint()
		if w.poolError != nil || !w.Weapons.savedMountsActive || w.Equipment.SuperFrames != 169 {
			t.Fatal("checkpoint failed to reconstruct saved equipment and the remaining suite")
		}
		count := 0
		for _, slot := range w.Pool.slots {
			if slot.allocated {
				count++
			}
		}
		if count != len(w.poolBindings) || allocated >= 0 && count != allocated {
			t.Fatalf("checkpoint leaked allocations: active %d tracked %d previous %d", count, len(w.poolBindings), allocated)
		}
		allocated = count
	}
}

func TestNashwanDormantEquipmentCannotBeStolen(t *testing.T) {
	pool := NewActorPool()
	original, err := pool.Allocate()
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.AttachHead(original.Slot, ActorPoolEquipment, 1, 32); err != nil {
		t.Fatal(err)
	}
	if err := pool.Move(original.Slot, ActorPoolDormantEquipment, false); err != nil {
		t.Fatal(err)
	}
	for id := 2; id <= ActorPoolCapacity; id++ {
		a, err := pool.Allocate()
		if err != nil {
			t.Fatal(err)
		}
		if err := pool.AttachHead(a.Slot, ActorPoolMoving, id, 200); err != nil {
			t.Fatal(err)
		}
	}
	replacement, err := pool.Allocate()
	if err != nil {
		t.Fatal(err)
	}
	if !replacement.Stolen || replacement.Slot == original.Slot || pool.Slot(original.Slot).EntityID != 1 {
		t.Fatal("full shared pool stole the protected detached equipment list")
	}
}

func TestFreshMineReplacementResetsPlantingAnchorOptional(t *testing.T) {
	w := originalNashwanWorld(t)
	w.Equipment.ApplyItem(ItemMineSmall)
	for range 34 {
		if err := w.Step(Input{Fire: true}); err != nil {
			t.Fatal(err)
		}
	}
	previous := w.Weapons.MineContext
	if previous.LastX == -100 || previous.Count == 0 {
		t.Fatal("ordinary held fire did not plant the initial mine")
	}
	for _, item := range []Item{ItemDrone, ItemMineLarge} {
		w.Equipment.ApplyItem(item)
		if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
			t.Fatal(err)
		}
	}
	if w.Weapons.MineContext.LastX != -100 || w.Weapons.MineContext.LastY != previous.LastY || w.Weapons.MineContext.Count != previous.Count {
		t.Fatal("source 0x51fe resets only the new cursor's horizontal planting anchor")
	}
}

func TestNashwanSavesPendingEquipmentRemovalOptional(t *testing.T) {
	w := originalNashwanWorld(t)
	old := w.Weapons.mounts[0].Binding
	// Shop departure constructs purchased gear before saving the original list.
	// Its replaced basic gun is still part of that list until removal runs.
	w.Equipment.ApplyItem(ItemFlamer)
	beginNashwan(t, w)
	for frame := range 170 {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		slot := w.Pool.Slot(old.Slot)
		list := ActorPoolDormantEquipment
		if frame == 169 {
			list = ActorPoolEquipment
		}
		if !slot.allocated || slot.EntityID != old.EntityID || slot.ResourceTag != 4 || slot.list != list {
			t.Fatalf("pass %d lost the source's saved pending removal: %+v", frame, slot)
		}
	}
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if _, ok := w.poolBindings[old.EntityID]; ok {
		t.Fatal("restored equipment did not run its deferred removal on the next pass")
	}
}
