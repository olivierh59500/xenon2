package engine

import "reflect"

type expertEncounterPreparation struct {
	owner *World
	pal   int
	plan  *expertEncounterPilot
	done  chan bool
}

// The next decision starts while the current three verified controls execute.
// At most one background job belongs to a pilot. Two owned state graphs rotate
// between planning and playback, retaining their large copy buffers.
func (e *expertEncounterPilot) prepareNext(w *World) {
	if e.prepared != nil || e.forecast.Load(w) != nil {
		return
	}
	for _, input := range e.inputs {
		for range e.pal {
			e.forecast.AdvancePALTick()
		}
		r, err := e.forecast.Advance(input)
		if err != nil || r.Boundary != ForecastRunning {
			return
		}
	}
	if !practicedEncounterWindow(e.forecast.State()) || e.future.Load(e.forecast.State()) != nil {
		return
	}
	if e.spare == nil {
		e.spare = &expertEncounterPilot{}
	}
	next := e.spare
	e.spare = nil
	job := &expertEncounterPreparation{owner: w, pal: e.pal, plan: next, done: make(chan bool, 1)}
	e.prepared = job
	source := e.future.State()
	go func() {
		_, ok := next.selectPlan(source, job.pal)
		job.done <- ok
	}()
}

// A completed forecast is useful only for the exact live state it anticipated.
// No wait occurs on the rendering thread. A late or changed prediction keeps
// the ordinary synchronous planner, and completed storage is recycled.
func (e *expertEncounterPilot) takePrepared(w *World, pal int) (*expertEncounterPilot, bool) {
	job := e.prepared
	if job == nil {
		return nil, false
	}
	select {
	case ok := <-job.done:
		e.prepared = nil
		next := job.plan
		if ok && job.owner == w && job.pal == pal && expertPreparedWorldMatches(w, next.forecast.State()) {
			next.world, next.at, next.spare = w, 1, e
			return next, true
		}
		e.spare = next
	default:
	}
	return nil, false
}

// The pool includes physical list order and every surviving residue. Mutable
// controller graphs and weapon state are checked separately; pointer identity
// and rebound function closures do not define gameplay equivalence.
func expertPreparedWorldMatches(w, q *World) bool {
	if q == nil || retainedGuardStateKey(w) != retainedGuardStateKey(q) || fifthPracticeMarker(w) != fifthPracticeMarker(q) ||
		w.Pool == nil || q.Pool == nil || *w.Pool != *q.Pool || w.Checkpoint != q.Checkpoint || w.WaveBonuses != q.WaveBonuses ||
		w.ThirdStage != q.ThirdStage || w.EffectActive != q.EffectActive || w.shipTrail != q.shipTrail {
		return false
	}
	if !reflect.DeepEqual(w.ThirdFinal, q.ThirdFinal) || !reflect.DeepEqual(w.BackgroundStars, q.BackgroundStars) ||
		!reflect.DeepEqual(w.Actors, q.Actors) || !reflect.DeepEqual(w.Projectiles, q.Projectiles) ||
		!reflect.DeepEqual(w.SmallShots, q.SmallShots) || !reflect.DeepEqual(w.Collectibles, q.Collectibles) {
		return false
	}
	if w.Weapons == nil || q.Weapons == nil {
		return w.Weapons == nil && q.Weapons == nil
	}
	return w.Weapons.nextID == q.Weapons.nextID && w.Weapons.savedMountsActive == q.Weapons.savedMountsActive &&
		reflect.DeepEqual(w.Weapons.mounts, q.Weapons.mounts) && reflect.DeepEqual(w.Weapons.savedMounts, q.Weapons.savedMounts) &&
		reflect.DeepEqual(w.Weapons.projectiles, q.Weapons.projectiles)
}
