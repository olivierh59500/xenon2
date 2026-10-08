package engine

import "testing"

// The original final checkpoint/encounter stream creates the native guardian.
// Initial inventory/shield describe an explicit checkpoint fixture, not a
// reconstructed four-stage victory. No guardian health is changed by this setup.
func fourthFinalNativeCheckpointScene(t testing.TB) (*World, *PresentationPilot) {
	t.Helper()
	e := NewEquipment()
	for _, item := range []Item{ItemPowerup, ItemCannon, ItemRearShot, ItemPowerup, ItemSpeedup, ItemSpeedup, ItemAutofire, ItemAutofire} {
		e.ApplyItem(item)
	}
	e.Shield, e.Lives = 31, 1
	data := playableOriginalWorldData(t, 4)
	data.InitialEquipment = &e
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 176, 152
	w.MinimumScrollY = 0
	w.RestartCheckpoint()
	w.ContinueCredits = 2
	if err = w.Step(Input{Fire: true}); err != nil {
		t.Fatal(err)
	}
	p := &PresentationPilot{PALRefreshes: 3}
	for pass := 0; pass < 80; pass++ {
		input := p.NormalInput(w)
		for range 3 {
			w.AdvancePALTick()
		}
		if err = w.Step(input); err != nil {
			t.Fatal(err)
		}
		if w.FourthFinal != nil {
			if w.FourthFinal.Parts[0].Health != 100 || w.FourthFinal.Parts[1].Health != 50 || w.FourthFinal.Parts[2].Health != 50 || !w.PlayerAlive {
				t.Fatal("original final selector changed native health")
			}
			return w, p
		}
	}
	t.Fatal("original final selector did not activate")
	return nil, nil
}

func TestFourthFinalNativeScopeResourcesAndTriggerReleaseOptional(t *testing.T) {
	w, _ := fourthFinalNativeCheckpointScene(t)
	if !fourthFinalNativeEligible(w, 3) {
		t.Fatalf("native checkpoint profile rejected: %+v", w.Equipment)
	}
	for _, mode := range []string{"level", "missing-final", "defeated", "ready", "dead", "blocked-fire", "strobe", "partial", "shop", "drops", "dive", "PAL", "primary", "rear", "extra-mount", "art", "ships", "paths", "stencil", "coverage", "weapons", "boxes"} {
		t.Run(mode, func(t *testing.T) {
			var clone WorldForecast
			if err := clone.Load(w); err != nil {
				t.Fatal(err)
			}
			q, pal := clone.State(), 3
			switch mode {
			case "level":
				q.Level.Number = 5
			case "missing-final":
				q.FourthFinal = nil
			case "defeated":
				q.FourthFinal.Defeated = true
			case "ready":
				q.Ready = true
			case "dead":
				q.PlayerAlive = false
			case "blocked-fire":
				q.blockedFireUntilRelease = true
			case "strobe":
				q.ScreenClearFrames = 1
			case "partial":
				q.stepContinuation.active = true
			case "shop":
				q.ShopReady = true
			case "drops":
				q.PendingExitDrops = 1
			case "dive":
				q.Dive.Phase = 1
			case "PAL":
				pal = 1
			case "primary":
				q.Equipment.Primary.Item = ItemFlamer
			case "rear":
				q.Equipment.Rear.Tier = 0
			case "extra-mount":
				q.Equipment.Mounts[1].Item = ItemCannon
			case "art":
				q.fourthFinalArt = nil
			case "ships":
				q.Level.Ships = nil
			case "paths":
				q.Level.Paths = nil
			case "stencil":
				q.Level.PlayerStencil = nil
			case "coverage":
				q.Coverage = nil
			case "weapons":
				q.Weapons = nil
			case "boxes":
				q.movingSpriteBoxes = nil
			}
			if fourthFinalNativeEligible(q, pal) {
				t.Fatal("final specialist escaped its native resource/profile/lifecycle scope")
			}
		})
	}
	w.blockedFireUntilRelease = true
	p := PresentationPilot{PALRefreshes: 3}
	input := p.NormalInput(w)
	if input.Fire || p.fourthFinalNative != nil {
		t.Fatal("specialist bypassed the ordinary trigger release")
	}
	for range 3 {
		w.AdvancePALTick()
	}
	if err := w.Step(input); err != nil {
		t.Fatal(err)
	}
	if w.blockedFireUntilRelease {
		t.Fatal("ordinary release did not clear the native trigger gate")
	}
	p.NormalInput(w)
	if p.fourthFinalNative == nil {
		t.Fatal("released native final did not admit specialist")
	}
}

func TestFourthFinalNativeReadOnlyIdempotenceAndCacheOwnershipOptional(t *testing.T) {
	w, p := fourthFinalNativeCheckpointScene(t)
	before := forecastIsolationDigest(w)
	input := p.NormalInput(w)
	controller := p.fourthFinalNative
	if controller == nil || !input.Fire {
		t.Fatal("native final pilot was not admitted")
	}
	key := controller.key
	if p.NormalInput(w) != input || controller.key != key {
		t.Fatal("same-frame native final sampling changed cached input")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("native final callback forecast changed live state")
	}
	// Same frame, different native guardian state must rebuild its decision.
	w.FourthFinal.Parts[0].Counter++
	p.NormalInput(w)
	if controller.key == key || controller.key != fourthFinalNativeState(w) {
		t.Fatal("same-frame guardian mutation retained an obsolete decision")
	}
	var foreign WorldForecast
	if err := foreign.Load(w); err != nil {
		t.Fatal(err)
	}
	old := controller
	p.NormalInput(foreign.State())
	if p.fourthFinalNative == nil || p.fourthFinalNative == old || p.fourthFinalNative.world != foreign.State() {
		t.Fatal("native final pilot retained foreign world ownership")
	}
}
