package engine

import "testing"

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
