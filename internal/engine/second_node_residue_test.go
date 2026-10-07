package engine

import "testing"

func TestOriginalSecondNodeBirthKeepsSourceAllocationAndSpareWordsOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	if w.secondGuardianActor.Binding.Slot != 5 {
		t.Fatal("node initialization changed the original body allocation order")
	}
	for index, actor := range w.secondNodes {
		if actor.Binding.Slot != 8-index {
			t.Fatal("node initialization changed descending native allocation identity")
		}
		s, r := actor.secondNode, actor.Binding.Residue
		if r.X != int16(s.TileX) || r.Y != int16(s.TileY) || r.Counter != 0 || r.VerticalFraction != uint16(index) || r.Health != s.Health || r.EmitterClock != 0 {
			t.Fatalf("source node%d birth state: %+v", index, r)
		}
		// This source-constructor comparison exercises unused retained words;
		// the separate lifecycle test derives its phase through actual updates.
		actor.Binding.Residue = secondResidueFixture()
		want := actor.Binding.Residue
		want.X, want.Y, want.Counter = int16(s.TileX), int16(s.TileY), 0
		want.VerticalFraction, want.Health, want.EmitterClock = uint16(index), s.Health, 0
		w.initializeSecondNodeResidue(actor)
		if got := w.Pool.Slot(actor.Binding.Slot).Residue; got != want {
			t.Fatalf("node constructor replaced spare physical words: got %+v want %+v", got, want)
		}
	}
}

func TestOriginalSecondNodeOpenPhaseSurvivesFlamerReuseOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	w.MaterializationFrames = 0
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2700, 2880, 2880
	w.Player.X = 240
	node := w.secondNodes[0]
	slot, id := node.Binding.Slot, node.ID
	for frame := 1; frame <= 8; frame++ {
		w.Frame = uint64(frame)
		w.advanceSecondNode(node)
		w.finishActorUpdate(node)
	}
	if node.secondNode.Phase != 4 || node.Collision.Empty() {
		t.Fatal("original callbacks did not establish eligible open phase4")
	}
	wantX, wantY := int16(node.secondNode.TileX), int16(node.secondNode.TileY)
	if r := w.Pool.Slot(slot).Residue; r.Counter != 4 || r.X != wantX || r.Y != wantY {
		t.Errorf("native open node retains tile coordinates and phase4: %+v", r)
	}
	// Native damage rendering at0x568c6 changes only presentation; physical
	// tile position and phase remain intact through the lethal hit at0x56944.
	w.damageActor(node, uint16(node.Health))
	if node.Active || w.Pool.Slot(slot).ResourceTag != 4 || !node.Flash {
		t.Fatal("ordinary lethal hit did not publish the source flash and retirement")
	}
	if r := w.Pool.Slot(slot).Residue; r.Counter != 4 || r.X != wantX || r.Y != wantY || r.Health != 0 {
		t.Errorf("node flash changed physical coordinates, phase or health: %+v", r)
	}
	w.releaseDeadPoolEntries(ActorPoolMoving)
	if w.Pool.FreeFirst() != slot {
		t.Fatal("retired node slot was unavailable for ordinary reuse")
	}
	w.Equipment.ApplyItem(ItemFlamer)
	context := w.weaponContext(Input{}, false)
	if err := w.Weapons.SynchronizeEquipment(context); err != nil {
		t.Fatal(err)
	}
	mount := &w.Weapons.mounts[0]
	if mount.Binding.Slot != slot || mount.Binding.EntityID == id {
		t.Fatal("flamer did not replace the actual retired node in its physical slot")
	}
	if mount.FlamerSound.Counter != 4 || !mount.FlamerSound.Started {
		t.Errorf("flamer did not inherit the node's native open phase: %+v", mount.FlamerSound)
	}
	if err := w.Weapons.AdvanceEquipment(context); err != nil {
		t.Fatal(err)
	}
	if !w.StopEffectsRequested || mount.FlamerSound.Counter != 0 {
		t.Fatalf("native release cleanup missing after node reuse: stop%v counter%d", w.StopEffectsRequested, mount.FlamerSound.Counter)
	}
}
