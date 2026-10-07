package engine

import "fmt"

const ActorPoolCapacity = 159
const NoActorSlot = -1

type ActorPoolList uint8

const (
	ActorPoolNone ActorPoolList = iota
	ActorPoolPlayer
	ActorPoolEquipment
	ActorPoolMoving
	ActorPoolProjectile
	ActorPoolScenery
	ActorPoolDormantEquipment
)

// ActorResidue contains named gameplay values that survive slot reuse. It is
// ordinary Go state; initial memory contents and program bytes are not stored.
type ActorResidue struct {
	X, Y                                 int16
	XFraction, YFraction                 uint16
	Counter, Direction                   int16
	HorizontalDriftRemainder             uint16
	VerticalVelocity                     int16
	VerticalFraction                     uint16
	Health, PowerOrScore                 uint16
	WaveBonusToken                       uint16
	StrongHealth                         bool
	MountOffsetX, MountOffsetY           int16
	MotionBudget                         int16
	EmitterClock                         uint16
	OwnerSlot, LeaderSlot, FollowingSlot int
}

// ActorPoolBinding is the small allocation boundary shared by World and weapon
// state. EntityID changes on reuse; Slot remains the physical reference.
type ActorPoolBinding struct {
	Slot, EntityID  int
	AllocationPhase uint8
	Residue         ActorResidue
}

func (r ActorResidue) FireAccumulator() uint8 { return uint8(r.EmitterClock >> 8) }
func (r ActorResidue) FireRate() uint8        { return uint8(r.EmitterClock) }
func (r *ActorResidue) SetFireState(accumulator, rate uint8) {
	r.EmitterClock = uint16(accumulator)<<8 | uint16(rate)
}

// ActorPoolSlot separates a physical reusable slot from its current entity's
// creation ID. References to slots can therefore survive entity replacement.
type ActorPoolSlot struct {
	EntityID                 int
	AllocationPhase          uint8
	ResourceTag              int16
	Linked                   bool
	AuxiliaryFlags           [2]bool
	Residue                  ActorResidue
	list                     ActorPoolList
	previous, next, freeNext int
	allocated                bool
}

type ActorAllocation struct {
	Slot, PreviousEntityID int
	AllocationPhase        uint8
	PreviousList           ActorPoolList
	PreviousTag            int16
	Stolen                 bool
}

// ActorPool reproduces shared capacity, list order and eviction priorities.
// Player/equipment entries use capacity but are never selected for stealing.
type ActorPool struct {
	slots       [ActorPoolCapacity]ActorPoolSlot
	first, last [7]int
	freeFirst   int
}

func NewActorPool() *ActorPool {
	p := &ActorPool{freeFirst: 0}
	for i := range p.first {
		p.first[i], p.last[i] = NoActorSlot, NoActorSlot
	}
	for i := range p.slots {
		p.slots[i].AllocationPhase = [16]uint8{30, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28}[i&15]
		p.slots[i].previous, p.slots[i].next, p.slots[i].freeNext = NoActorSlot, NoActorSlot, i+1
		p.slots[i].Residue.OwnerSlot, p.slots[i].Residue.LeaderSlot, p.slots[i].Residue.FollowingSlot = NoActorSlot, NoActorSlot, NoActorSlot
	}
	p.slots[ActorPoolCapacity-1].freeNext = NoActorSlot
	return p
}

func (p *ActorPool) Slot(index int) *ActorPoolSlot {
	if index < 0 || index >= len(p.slots) {
		return nil
	}
	return &p.slots[index]
}

func (p *ActorPool) First(list ActorPoolList) int {
	if list == ActorPoolNone || int(list) >= len(p.first) {
		return NoActorSlot
	}
	return p.first[list]
}

func (p *ActorPool) Last(list ActorPoolList) int {
	if list == ActorPoolNone || int(list) >= len(p.last) {
		return NoActorSlot
	}
	return p.last[list]
}

func (p *ActorPool) FreeFirst() int { return p.freeFirst }

func (p *ActorPool) Next(index int) int {
	if slot := p.Slot(index); slot != nil {
		return slot.next
	}
	return NoActorSlot
}

