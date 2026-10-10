package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func pooledTwoPlayerSession(t *testing.T) *Session {
	t.Helper()
	data := testWorld(t).Level
	data.Common = &visualassets.SpriteAtlas{}
	s, err := NewSession(data, 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestTwoPlayerPoolHasOneCapacityAndProtectsInactiveLists(t *testing.T) {
	s := pooledTwoPlayerSession(t)
	active, other := s.Players[0], s.Players[1]
	if active.Pool.storage() != other.Pool.storage() || active.Pool.First(ActorPoolPlayer) == other.Pool.First(ActorPoolPlayer) {
		t.Fatal("saved games need separate list heads in one physical pool")
	}
	var protected [5]ActorPoolSlot
	for i, binding := range other.poolShadows {
		protected[i] = *other.Pool.Slot(binding.Slot)
	}
	protected[4] = *other.Pool.Slot(other.Weapons.mounts[0].Binding.Slot)
	allocations := 0
	for active.Pool.FreeFirst() != NoActorSlot {
		if _, err := active.reserveWorldActor(200, ActorPoolMoving, false); err != nil {
			t.Fatal(err)
		}
		allocations++
	}
	if allocations != 149 || other.Pool.FreeFirst() != NoActorSlot {
		t.Fatalf("two saved games exceeded 159 combined objects: available%d otherFree%d", allocations, other.Pool.FreeFirst())
	}
	victim := active.Pool.First(ActorPoolMoving)
	replacement, err := active.reserveWorldActor(12, ActorPoolProjectile, false)
	if err != nil || replacement.Slot != victim {
		t.Fatalf("full active pool did not reclaim its own moving head: %+v error%v", replacement, err)
	}
	for i, binding := range other.poolShadows {
		if *other.Pool.Slot(binding.Slot) != protected[i] {
			t.Fatal("capacity pressure changed the inactive player's protected shadow")
		}
	}
	if *other.Pool.Slot(other.Weapons.mounts[0].Binding.Slot) != protected[4] {
		t.Fatal("capacity pressure changed the inactive player's equipment")
	}
	if !s.switchTurn() || s.ActiveWorld() != other {
		t.Fatal("normal turn admission failed after saturation")
	}
	seen := make(map[int]bool)
	for index := 0; index < ActorPoolCapacity; index++ {
		slot := other.Pool.Slot(index)
		if !slot.allocated {
			continue
		}
		if slot.EntityID == 0 || seen[slot.EntityID] {
			t.Fatal("shared physical records retained ambiguous creation identities")
		}
		seen[slot.EntityID] = true
	}
}

func TestTwoPlayerForecastOwnsPhysicalStorageForEitherActivePlayer(t *testing.T) {
	s := pooledTwoPlayerSession(t)
	for player := 0; player < 2; player++ {
		source := s.Players[player]
		before := *source.Pool.storage()
		var forecast WorldForecast
		if err := forecast.Load(source); err != nil {
			t.Fatal(err)
		}
		copy := forecast.State()
		if copy.Pool.storage() == source.Pool.storage() || !copy.Pool.sameState(source.Pool) {
			t.Fatal("forecast did not independently copy the shared physical arena")
		}
		if _, err := copy.reserveWorldActor(200, ActorPoolMoving, false); err != nil {
			t.Fatal(err)
		}
		if *source.Pool.storage() != before || s.Players[player^1].Pool.storage() != source.Pool.storage() {
			t.Fatal("forecast allocation mutated a saved player's live physical records")
		}
	}
}

func TestTwoPlayerCapacityKeepsInactiveGuardianRecordsOptional(t *testing.T) {
	s, err := NewSession(originalWorldData(t, 5), 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	for _, world := range s.Players {
		if err := world.activateFifthGuardian(visualassets.FixedEncounter{Y: 2336}, false); err != nil {
			t.Fatal(err)
		}
	}
	active, inactive := s.Players[0], s.Players[1]
	protected := make(map[int]ActorPoolSlot)
	for _, binding := range inactive.poolBindings {
		protected[binding.Slot] = *inactive.Pool.Slot(binding.Slot)
	}
	if len(protected) <= 5 {
		t.Fatal("inactive fixture did not reserve its real guardian components")
	}
	for active.Pool.FreeFirst() != NoActorSlot {
		if _, err := active.reserveWorldActor(200, ActorPoolMoving, false); err != nil {
			t.Fatal(err)
		}
	}
	for range 8 {
		if _, err := active.reserveWorldActor(12, ActorPoolProjectile, false); err != nil {
			t.Fatal(err)
		}
	}
	for index, before := range protected {
		if *inactive.Pool.Slot(index) != before {
			t.Fatalf("active capacity pressure changed inactive guardian/equipment slot%d", index)
		}
	}
}

func TestNextStageRetainsOneArenaForBothSavedGames(t *testing.T) {
	s := pooledTwoPlayerSession(t)
	s.Players[0].LevelFinished = true
	if route, err := s.CompleteStage(stageData(t, 2)); err != nil || route != WaitForOtherPlayer {
		t.Fatalf("first completed player: route%d error%v", route, err)
	}
	s.Players[1].LevelFinished = true
	if route, err := s.CompleteStage(stageData(t, 2)); err != nil || route != LoadedNextStage {
		t.Fatalf("second completed player: route%d error%v", route, err)
	}
	if s.Players[0].Pool.storage() != s.Players[1].Pool.storage() || s.Players[0].Pool.First(ActorPoolPlayer) == s.Players[1].Pool.First(ActorPoolPlayer) {
		t.Fatal("shared stage loading split the physical arena or joined saved player lists")
	}
}
