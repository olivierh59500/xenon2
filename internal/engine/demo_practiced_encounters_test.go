package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestPostMiddleEncounterPracticeKeepsCannonRouteBetweenFormations(t *testing.T) {
	w := testWorld(t)
	w.Level.Number, w.ScrollY = 3, 1000
	w.ThirdMiddle = &ThirdGuardianState{Defeated: true}
	w.Coverage, w.Level.PlayerStencil = &TerrainCoverage{}, &visualassets.PlayerTerrainStencil{}
	if practicedEncounterWindow(w) {
		t.Fatal("ordinary post-middle corridor must retain its cannon route")
	}
	actor := &WorldActor{Active: true, X: 352, Y: 120, part: &visualassets.ActorPart{MotionMode: "path-entry-edge-frames"}}
	w.Actors = append(w.Actors, actor)
	if !practicedEncounterWindow(w) {
		t.Fatal("directed incoming formation did not receive full callback anticipation")
	}
	actor.Active = false
	if practicedEncounterWindow(w) {
		t.Fatal("expired formation kept control away from the ordinary cannon route")
	}
	actor.Active, actor.X = true, 500
	if practicedEncounterWindow(w) {
		t.Fatal("distant formation took control of the local cannon route")
	}
	actor.X = 343
	w.ThirdMiddle.Defeated = false
	if practicedEncounterWindow(w) {
		t.Fatal("formation anticipation replaced the active middle guardian policy")
	}
	w.ThirdMiddle.Defeated, w.ScrollY = true, 208
	if practicedEncounterWindow(w) {
		t.Fatal("formation anticipation replaced the final guardian policy")
	}
}

func TestPostMiddlePracticeRehearsesNearbyCannonFiringLanes(t *testing.T) {
	w := testWorld(t)
	w.Level.Number, w.ScrollY = 3, 1000
	w.Player.X, w.Player.Y = 160, 150
	w.ThirdMiddle = &ThirdGuardianState{Defeated: true}
	w.Coverage, w.Level.PlayerStencil = &TerrainCoverage{}, &visualassets.PlayerTerrainStencil{}
	a := &WorldActor{Active: true, Health: 24, thirdCannon: &ThirdCannonState{},
		Collision: CollisionRect{Left: 144, Right: 175, Top: 60, Bottom: 87}}
	w.Actors = []*WorldActor{a}
	if !practicedEncounterWindow(w) {
		t.Fatal("nearby cannon did not receive full callback anticipation")
	}
	var pilot expertEncounterPilot
	count := pilot.prepareGoals(w)
	for _, x := range []int{145, 159, 174} {
		for _, y := range []int{144, 172} {
			found := false
			for _, goal := range pilot.goals[:count] {
				found = found || goal.x == x && goal.y == y && !goal.route
			}
			if !found {
				t.Fatalf("weak-point firing lane %d,%d was displaced by generic goals", x, y)
			}
		}
	}
	w.Player.X = 260
	if practicedEncounterWindow(w) {
		t.Fatal("distant cannon displaced the route to its firing lane")
	}
	w.Player.X, w.Player.Y = 160, 50
	if practicedEncounterWindow(w) {
		t.Fatal("passed cannon below the ship kept forward engagement control")
	}
	w.Player.Y = 176
	if practicedEncounterWindow(w) {
		t.Fatal("cannon firing goals replaced a rearward terrain leg")
	}
	w.Player.Y, w.Player.ScrollReverseRequested = 150, true
	if practicedEncounterWindow(w) {
		t.Fatal("cannon firing goals interrupted reverse scrolling")
	}
	w.Player.ScrollReverseRequested = false
	w.Player.Y, a.Active = 150, false
	if practicedEncounterWindow(w) {
		t.Fatal("destroyed cannon kept its encounter window")
	}
}

func TestEncounterGoalsReserveReachableBubblesInCrowdedCannonRow(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y, w.Equipment.SpeedTier = 160, 150, 2
	for i := range 4 {
		w.Actors = append(w.Actors, &WorldActor{Active: true, Health: 24, thirdCannon: &ThirdCannonState{},
			Collision: CollisionRect{Left: 32 + i*64, Right: 63 + i*64, Top: 60, Bottom: 87}})
	}
	for i := range 4 {
		w.Collectibles = append(w.Collectibles, &WorldCollectible{ID: i + 1, Active: true, Cash: 100,
			X: float64(140 + i*10), Y: 130, Motion: CashMotion{X: 140 + i*10, Y: 130, Mode: 7}})
	}
	var pilot expertEncounterPilot
	count := pilot.prepareGoals(w)
	for _, item := range w.Collectibles {
		found := false
		for _, goal := range pilot.goals[:count] {
			found = found || goal.cash == item.ID
		}
		if !found {
			t.Fatalf("reachable bubble %d was displaced by the crowded cannon row", item.ID)
		}
		if !item.Active || item.Motion.X != int(item.X) || item.Motion.Y != int(item.Y) {
			t.Fatal("planning collected or moved the live reward")
		}
	}
}

func TestEncounterPracticeMatchesSerialCallbacksAndKeepsLiveWorldOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	p := PresentationPilot{PALRefreshes: 3}
	var serial expertEncounterWorker
	for pass := range 6 {
		before := forecastIsolationDigest(w)
		p.encounterPractice = &expertEncounterPilot{}
		input, ok := p.encounterPractice.selectPlan(w, 3)
		if !ok || forecastIsolationDigest(w) != before {
			t.Fatal("encounter rehearsal changed its source world")
		}
		e := p.encounterPractice
		count := e.prepareGoals(w)
		key := retainedGuardStateKey(w)
		for index, goal := range e.goals[:count] {
			want := serial.evaluate(w, goal, key, 3)
			if got := e.results[index]; got != want || !got.ok {
				t.Fatalf("worker changed callback outcomes at pass%d candidate%d", pass, index)
			}
		}
		at := e.at
		if repeated, ok := p.practicedEncounterInput(w); !ok || repeated != input || e.at != at {
			t.Fatal("repeated command consumed a retained encounter plan")
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		if retainedGuardStateKey(w) != e.keys[1] {
			t.Fatal("first ordinary live command diverged from its rehearsal")
		}
	}
}

func BenchmarkExpertEncounterDecision(b *testing.B) {
	w, err := NewWorld(playableOriginalWorldData(b, 3))
	if err != nil {
		b.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	p := PresentationPilot{PALRefreshes: 3}
	p.encounterPractice = &expertEncounterPilot{}
	if _, ok := p.encounterPractice.selectPlan(w, 3); !ok {
		b.Fatal("source encounter scene rejected")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		// Force a new decision while retaining all private copy buffers.
		p.encounterPractice.world = nil
		if _, ok := p.encounterPractice.selectPlan(w, 3); !ok {
			b.Fatal("source encounter scene rejected")
		}
	}
}
