package engine

import (
	"math"
	"runtime"
	"sync"
)

const expertEncounterHorizon = 72
const expertEncounterCommit = 3

// An encounter plan rehearses normal controls against the complete simulation.
// Upcoming formations are created by the ordinary encounter cursor, with their
// original paths, firing clocks, shared random stream and terrain callbacks.
// Only a few commands are retained before reassessing the live situation.
type expertEncounterPilot struct {
	forecast WorldForecast
	future   WorldForecast
	prepared *expertEncounterPreparation
	spare    *expertEncounterPilot
	workers  [8]expertEncounterWorker
	results  [24]expertEncounterResult
	goals    [24]expertEncounterGoal
	inputs   [expertEncounterCommit]Input
	keys     [expertEncounterCommit + 1]retainedGuardKey
	world    *World
	at, pal  int
}

type expertEncounterWorker struct {
	forecast WorldForecast
	policy   DemoPilot
}

type expertEncounterResult struct {
	score  expertEncounterScore
	inputs [expertEncounterCommit]Input
	keys   [expertEncounterCommit + 1]retainedGuardKey
	ok     bool
}

type expertEncounterGoal struct {
	x, y        int
	actor, cash int
	route       bool
}

type expertEncounterScore struct {
	alive            bool
	damage, contacts int
	value            int
}

func (a expertEncounterScore) better(b expertEncounterScore) bool {
	return a.alive && !b.alive || a.alive == b.alive &&
		(a.damage < b.damage || a.damage == b.damage &&
			(a.contacts < b.contacts || a.contacts == b.contacts && a.value > b.value))
}

func practicedEncounterWindow(w *World) bool {
	if w == nil {
		return false
	}
	section := w.Level.Number == 3 && w.ThirdMiddle == nil && w.ScrollY > 208
	if w.Level.Number == 3 && w.ThirdMiddle != nil && w.ThirdMiddle.Defeated && w.ScrollY > 208 {
		// Rehearse the directed formations through their actual sprite/collision
		// changes, while keeping the existing cannon route between encounters.
		for _, actor := range w.Actors {
			if actor.Active && actor.part != nil && actor.part.MotionMode == "path-entry-edge-frames" && actor.X >= -48 && actor.X <= 368 && actor.Y >= -48 && actor.Y <= 240 {
				section = true
				break
			}
		}
	}
	return section && w.PlayerAlive &&
		!w.GameOver && !w.Ready && !w.ShopReady && !w.LevelFinished && !w.ExitReady &&
		w.PendingExitDrops == 0 && w.ScreenClearFrames == 0 && !w.stepContinuation.active && w.Rewind.Timer == 0 && w.Coverage != nil && w.Level.PlayerStencil != nil
}

func (p *PresentationPilot) practicedEncounterInput(w *World) (Input, bool) {
	if !practicedEncounterWindow(w) {
		if p.encounterPractice != nil {
			p.encounterPractice.world = nil
		}
		return Input{}, false
	}
	if p.encounterPractice == nil {
		p.encounterPractice = &expertEncounterPilot{}
	}
	e := p.encounterPractice
	pal := thirdMiddlePALRefreshes(p.PALRefreshes)
	key := retainedGuardStateKey(w)
	if e.world == w && e.pal == pal {
		index := e.at
		if index > 0 && key == e.keys[index-1] {
			return e.inputs[index-1], true
		}
		if index < len(e.inputs) && key == e.keys[index] && e.replayRemaining(w, index) {
			e.at++
			return e.inputs[index], true
		}
	}
	if ready, ok := e.takePrepared(w, pal); ok {
		p.encounterPractice = ready
		ready.prepareNext(w)
		return ready.inputs[0], true
	}
	input, ok := e.selectPlan(w, pal)
	if ok {
		e.prepareNext(w)
	}
	return input, ok
}

