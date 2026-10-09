package engine

import "sync"

type thirdMiddleCandidate struct {
	result       ForecastResult
	hp, distance int
	stateful     bool
}

func emptyThirdMiddleCandidate() thirdMiddleCandidate {
	return thirdMiddleCandidate{result: ForecastResult{Shield: -1}, hp: 1000000, distance: 1000000}
}

func thirdMiddleCandidateBetter(candidate, best thirdMiddleCandidate) bool {
	return candidate.result.Alive && !best.result.Alive || candidate.result.Alive == best.result.Alive &&
		(candidate.result.Shield > best.result.Shield || candidate.result.Shield == best.result.Shield &&
			(candidate.hp < best.hp || candidate.hp == best.hp && candidate.distance < best.distance))
}

// Both evaluation paths execute this same complete callback sequence. Only
// independent workers stop before stateful post-defeat route continuation.
func evaluateThirdMiddleCandidate(forecast *WorldForecast, policy *DemoPilot, motion MotionInput, fallback Input, pal int, independent bool) (thirdMiddleCandidate, error) {
	var candidate thirdMiddleCandidate
	for pass := 0; pass < 36; pass++ {
		state := forecast.State()
		input := fallback
		if pass < 3 || state.ThirdMiddle != nil && !state.ThirdMiddle.Defeated {
			input.Motion = motion
			// Evaluate a complete held escape while the guardian is active.
			// Returning to the short pursuit policy after three passes can
			// make every escape inherit the same avoidable arm collision.
		} else {
			if independent && (state.ThirdMiddle == nil || state.ThirdMiddle.Defeated) {
				candidate.stateful = true
				return candidate, nil
			}
			input = policy.NormalInput(state)
		}
		input.Fire = !state.blockedFireUntilRelease && state.Dive.Phase == 0 && presentationShotOpportunityForMotion(state, input.Motion)
		for range pal {
			forecast.AdvancePALTick()
		}
		var err error
		candidate.result, err = forecast.Advance(input)
		if err != nil {
			return candidate, err
		}
		if candidate.result.Boundary != ForecastRunning {
			break
		}
	}
	end := forecast.State()
	candidate.hp = max(0, int(int16(end.ThirdMiddle.EyeHealth[0]))) + max(0, int(int16(end.ThirdMiddle.EyeHealth[1])))
	target := 3
	if int16(end.ThirdMiddle.EyeHealth[0]) <= 0 {
		target = 4
	}
	candidate.distance = absDemo(end.Player.X-end.ThirdMiddle.Parts[target].X) + absDemo(end.Player.Y-176)
	return candidate, nil
}

type thirdMiddleForecastWorker struct {
	forecast  WorldForecast
	policy    DemoPilot
	candidate thirdMiddleCandidate
	err       error
}

// Each candidate has a private persistent state graph. Nine bounds the work;
// the Go scheduler bounds simultaneously executing workers to GOMAXPROCS.
// Goroutines last only for the current decision, so resetting a pilot cannot
// leave an idle worker holding its previous world's state forever.
type thirdMiddleForecastWorkers struct {
	workers [9]thirdMiddleForecastWorker
}

func (worker *thirdMiddleForecastWorker) evaluate(w *World, motion MotionInput, fallback Input, pal int, sourcePolicy DemoPilot) {
	navigation, scratch := worker.policy.navigation, worker.policy.secondArenaScratch
	nativeMotion := worker.policy.nativeMotion
	worker.policy = sourcePolicy
	worker.policy.navigation = navigation
	worker.policy.nativeMotion = nativeMotion
	worker.policy.secondArenaScratch = append(scratch[:0], sourcePolicy.secondArenaScratch...)
	worker.err = worker.forecast.Load(w)
	if worker.err != nil {
		return
	}
	prepareThirdMiddlePolicy(&worker.policy, worker.forecast.State(), w.ScrollY)
	worker.candidate, worker.err = evaluateThirdMiddleCandidate(&worker.forecast, &worker.policy, motion, fallback, pal, true)
}

func (p *PresentationPilot) forecastThirdMiddleParallel(w *World, fallback Input) Input {
	if err := p.forecast.Load(w); err != nil {
		return fallback
	}
	prepareThirdMiddlePolicy(&p.middleForecastPolicy, p.forecast.State(), w.ScrollY)
	if p.middleWorkers == nil {
		p.middleWorkers = &thirdMiddleForecastWorkers{}
	}
	workers, pal := p.middleWorkers, thirdMiddlePALRefreshes(p.PALRefreshes)
	var pending sync.WaitGroup
	pending.Add(len(workers.workers) - 1)
	for index := 1; index < len(workers.workers); index++ {
		go func(index int) {
			defer pending.Done()
			workers.workers[index].evaluate(w, demoDirections[index], fallback, pal, p.middleForecastPolicy)
		}(index)
	}
	workers.workers[0].evaluate(w, demoDirections[0], fallback, pal, p.middleForecastPolicy)
	pending.Wait()
	for index := range workers.workers {
		if workers.workers[index].err != nil || workers.workers[index].candidate.stateful {
			// The serial continuation can carry route state between candidates.
			// Re-evaluate in its original order rather than sharing that state.
			return p.forecastThirdMiddleSerial(w, fallback)
		}
	}
	best := fallback
	score := emptyThirdMiddleCandidate()
	for index, motion := range demoDirections {
		candidate := workers.workers[index].candidate
		if thirdMiddleCandidateBetter(candidate, score) {
			best, score = fallback, candidate
			best.Motion = motion
		}
	}
	// Preserve candidate eight for read-only inspection. An owning forecast
	// cannot be assigned by value because its pointers refer to its storage.
	if err := p.forecast.Load(workers.workers[8].forecast.State()); err != nil {
		return fallback
	}
	best.Fire = !w.blockedFireUntilRelease && w.Dive.Phase == 0 && presentationShotOpportunityForMotion(w, best.Motion)
	return best
}
