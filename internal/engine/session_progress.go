package engine

import "fmt"

type StageTransition uint8

const (
	WaitForOtherPlayer StageTransition = iota
	LoadedNextStage
)

type ResumeRoute uint8

const (
	ResumeGameplay ResumeRoute = iota
	ResumeNextStage
	ResumeMerchantEnding
)

// AfterReady keeps saved completed turns in the original admission sequence.
// An opponent's final loss can reopen the surviving player's completed stage;
// READY precedes the next load, or the merchant ending on the fifth stage.
func (s *Session) AfterReady() ResumeRoute {
	if !s.Completed[s.Current] {
		return ResumeGameplay
	}
	if s.ActiveWorld().Level.Number == 5 {
		return ResumeMerchantEnding
	}
	return ResumeNextStage
}

// CompleteStage follows the original shared-level gate. A surviving second
// player finishes the same level before either player receives a fresh map.
func (s *Session) CompleteStage(next LevelData) (StageTransition, error) {
	current := s.ActiveWorld()
	if !current.LevelFinished {
		return WaitForOtherPlayer, fmt.Errorf("stage guardian has not been defeated")
	}
	if !s.Completed[s.Current] || s.completedReentry {
		if current.Level.Number == 5 {
			current.finishFifthStage()
		}
		s.Completed[s.Current] = true
	}
	s.completedReentry = false
	current.ShopReady = false
	other := s.Current ^ 1
	if s.PlayerCount == 2 && s.Players[other].Equipment.Lives > 0 && !s.Players[other].GameOver && !s.Completed[other] {
		if !s.switchTurn() {
			return WaitForOtherPlayer, fmt.Errorf("remaining player cannot enter its turn")
		}
		return WaitForOtherPlayer, nil
	}
	wanted := current.Level.Number%5 + 1
	if next.Number != wanted {
		return LoadedNextStage, fmt.Errorf("next stage must be %d", wanted)
	}
	difficulty := max(1, s.Difficulty)
	if current.Level.Number == 5 {
		difficulty++
	}
	random := s.ActiveWorld().RandomState()
	replacements := s.Players
	pool := NewActorPool()
	for index, old := range s.Players {
		if old == nil {
			continue
		}
		world, err := advanceStageWorldUsingPool(old, next, random, difficulty, pool, false)
		if err != nil {
			return LoadedNextStage, err
		}
		replacements[index] = world
		if index+1 < s.PlayerCount {
			pool = newActorPoolView(pool)
		}
	}
	for _, world := range replacements {
		if world == nil {
			continue
		}
		if err := world.initializeLevelActors(); err != nil {
			return LoadedNextStage, err
		}
		world.captureRenderTerrain()
		world.captureActorRenderTerrain()
	}
	s.Players, s.Completed, s.Difficulty = replacements, [2]bool{}, difficulty
	s.hasPlayed = [2]bool{true, s.PlayerCount == 2}
	// The shared-level initializer clears both completion flags before the
	// normal turn admission. Two surviving players therefore alternate here,
	// rather than leaving the player who completed last in control.
	if s.PlayerCount == 2 && s.Players[other].Equipment.Lives > 0 && !s.Players[other].GameOver {
		s.Current = other
	}
	return LoadedNextStage, nil
}

func advanceStageWorld(old *World, data LevelData, random RandomState, difficulty int) (*World, error) {
	return advanceStageWorldUsingPool(old, data, random, difficulty, NewActorPool(), true)
}

func advanceStageWorldUsingPool(old *World, data LevelData, random RandomState, difficulty int, pool *ActorPool, initializeActors bool) (*World, error) {
	equipment := old.Equipment
	equipment.RestoreSuperLoadout()
	loadout := equipment.WeaponLoadout
	shield, advance := equipment.Shield, equipment.FireAdvance
	equipment.RestoreCheckpointLoadout(loadout)
	// Death resets these properties separately. The level initializer does not.
	equipment.Shield, equipment.FireAdvance = shield, advance
	data.InitialEquipment, data.InitialRandom = &equipment, &random
	if data.Rules != nil {
		rules := *data.Rules
		rules.OrdinaryHealthMultiplier *= difficulty
		rules.StrongHealthMultiplier *= difficulty
		data.Rules = &rules
	}
	world, err := newWorldWithPool(data, pool, initializeActors)
	if err != nil {
		return nil, err
	}
	world.Score, world.DisplayScore = old.Score, old.DisplayScore
	world.ContinueCredits = old.ContinueCredits
	world.SetCheats(old.Cheats)
	world.Ready = true
	world.GameOver = old.GameOver
	world.PlayerAlive = !old.GameOver
	world.Checkpoint.Loadout = loadout
	return world, nil
}

// finishFifthStage records the original victory credit and basic-weapon reset.
// A live Nashwan suite restores its recorded regular weapons during cleanup.
func (w *World) finishFifthStage() {
	w.ContinueCredits++
	if w.Equipment.SuperLoadoutActive {
		w.Equipment.RestoreSuperLoadout()
	} else {
		w.Equipment.WeaponLoadout = WeaponLoadout{}
		w.Equipment.EnsureBasicWeapon()
	}
	w.Checkpoint.Loadout = w.Equipment.WeaponLoadout
}