func (e *expertEncounterPilot) selectPlan(w *World, pal int) (Input, bool) {
	key := retainedGuardStateKey(w)
	count := e.prepareGoals(w)
	best := expertEncounterScore{damage: math.MaxInt, contacts: math.MaxInt, value: math.MinInt}
	var selected [expertEncounterCommit]Input
	var selectedKeys [expertEncounterCommit + 1]retainedGuardKey
	workers := min(len(e.workers), runtime.GOMAXPROCS(0), count)
	var pending sync.WaitGroup
	evaluate := func(index int) {
		worker := &e.workers[index]
		for candidate := index; candidate < count; candidate += workers {
			e.results[candidate] = worker.evaluate(w, e.goals[candidate], key, pal)
		}
	}
	for index := 1; index < workers; index++ {
		pending.Add(1)
		go func(index int) {
			defer pending.Done()
			evaluate(index)
		}(index)
	}
	evaluate(0)
	pending.Wait()
	// Candidate order, including ties, is independent of worker scheduling.
	for _, candidate := range e.results[:count] {
		if !candidate.ok {
			return Input{}, false
		}
		if candidate.score.better(best) {
			best, selected, selectedKeys = candidate.score, candidate.inputs, candidate.keys
		}
	}
	if err := e.forecast.Load(w); err != nil {
		return Input{}, false
	}
	e.world, e.pal, e.at, e.inputs, e.keys = w, pal, 1, selected, selectedKeys
	return selected[0], true
}

func (worker *expertEncounterWorker) evaluate(w *World, goal expertEncounterGoal, key retainedGuardKey, pal int) expertEncounterResult {
	var result expertEncounterResult
	if err := worker.forecast.Load(w); err != nil {
		return result
	}
	result.keys[0] = key
	navigation := worker.policy.navigation
	if navigation != nil {
		navigation.path = navigation.path[:0]
		// A reused forecast has the same address but a different planning
		// history. Keep occupancy buffers, not a previous candidate's route.
		navigation.goal, navigation.frame = 0, 0
		navigation.retreat = false
		navigation.targetX, navigation.pathTargetX, navigation.pointTargetX = 0, 0, 0
		clear(navigation.pointClosed)
	}
	worker.policy = DemoPilot{practicedRoute: true, navigation: navigation, palRefreshes: pal}
	worker.policy.Config.DisableBonuses = true
	result.score.alive = true
	for pass := 0; pass < expertEncounterHorizon; pass++ {
		q := worker.forecast.State()
		x, y := goal.position(q)
		input := Input{Motion: expertEncounterMotion(q, x, y)}
		if goal.route {
			input = worker.policy.NormalInput(q)
		}
		if q.ThirdMiddle != nil && !q.ThirdMiddle.Defeated {
			input, _ = worker.policy.ThirdMiddleInput(q)
		}
		input.Fire = !q.blockedFireUntilRelease && q.Dive.Phase == 0 && (presentationShotOpportunityForMotion(q, input.Motion) || presentationAuxiliaryShotOpportunity(q, input.Motion))
		shield := q.Equipment.Shield
		for range pal {
			worker.forecast.AdvancePALTick()
		}
		r, err := worker.forecast.Advance(input)
		if err != nil {
			return result
		}
		result.score.damage += max(0, shield-r.Shield)
		q = worker.forecast.State()
		if q.Rewind.Timer != 0 {
			result.score.contacts++
		}
		if pass < expertEncounterCommit {
			result.inputs[pass], result.keys[pass+1] = input, retainedGuardStateKey(q)
		}
		if !r.Alive {
			result.score.alive = false
		}
		if r.Boundary != ForecastRunning {
			break
		}
	}
	q := worker.forecast.State()
	x, y := goal.position(q)
	result.score.value = (q.Score-w.Score)*4 + (q.Money-w.Money)*20 + (w.ScrollY-q.ScrollY)*20 - absDemo(q.Player.X-x) - absDemo(q.Player.Y-y)
	result.ok = true
	return result
}

// The compact key identifies a route position; exact callbacks validate reuse
// when enemy/projectile state changes without moving the ship or the camera.
func (e *expertEncounterPilot) replayRemaining(w *World, index int) bool {
	if err := e.forecast.Load(w); err != nil {
		return false
	}
	for pass := index; pass < len(e.inputs); pass++ {
		for range e.pal {
			e.forecast.AdvancePALTick()
		}
		if _, err := e.forecast.Advance(e.inputs[pass]); err != nil || retainedGuardStateKey(e.forecast.State()) != e.keys[pass+1] {
			return false
		}
	}
	return true
}

