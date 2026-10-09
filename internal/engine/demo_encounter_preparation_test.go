package engine

import (
	"testing"
	"time"
)

func preparedEncounterFixture(t testing.TB) (*World, *expertEncounterPilot) {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	e := &expertEncounterPilot{}
	if _, ok := e.selectPlan(w, 3); !ok {
		t.Fatal("original encounter scene rejected")
	}
	before := forecastIsolationDigest(w)
	e.prepareNext(w)
	waitPreparedEncounter(t, e)
	if forecastIsolationDigest(w) != before {
		t.Fatal("background preparation changed the live scene")
	}
	for _, input := range e.inputs {
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
	}
	return w, e
}

func waitPreparedEncounter(t testing.TB, e *expertEncounterPilot) {
	t.Helper()
	if e.prepared == nil {
		t.Fatal("source continuation was not prepared")
	}
	select {
	case ok := <-e.prepared.done:
		if !ok {
			t.Fatal("source continuation failed")
		}
		e.prepared.done <- ok
	case <-time.After(5 * time.Second):
		t.Fatal("bounded source preparation did not finish")
	}
}

func TestPreparedEncounterMatchesIndependentDecisionAndReusesStorageOptional(t *testing.T) {
	w, first := preparedEncounterFixture(t)
	before := forecastIsolationDigest(w)
	second, ok := first.takePrepared(w, 3)
	if !ok || second == first || second.spare != first || forecastIsolationDigest(w) != before {
		t.Fatal("prepared decision failed ownership, storage reuse or source isolation")
	}
	var reference expertEncounterPilot
	if _, ok := reference.selectPlan(w, 3); !ok || second.inputs != reference.inputs || second.keys != reference.keys {
		t.Fatal("preparation changed the independent source decision")
	}
	second.prepareNext(w)
	waitPreparedEncounter(t, second)
	for _, input := range second.inputs {
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
	}
	if returned, ok := second.takePrepared(w, 3); !ok || returned != first {
		t.Fatal("the two owned planning buffers did not rotate back")
	}
}

func TestPreparedEncounterRejectsControllerChangesBeyondPhysicalPoolOptional(t *testing.T) {
	w, e := preparedEncounterFixture(t)
	if len(w.Actors) == 0 {
		t.Fatal("original scene has no actor to test")
	}
	pool, key := *w.Pool, retainedGuardStateKey(w)
	w.Actors[0].motion.ProgramCounter++
	if *w.Pool != pool || retainedGuardStateKey(w) != key {
		t.Fatal("controller-only change unexpectedly changed the physical pool")
	}
	before := forecastIsolationDigest(w)
	if _, ok := e.takePrepared(w, 3); ok || e.spare == nil || forecastIsolationDigest(w) != before {
		t.Fatal("changed actor controller reused stale input or modified live state")
	}
}

func BenchmarkPreparedEncounterStateValidation(b *testing.B) {
	w, e := preparedEncounterFixture(b)
	q := e.prepared.plan.forecast.State()
	if !expertPreparedWorldMatches(w, q) {
		b.Fatal("source continuation differs")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if !expertPreparedWorldMatches(w, q) {
			b.Fatal("source continuation differs")
		}
	}
}
