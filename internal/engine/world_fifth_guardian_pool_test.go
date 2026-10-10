package engine

import (
	"fmt"
	"testing"

	"xenon2/internal/visualassets"
)

func fifthGuardianPoolFixture(t *testing.T, final bool) (*World, []*WorldActor, int) {
	t.Helper()
	w := testWorld(t)
	w.Level.Number, w.ScrollY = 5, 900
	middle := &visualassets.GuardianGroup{Components: make([]visualassets.GuardianComponent, 10)}
	ending := &visualassets.GuardianGroup{Components: make([]visualassets.GuardianComponent, 22)}
	for _, group := range []*visualassets.GuardianGroup{middle, ending} {
		for index := range group.Components {
			group.Components[index] = visualassets.GuardianComponent{ResourceTag: 276, Health: 40, OffsetX: index * 3, OffsetY: index * 4}
		}
	}
	middle.Components[0] = visualassets.GuardianComponent{ResourceTag: 252, Behavior: "middle-body-controller"}
	middle.Components[5] = visualassets.GuardianComponent{ResourceTag: 288, Health: 200}
	ending.Components[0] = visualassets.GuardianComponent{ResourceTag: 308, Health: 20, Behavior: "final-body-controller"}
	ending.Components[21] = visualassets.GuardianComponent{ResourceTag: 80, Health: 20, StrongHealth: true}
	w.fifthMiddleArt, w.fifthFinalArt = middle, ending
	for _, name := range []string{"explosion-small", "explosion-large", "cash-small", "cash-large"} {
		w.commonAnimations[name] = visualassets.NamedActorAnimation{Ending: "hold", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: name, Duration: 2}}}}
	}
	first := w.Pool.FreeFirst()
	for slot := first; slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
		w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	}
	if err := w.activateFifthGuardian(visualassets.FixedEncounter{Y: 1008}, final); err != nil {
		t.Fatal(err)
	}
	actors := w.fifthMiddleActors[:]
	if final {
		actors = w.fifthFinalActors[:]
	}
	return w, append([]*WorldActor(nil), actors...), first
}

func TestFifthGuardianConstructorsKeepSpareWordsAndPublishLiveFlags(t *testing.T) {
	for _, final := range []bool{false, true} {
		w, actors, _ := fifthGuardianPoolFixture(t, final)
		for _, actor := range actors {
			slot := w.Pool.Slot(actor.Binding.Slot)
			seed := waveConstructorResidueFixture(actor.Binding.Slot)
			if !slot.AuxiliaryFlags[0] || slot.Residue.MountOffsetX != seed.MountOffsetX || slot.Residue.MountOffsetY != seed.MountOffsetY || slot.Residue.XFraction != seed.XFraction || slot.Residue.YFraction != seed.YFraction {
				t.Fatalf("final %v part %d constructor overwrote retained fields or missed its live flag: %+v", final, actor.fifthIndex, slot)
			}
		}
	}
}

func TestFifthMiddleMountEffectIgnoresInheritedContactStrength(t *testing.T) {
	w, actors, _ := fifthGuardianPoolFixture(t, false)
	mount := actors[1]
	if !mount.part.StrongHealth {
		t.Fatal("fixture omitted the inherited strong-contact byte")
	}
	w.damageFifthGuardian(mount, 40)
	var effect *WorldActor
	for _, actor := range w.Actors {
		if actor.ActorList == "transient" {
			effect = actor
		}
	}
	if effect == nil || effect.Sprite != "explosion-small" || effect.X != 120 || effect.Y != 108 || w.SoundRequests[2] != "sampled-effect-05" || !mount.Active || !w.FifthMiddle.Parts[1].Destroyed {
		t.Fatal("middle mount lost its fixed small effect, eight-pixel centre or retained wreck")
	}
}

func TestFifthCoreDeathReleasesMovingEntriesBeforeRewardAllocation(t *testing.T) {
	for _, final := range []bool{false, true} {
		t.Run(fmt.Sprint(final), func(t *testing.T) {
			w, actors, first := fifthGuardianPoolFixture(t, final)
			binding, err := w.reserveWorldActor(200, ActorPoolMoving, true)
			if err != nil {
				t.Fatal(err)
			}
			other := &WorldActor{ID: binding.EntityID, Binding: binding, Active: true, ActorList: "moving", part: &visualassets.ActorPart{ResourceTag: 200}}
			w.poolActors[binding.Slot] = other
			w.Actors = append(w.Actors, other)
			before := w.nextActorID
			core, damage, effects, cash := actors[5], uint16(200), 20, 10
			if final {
				core, damage, effects, cash = actors[21], 20, 40, 20
			}
			w.damageFifthGuardian(core, damage)
			if w.poolError != nil || w.Pool.First(ActorPoolMoving) != NoActorSlot || w.Pool.FreeFirst() != first+effects+cash || other.Active || w.LevelFinished != final {
				t.Fatalf("core cleanup delayed native release or lost its boundary: first free %d expected %d error %v", w.Pool.FreeFirst(), first+effects+cash, w.poolError)
			}
			var earliest *WorldActor
			count := 0
			for _, actor := range w.Actors {
				if actor.ID > before && actor.ActorList == "transient" {
					count++
					if earliest == nil || actor.ID < earliest.ID {
						earliest = actor
					}
				}
			}
			if count != effects || len(w.Collectibles) != cash || w.PendingExitDrops != cash || earliest == nil || !earliest.Active {
				t.Fatal("core death lost its real effect or cash count")
			}
			firstCash := w.Collectibles[0]
			for _, coin := range w.Collectibles {
				if coin.ID < firstCash.ID {
					firstCash = coin
				}
			}
			firstSlot := earliest.Binding.Slot
			if final {
				firstSlot = firstCash.Binding.Slot
				if firstCash.ID >= earliest.ID {
					t.Fatal("final death did not allocate cash before explosions")
				}
			} else if earliest.ID >= firstCash.ID {
				t.Fatal("middle death did not allocate explosions before cash")
			}
			if firstSlot != binding.Slot {
				t.Fatal("head-first release did not make the last moving entry the first new allocation")
			}
			for _, actor := range actors {
				if actor.Active || actor.Binding.EntityID != 0 {
					t.Fatal("released guardian retained its logical physical ownership")
				}
			}
		})
	}
}