func (e *expertEncounterPilot) prepareGoals(w *World) int {
	count := 0
	add := func(goal expertEncounterGoal) {
		if count == len(e.goals) {
			return
		}
		for _, previous := range e.goals[:count] {
			if previous == goal {
				return
			}
		}
		e.goals[count] = goal
		count++
	}
	add(expertEncounterGoal{x: w.Player.X, y: 120})
	add(expertEncounterGoal{route: true})
	for _, y := range []int{172, 120, 56} {
		for _, x := range []int{48, 104, 160, 216, 272} {
			add(expertEncounterGoal{x: x, y: y})
		}
	}
	for _, cash := range w.Collectibles {
		if cash.Active && cash.Y >= 0 && cash.Y < 184 {
			add(expertEncounterGoal{x: int(cash.X), y: int(cash.Y), cash: cash.ID})
		}
	}
	for _, actor := range w.Actors {
		bounds, ok := presentationTargetBounds(w, actor)
		if ok && bounds.Bottom >= 0 && bounds.Top < w.Player.Y && bounds.Left < 320 && bounds.Right >= 0 {
			add(expertEncounterGoal{x: (bounds.Left + bounds.Right) / 2, y: 144, actor: actor.ID})
		}
	}
	return count
}

func (g expertEncounterGoal) position(w *World) (int, int) {
	if g.actor != 0 {
		for _, actor := range w.Actors {
			if actor.ID != g.actor {
				continue
			}
			bounds, ok := presentationTargetBounds(w, actor)
			if !ok || bounds.Bottom < 0 || bounds.Top >= w.Player.Y-12 {
				break
			}
			flight := max(1, min(16, (w.Player.Y-(bounds.Top+bounds.Bottom)/2-6)/9))
			if view, supported := demoActorPrediction(w, actor, flight, w.ScrollY-flight*w.BaseScrollStep); supported && view.Active {
				bounds = view.Bounds
			}
			return max(24, min(296, (bounds.Left+bounds.Right)/2)), g.y
		}
	}
	if g.cash != 0 {
		for _, cash := range w.Collectibles {
			if cash.ID == g.cash && cash.Active {
				return presentationBonusIntercept(w, cash)
			}
		}
	}
	return g.x, g.y
}

// A small feedback controller can turn at any pass. The complete world rollout
// ranks its actual hits and rewards; this local steering only rejects immediate
// terrain contact and accounts for the ship's acceleration and banking.
func expertEncounterMotion(w *World, x, y int) MotionInput {
	return expertEncounterSteering(w, x, y, false)
}

func expertEncounterSteering(w *World, x, y int, worldCoordinates bool) MotionInput {
	var hazards [3][ActorPoolCapacity * 2]CollisionRect
	var counts [3]int
	for future := 1; future <= len(hazards); future++ {
		add := func(bounds CollisionRect) {
			if !bounds.Empty() && counts[future-1] < len(hazards[future-1]) {
				hazards[future-1][counts[future-1]] = bounds
				counts[future-1]++
			}
		}
		for _, actor := range w.Actors {
			if !demoActorHazard(actor) {
				continue
			}
			if view, supported := demoActorPrediction(w, actor, future, w.ScrollY-future*w.BaseScrollStep); supported {
				if view.Active {
					add(view.Bounds)
				}
			} else {
				bounds := actor.Collision
				dx, dy := int(actor.X-actor.PreviousX)*future, int(actor.Y-actor.PreviousY)*future
				bounds.Left, bounds.Right, bounds.Top, bounds.Bottom = bounds.Left+dx, bounds.Right+dx, bounds.Top+dy, bounds.Bottom+dy
				add(bounds)
			}
		}
		for _, shot := range w.Projectiles {
			if sx, sy, active := demoProjectilePosition(w, shot, future, w.ScrollDelta); active {
				add(CollisionRect{Left: sx - 4, Right: sx + 4, Top: sy - 4, Bottom: sy + 4})
			}
		}
	}
	best, value := MotionInput{}, math.MaxInt
	for _, motion := range demoDirections {
		if motion.Down && w.Player.Y >= 176 && (!worldCoordinates || y <= w.Player.Y+w.ScrollY) {
			continue
		}
		state := newDemoMotionForecast(w)
		cost := 0
		for step := range 3 {
			if !state.advance(w, motion) || state.rewind.Timer != 0 {
				cost += 1000000
				break
			}
			playerY := state.player.Y
			if worldCoordinates {
				playerY += state.scroll.Y
			}
			cost += absDemo(state.player.X-x) + absDemo(playerY-y)*2
			bounds := thirdMiddlePlayerBounds(w, state.player)
			for _, hazard := range hazards[step][:counts[step]] {
				if bounds.Intersects(hazard) {
					cost += 100000
				}
			}
		}
		if cost < value {
			best, value = motion, cost
		}
	}
	return best
}
