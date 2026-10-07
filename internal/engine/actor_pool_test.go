package engine

import "testing"

func TestActorPoolNativeTraceOptional(t *testing.T) {
	projectiles := [][]int{{}, {}, {40, 16, 12}, {12, 4, 16}, {40, 36}, {}, {}, {}, {}, {}}
	scenery := [][]int{{}, {}, {}, {}, {200}, {110}, {}, {}, {}, {}}
	moving := [][]int{{}, {}, {}, {}, {}, {200}, {100, 184, 200}, {100, 4, 200}, {100, 184, 120}, {40000, 250}}
	var pool *ActorPool
	previousCase := -1
	nativeCombatRows(t, "actor-allocator-trace.csv", func(v []int64) {
		test := int(v[0])
		if test != previousCase {
			pool = NewActorPool()
			for i := range pool.slots {
				n := &pool.slots[i]
				n.Linked, n.AuxiliaryFlags = true, [2]bool{true, true}
				n.Residue = ActorResidue{XFraction: uint16(0x4322 + i), YFraction: uint16(0x1235 + i), Counter: int16(0x124 + i), HorizontalDriftRemainder: uint16(0xab01 + i)}
				n.Residue.SetFireState(uint8(121+i), uint8(31+i))
			}
			if test < 2 {
				pool.slots[test].freeNext = NoActorSlot
			} else {
				pool.freeFirst = NoActorSlot
				index := 0
				for _, list := range []ActorPoolList{ActorPoolProjectile, ActorPoolScenery, ActorPoolMoving} {
					tags := projectiles[test]
					if list == ActorPoolScenery {
						tags = scenery[test]
					}
					if list == ActorPoolMoving {
						tags = moving[test]
					}
					for _, tag := range tags {
						pool.slots[index].allocated = true
						if err := pool.AttachTail(index, list, index+1, int16(tag)); err != nil {
							t.Fatal(err)
						}
						index++
					}
				}
			}
			previousCase = test
		}
		selected := int(v[2]) - 1
		if v[1] == 1 {
			for i := range pool.slots {
				if pool.slots[i].list == ActorPoolProjectile {
					pool.slots[i].list, pool.slots[i].previous, pool.slots[i].next = ActorPoolNone, NoActorSlot, NoActorSlot
				}
			}
			pool.first[ActorPoolProjectile], pool.last[ActorPoolProjectile] = NoActorSlot, NoActorSlot
			if err := pool.AttachHead(selected, ActorPoolProjectile, selected+1, 40); err != nil {
				t.Fatal(err)
			}
			if err := pool.Release(selected); err != nil {
				t.Fatal(err)
			}
		} else {
			allocation, err := pool.Allocate()
			if err != nil {
				t.Fatal(err)
			}
			if allocation.Slot != selected {
				t.Fatalf("allocator %v: selected=%d", v, allocation.Slot+1)
			}
		}
		n := pool.Slot(selected)
		if uint16(n.ResourceTag) != uint16(v[3]) || n.Linked != (v[4] != 0) || n.AuxiliaryFlags[0] != (v[5] != 0) || n.AuxiliaryFlags[1] != (v[6] != 0) || n.Residue.XFraction != uint16(v[7]) || n.Residue.YFraction != uint16(v[8]) || n.Residue.Counter != int16(v[9]) || n.Residue.HorizontalDriftRemainder != uint16(v[10]) || n.Residue.FireAccumulator() != uint8(v[11]) || n.Residue.FireRate() != uint8(v[12]) || pool.FreeFirst()+1 != int(v[13]) || pool.First(ActorPoolProjectile)+1 != int(v[14]) || pool.First(ActorPoolScenery)+1 != int(v[15]) || pool.First(ActorPoolMoving)+1 != int(v[16]) {
			t.Fatalf("allocator %v: slot=%+v free=%d heads=%v", v, n, pool.FreeFirst()+1, pool.first)
		}
	})
}

func TestActorPoolStableSlotIdentityAndRetainedResidue(t *testing.T) {
	pool := NewActorPool()
	allocation, err := pool.Allocate()
	if err != nil {
		t.Fatal(err)
	}
	slot := allocation.Slot
	if err := pool.AttachHead(slot, ActorPoolMoving, 17, 200); err != nil {
		t.Fatal(err)
	}
	n := pool.Slot(slot)
	n.Linked, n.AuxiliaryFlags = true, [2]bool{true, true}
	n.Residue.HorizontalDriftRemainder, n.Residue.Counter = 65500, -9
	if err := pool.Release(slot); err != nil {
		t.Fatal(err)
	}
	if !n.Linked || !n.AuxiliaryFlags[0] || n.Residue.Counter != -9 {
		t.Fatal("release cleared retained state")
	}
	reused, err := pool.Allocate()
	if err != nil {
		t.Fatal(err)
	}
	if reused.Slot != slot || n.Linked || n.AuxiliaryFlags[0] || n.Residue.HorizontalDriftRemainder != 65500 {
		t.Fatal("allocation did not preserve the same physical slot")
	}
	if err := pool.AttachHead(slot, ActorPoolProjectile, 29, 16); err != nil {
		t.Fatal(err)
	}
	if pool.Slot(slot).EntityID != 29 {
		t.Fatal("stable slot did not expose its replacement entity")
	}
}

func TestActorPoolProtectedListsAndLinkedBodyOrder(t *testing.T) {
	pool := NewActorPool()
	for i := range ActorPoolCapacity {
		allocation, err := pool.Allocate()
		if err != nil {
			t.Fatal(err)
		}
		if err := pool.AttachHead(allocation.Slot, ActorPoolEquipment, i+1, 52); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Allocate(); err == nil {
		t.Fatal("pool stole an equipment entry")
	}
	for _, i := range []int{2, 1, 0} {
		if err := pool.Release(i); err != nil {
			t.Fatal(err)
		}
	}
	head := NoActorSlot
	for i := range 3 {
		allocation, err := pool.Allocate()
		if err != nil {
			t.Fatal(err)
		}
		if err := pool.AttachAfter(allocation.Slot, ActorPoolMoving, 500+i, 200, head); err != nil {
			t.Fatal(err)
		}
		head = allocation.Slot
	}
	ids := pool.EntityIDs(ActorPoolMoving, nil)
	if len(ids) != 3 || ids[0] != 500 || ids[1] != 501 || ids[2] != 502 {
		t.Fatalf("linked descriptor order: %v", ids)
	}
}
