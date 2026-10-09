package engine

import "sync"

type expertThirdFinalWorker struct {
	forecast WorldForecast
	score    expertEncounterScore
	health   int
	ok       bool
}

type expertThirdFinalWorkers struct {
	workers [9]expertThirdFinalWorker
}

func (worker *expertThirdFinalWorker) evaluate(w *World, motion MotionInput, pal int) {
	worker.ok = false
	if worker.forecast.Load(w) != nil {
		return
	}
	worker.score = expertEncounterScore{alive: true}
	for pass := 0; pass < 36; pass++ {
		q := worker.forecast.State()
		input := Input{Motion: motion}
		// A held escape does not assume that a later pursuit will be safe.
		// Native callbacks decide every hit, armored interception and loss.
		input.Fire = !q.blockedFireUntilRelease && q.Dive.Phase == 0 && (presentationGeometricShotOpportunity(q, input.Motion) || presentationAuxiliaryShotOpportunity(q, input.Motion))
		shield := q.Equipment.Shield
		for range pal {
			worker.forecast.AdvancePALTick()
		}
		r, err := worker.forecast.Advance(input)
		if err != nil {
			return
		}
		worker.score.damage += max(0, shield-r.Shield)
		worker.score.alive = r.Alive
		if worker.forecast.State().Rewind.Timer != 0 {
			worker.score.contacts++
		}
		if r.Boundary != ForecastRunning {
			break
		}
	}
	q := worker.forecast.State()
	worker.health = max(0, int(int16(q.ThirdFinal.Health)))
	worker.score.value = -worker.health*1000 - absDemo(q.Player.Y-176)
	for _, actor := range q.Actors {
		if actor.Active && actor.thirdFinalMember != nil && actor.thirdPart != nil && actor.thirdPart.Index == 0 {
			worker.score.value -= absDemo(q.Player.X - int(actor.X))
			break
		}
	}
	worker.ok = true
}

func (p *PresentationPilot) forecastThirdFinalInput(w *World, fallback Input) Input {
	if w.Level.Number != 3 || w.ScrollY > 208 || w.ThirdFinal == nil || w.ThirdFinal.Defeated || w.ThirdFinal.LaunchCount == 0 || !w.PlayerAlive || w.Ready {
		return fallback
	}
	if p.finalForecasts == nil {
		p.finalForecasts = &expertThirdFinalWorkers{}
	}
	workers := p.finalForecasts
	pal := thirdMiddlePALRefreshes(p.PALRefreshes)
	var pending sync.WaitGroup
	for index := 1; index < len(workers.workers); index++ {
		pending.Add(1)
		go func(index int) {
			defer pending.Done()
			workers.workers[index].evaluate(w, demoDirections[index], pal)
		}(index)
	}
	workers.workers[0].evaluate(w, demoDirections[0], pal)
	pending.Wait()
	best := expertEncounterScore{damage: int(^uint(0) >> 1)}
	for index := range workers.workers {
		candidate := &workers.workers[index]
		if !candidate.ok {
			return fallback
		}
		if candidate.score.better(best) {
			best, fallback.Motion = candidate.score, demoDirections[index]
		}
	}
	// Authenticate the current shot through the existing native point observer;
	// later volleys in the motion forecast cannot justify an empty current shot.
	fallback.Fire = !w.blockedFireUntilRelease && w.Dive.Phase == 0 && presentationShotOpportunityWithForecast(w, fallback.Motion, &p.guardianAimForecast, pal)
	return fallback
}
