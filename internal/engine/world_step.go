package engine

import "xenon2/internal/visualassets"

// worldStepContinuation keeps the source pass suspended inside reward18's
// blocking callback. Pooled traversal resumes by physical slot, not entity ID.
type worldStepContinuation struct {
	active, pooledDone bool
	input              Input
	deathFinished      bool
	projectileNext     int
	unbound            *worldUnboundContinuation
}

// Synthetic diagnostics can supply unbound objects. Their existing list
// snapshots and merge cursors are retained separately from the native cursor.
type worldUnboundContinuation struct {
	actors                                    []*WorldActor
	projectiles                               []*WorldProjectile
	smallShots                                []*WorldSmallShot
	collectibles                              []*WorldCollectible
	actorAt, enemyAt, playerAt, collectibleAt int
	actorsDone                                bool
}

func (w *World) finishStep() error {
	input, deathFinished := w.stepContinuation.input, w.stepContinuation.deathFinished
	if !w.stepContinuation.pooledDone {
		if err := w.advancePooledProjectiles(input); err != nil {
			return err
		}
		if w.ScreenClearFrames != 0 {
			return nil
		}
		w.stepContinuation.pooledDone = true
	}
	if w.stepContinuation.unbound != nil || w.hasUnboundProjectiles() {
		if err := w.advanceUnboundProjectiles(input); err != nil {
			return err
		}
		if w.ScreenClearFrames != 0 {
			return nil
		}
	}
	if w.Weapons != nil {
		w.Weapons.Compact()
	}
	if err := w.advanceActorPhase(ActorPoolScenery, input); err != nil {
		return err
	}
	w.captureActorRenderTerrain()
	w.advanceTimedEquipment()
	if w.Dive.Phase != 0 || !w.PlayerAlive {
		if w.MaterializationFrames < 16 {
			w.MaterializationFrames++
		}
	} else if w.MaterializationFrames > 0 {
		w.MaterializationFrames--
	}
	if err := w.fire.Tick(input.Fire); err != nil {
		return err
	}
	if w.Weapons != nil {
		if err := w.Weapons.AdvanceSparks(w.weaponContext(input, false)); err != nil {
			return err
		}
		w.Weapons.Compact()
	}
	var spawnErr error
	w.cursor.Activate(w.ScrollY, w.Level.Encounters,
		func(wave visualassets.Wave) {
			if spawnErr == nil {
				spawnErr = w.spawnWave(wave)
			}
		}, w.spawnFixed)
	if spawnErr != nil {
		return spawnErr
	}
	if input.Dive && w.PlayerAlive {
		if w.Dive.Request(&w.Equipment.DiveCharges) {
			w.SoundRequests[2] = "synthesized-effect-16"
		}
	}
	scroll := ScrollState{Y: w.ScrollY, Minimum: w.MinimumScrollY, Maximum: w.MaximumScrollY, DeviationPasses: w.ScrollDeviationPasses}
	scroll.Advance(w.Player.ScrollStep, w.BaseScrollStep, input.Motion.Down)
	w.ScrollDelta, w.ScrollY = scroll.ActualStep, scroll.Y
	w.MaximumScrollY, w.VisitedScrollY = scroll.Maximum, scroll.Maximum
	w.ScrollDeviationPasses = scroll.DeviationPasses
	w.compactActors()
	if w.poolError != nil {
		return w.poolError
	}
	if deathFinished {
		w.shipLossCompleted = true
		if !w.Cheats.InfiniteLives {
			w.Equipment.Lives--
		}
		w.Equipment.Shield = 39
		w.Equipment.FireAdvance = 1
		if w.Equipment.Lives == 0 {
			w.GameOver = true
		} else {
			if !w.deferCheckpointRestart {
				w.RestartCheckpoint()
			}
			w.Ready = true
		}
	}
	w.stepContinuation = worldStepContinuation{}
	return nil
}