// Allocate prefers the free head, then expendable projectile types, any
// projectile, scenery, ordinary moving enemies, and finally the moving head.
// A stolen entry is unlinked without invoking its gameplay removal callback.
func (p *ActorPool) Allocate() (ActorAllocation, error) {
	index := p.freeFirst
	stolen := index == NoActorSlot
	if !stolen {
		p.freeFirst = p.slots[index].freeNext
	} else {
		for candidate := p.first[ActorPoolProjectile]; candidate != NoActorSlot; candidate = p.slots[candidate].next {
			tag := p.slots[candidate].ResourceTag
			if tag == 4 || tag == 12 || tag == 16 {
				index = candidate
				break
			}
		}
		if index == NoActorSlot {
			index = p.first[ActorPoolProjectile]
		}
		if index == NoActorSlot {
			index = p.first[ActorPoolScenery]
		}
		if index == NoActorSlot {
			for candidate := p.first[ActorPoolMoving]; candidate != NoActorSlot; candidate = p.slots[candidate].next {
				tag := p.slots[candidate].ResourceTag
				if tag >= 200 || tag == 4 {
					index = candidate
					break
				}
			}
		}
		if index == NoActorSlot {
			index = p.first[ActorPoolMoving]
		}
	}
	if index == NoActorSlot {
		return ActorAllocation{}, fmt.Errorf("actor pool has no stealable entry")
	}
	node := &p.slots[index]
	result := ActorAllocation{Slot: index, AllocationPhase: node.AllocationPhase, PreviousEntityID: node.EntityID, PreviousList: node.list, PreviousTag: node.ResourceTag, Stolen: stolen}
	if stolen {
		p.unlink(index)
	}
	node.allocated, node.Linked = true, false
	node.AuxiliaryFlags = [2]bool{}
	node.previous, node.next, node.freeNext = NoActorSlot, NoActorSlot, NoActorSlot
	return result, nil
}

func (p *ActorPool) unlink(index int) {
	n := &p.slots[index]
	if n.list == ActorPoolNone {
		return
	}
	if n.previous == NoActorSlot {
		p.first[n.list] = n.next
	} else {
		p.slots[n.previous].next = n.next
	}
	if n.next == NoActorSlot {
		p.last[n.list] = n.previous
	} else {
		p.slots[n.next].previous = n.previous
	}
	n.list, n.previous, n.next = ActorPoolNone, NoActorSlot, NoActorSlot
}

func (p *ActorPool) AttachHead(index int, list ActorPoolList, entityID int, tag int16) error {
	return p.attach(index, list, entityID, tag, NoActorSlot)
}

func (p *ActorPool) AttachTail(index int, list ActorPoolList, entityID int, tag int16) error {
	if list == ActorPoolNone || int(list) >= len(p.last) {
		return fmt.Errorf("invalid actor list")
	}
	return p.attach(index, list, entityID, tag, p.last[list])
}

// AttachAfter preserves descriptor order for linked enemy body components.
func (p *ActorPool) AttachAfter(index int, list ActorPoolList, entityID int, tag int16, predecessor int) error {
	return p.attach(index, list, entityID, tag, predecessor)
}

// Move retains an allocation while changing its list. Nashwan equipment stays
// protected in its saved list until the original actors are restored.
func (p *ActorPool) Move(index int, list ActorPoolList, tail bool) error {
	n := p.Slot(index)
	if n == nil || !n.allocated || list == ActorPoolNone || int(list) >= len(p.first) {
		return fmt.Errorf("invalid actor list move")
	}
	entityID, tag := n.EntityID, n.ResourceTag
	p.unlink(index)
	if tail {
		return p.AttachTail(index, list, entityID, tag)
	}
	return p.AttachHead(index, list, entityID, tag)
}

func (p *ActorPool) attach(index int, list ActorPoolList, entityID int, tag int16, predecessor int) error {
	n := p.Slot(index)
	if n == nil || !n.allocated || n.list != ActorPoolNone || list == ActorPoolNone || int(list) >= len(p.first) {
		return fmt.Errorf("invalid actor attachment")
	}
	if predecessor != NoActorSlot {
		previous := p.Slot(predecessor)
		if previous == nil || previous.list != list {
			return fmt.Errorf("actor predecessor is outside its list")
		}
	}
	next := p.first[list]
	if predecessor != NoActorSlot {
		next = p.slots[predecessor].next
		p.slots[predecessor].next = index
	} else {
		p.first[list] = index
	}
	if next == NoActorSlot {
		p.last[list] = index
	} else {
		p.slots[next].previous = index
	}
	n.list, n.EntityID, n.ResourceTag = list, entityID, tag
	n.previous, n.next = predecessor, next
	return nil
}

// Release pushes the slot onto the free head and clears only its resource type.
// Residue and auxiliary flags are deliberately retained until allocation.
func (p *ActorPool) Release(index int) error {
	n := p.Slot(index)
	if n == nil || !n.allocated {
		return fmt.Errorf("actor slot is not allocated")
	}
	p.unlink(index)
	n.ResourceTag, n.allocated = 0, false
	n.freeNext, p.freeFirst = p.freeFirst, index
	return nil
}

// MarkDead retains list membership until that list reaches its removal phase.
// Same-pass allocations can still select this expendable entry.
func (p *ActorPool) MarkDead(index int) error {
	n := p.Slot(index)
	if n == nil || !n.allocated {
		return fmt.Errorf("actor slot is not allocated")
	}
	n.ResourceTag, n.Linked, n.AuxiliaryFlags[0] = 4, false, false
	return nil
}

func (p *ActorPool) EntityIDs(list ActorPoolList, dst []int) []int {
	for index := p.First(list); index != NoActorSlot; index = p.slots[index].next {
		dst = append(dst, p.slots[index].EntityID)
	}
	return dst
}
