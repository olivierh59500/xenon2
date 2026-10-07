package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestWeaponContextReuseRefreshesInputAndPhaseState(t *testing.T) {
	w := testWorld(t)
	var err error
	w.Weapons, err = NewWeaponRuntime(&visualassets.SpriteAtlas{})
	if err != nil {
		t.Fatal(err)
	}
	first := w.weaponContext(Input{Fire: true, Motion: MotionInput{Left: true}}, true)
	// A weapon phase can customize a context after creation. Those fields must
	// not leak into the following phase when only its callbacks are reused.
	first.SkipSmallWeapons, first.AvailableActorSlots = true, 23
	first.NextActorDriftResidue = func() uint16 { return 0x1234 }
	w.Weapons.context = first
	w.Player.X, w.Player.Y = 207, 88
	w.MaterializationFrames, w.Dive.Phase, w.PlayerAlive = 7, 1, false
	w.EffectActive[1] = true
	w.SetRandomState(RandomState{A: 0x13579bdf, B: 0x2468ace0})
	input := Input{Motion: MotionInput{Right: true}}
	next := w.weaponContext(input, false)
	if next.Held || next.Pulse || next.Motion != input.Motion || next.ShipX != 207 || next.ShipY != 88 || !next.Diving || !next.Materializing || !next.ShipDestroyed || next.MaterializationFrames != 7 {
		t.Fatal("cached callbacks retained old input, ship or phase state")
	}
	if next.SkipSmallWeapons || next.AvailableActorSlots != 0 || next.NextActorDriftResidue != nil {
		t.Fatal("phase-specific weapon options leaked into a new context")
	}
	before := w.RandomState()
	if got, want := next.NextRandom(), before.Next(); got != want || w.RandomState() != before {
		t.Fatal("reused random callback did not use the world's current random state")
	}
	next.SoundVoice(1, "test")
	next.SoundVoiceIfEmpty(1, "replacement")
	next.ImmediateSoundVoice(2, "immediate")
	next.StopEffects()
	if w.SoundRequests[1] != "test" || w.ImmediateSoundRequests[2] != "immediate" || !next.EffectActive(1) || !w.StopEffectsRequested {
		t.Fatal("reused sound callbacks did not address current world state")
	}
}

func TestWeaponContextRebindsCopiedRuntimeToItsCurrentWorld(t *testing.T) {
	live := testWorld(t)
	var err error
	live.Weapons, err = NewWeaponRuntime(&visualassets.SpriteAtlas{})
	if err != nil {
		t.Fatal(err)
	}
	live.Weapons.context = live.weaponContext(Input{}, false)
	copy := *live
	runtime := *live.Weapons
	copy.Weapons = &runtime
	context := copy.weaponContext(Input{}, false)
	id, seed := live.nextActorID, live.RandomState()
	context.NextID()
	context.NextRandom()
	context.Sound("copied")
	context.StopEffects()
	if live.nextActorID != id || live.RandomState() != seed || live.SoundRequests[2] != "" || live.StopEffectsRequested {
		t.Fatal("copied runtime callbacks changed the source world")
	}
	if copy.nextActorID != id+1 || copy.RandomState() == seed || copy.SoundRequests[2] != "copied" || !copy.StopEffectsRequested {
		t.Fatal("copied runtime callbacks were not rebound to the new world")
	}
}
