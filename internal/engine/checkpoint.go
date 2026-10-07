package engine

// CheckpointState stores the original restart marker, wallet and seven weapon
// entries. Shield, speed and the shared firing clock are separate game state.
type CheckpointState struct {
	ScrollY, PlayerX, WorldY int
	Money                    int
	Loadout                  WeaponLoadout
}

func (w *World) captureCheckpoint(scrollY, playerX, worldY int) {
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX, w.Checkpoint.WorldY = scrollY, playerX, worldY
	w.Checkpoint.Money = w.Money
	if w.Equipment.SuperFrames == 0 {
		w.Checkpoint.Loadout = w.Equipment.WeaponLoadout
	}
}

// RestoreCheckpointLoadout runs the original initializer order before restoring
// each power tier. Replacing the loadout directly would miss shared-clock changes.
func (e *Equipment) RestoreCheckpointLoadout(loadout WeaponLoadout) {
	e.RestoreSuperLoadout()
	e.WeaponLoadout = WeaponLoadout{}
	for _, saved := range []WeaponSlot{loadout.Primary, loadout.Mounts[0], loadout.Mounts[1], loadout.Mounts[2], loadout.Mounts[3], loadout.Rear, loadout.Side} {
		if saved.Item == ItemNone {
			continue
		}
		serial := e.NextWeaponSerial
		e.ApplyItem(saved.Item)
		for _, slot := range e.slots() {
			if slot.Item == saved.Item && slot.Serial > serial {
				slot.Tier = saved.Tier
				break
			}
		}
	}
	e.FireAdvance = 1
	e.Shield = 39
	e.BeginSuperLoadout()
}

// RestartCheckpoint restores the independent state after the game director's
// death and ready screens. The mutable level map is deliberately retained.
func (w *World) RestartCheckpoint() {
	scrollChange := w.ScrollY - w.Checkpoint.ScrollY
	w.Equipment.RestoreCheckpointLoadout(w.Checkpoint.Loadout)
	w.Money = w.Checkpoint.Money
	w.Player = PlayerMotionState{X: w.Checkpoint.PlayerX, Y: 176, SpeedTier: w.Equipment.SpeedTier}
	w.PreviousPlayer = w.Player
	w.ScrollY, w.PreviousScrollY, w.RenderScrollY = w.Checkpoint.ScrollY, w.Checkpoint.ScrollY, w.Checkpoint.ScrollY
	w.MaximumScrollY, w.VisitedScrollY = w.ScrollY, w.ScrollY
	w.cursor = RestartEncounterCursor(w.ScrollY)
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.Dive = DiveState{}
	w.PlayerAlive = true
	w.PlayerSprite = ""
	w.MaterializationFrames = 8
	w.Actors, w.Projectiles, w.SmallShots, w.Collectibles = nil, nil, nil, nil
	if w.firstGuardianActor != nil && !w.FirstGuardian.Defeated {
		w.Actors = append(w.Actors, w.firstGuardianActor)
		if w.FirstGuardianSegments != nil && w.FirstGuardianSegments.Alive {
			var segments []*WorldActor
			for i, segment := range w.firstGuardianParts {
				segment.Y += float64(scrollChange)
				segment.PreviousY = segment.Y
				w.FirstGuardianSegments.Pieces[i].Y += int32(scrollChange) << 16
				segments = append(segments, segment)
			}
			w.Actors = append(segments, w.Actors...)
		}
	}
	if w.secondGuardianActor != nil && w.secondGuardianActor.Active {
		w.Actors = append(w.Actors, w.secondGuardianActor)
	}
	w.restoreSecondArenaActors()
	w.restoreThirdMiddleActors()
	w.restoreFourthGuardianActors(scrollChange)
	w.restoreFifthGuardianActors(scrollChange)
	if w.FirstMiddle != nil {
		w.FirstMiddle.Updated = [5]bool{}
		w.FirstMiddle.GateCounters = [16]int{}
	}
	if err := w.restoreCheckpointPool(); err != nil {
		w.poolError = err
	}
	w.fire = NewFireCadence(w.Equipment)
	if w.Weapons != nil {
		w.Weapons.ResetProjectiles()
	}
	w.PendingExitDrops = 0
	w.ExitReady = false
	w.ShopReady = false
	w.LevelFinished = false
	clear(w.WaveBonuses.Entries[:])
}
