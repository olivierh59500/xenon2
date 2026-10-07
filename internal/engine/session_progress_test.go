package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func stageData(t *testing.T, number int) LevelData {
	t.Helper()
	data := testWorld(t).Level
	data.Number = number
	data.Rules = &visualassets.LevelRules{Level: number, InitialMaximumScrollY: 4608, OrdinaryHealthMultiplier: 1, StrongHealthMultiplier: 2}
	return data
}

func TestMiddleShopResumesCameraAndSavesPurchasedLoadout(t *testing.T) {
	w := testWorld(t)
	w.ShopReady = true
	w.ScrollY, w.RenderScrollY = 3000, 3001
	w.Player.X = 134
	w.Equipment.ApplyItem(ItemCannon)
	w.Money = 700
	w.ResumeShop()
	if w.ShopReady || w.Ready || w.ScrollY != 3000 || w.RenderScrollY != 3001 || w.Player.X != 134 || w.Money != 700 || w.Checkpoint.Loadout != w.Equipment.WeaponLoadout {
		t.Fatal("middle shop restarted the level or forgot inventory")
	}
}

func TestTwoPlayersFinishBeforeAdvancingWithIndependentState(t *testing.T) {
	s, err := NewSession(stageData(t, 1), 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	first, second := s.Players[0], s.Players[1]
	first.Equipment.ApplyItem(ItemDoubleShot)
	first.Equipment.Primary.Tier = 1
	first.Score = 4320
	first.Equipment.Shield = 17
	first.Equipment.FireAdvance = 3
	first.Money = 800
	first.Level.Terrain.Map[10] = 99
	second.Level.Terrain.Map[10] = 88
	first.LevelFinished = true
	transition, err := s.CompleteStage(stageData(t, 2))
	if err != nil || transition != WaitForOtherPlayer || s.Current != 1 || s.Players[0] != first || s.Players[1] != second {
		t.Fatal("first completion loaded the next level early")
	}
	second.Equipment.ApplyItem(ItemRearShot)
	second.Score = 1200
	second.Money = 300
	second.LevelFinished = true
	transition, err = s.CompleteStage(stageData(t, 2))
	if err != nil || transition != LoadedNextStage || s.Current != 1 {
		t.Fatal("both players did not advance together")
	}
	first, second = s.Players[0], s.Players[1]
	if first.Level.Number != 2 || second.Level.Number != 2 || first.Score != 4320 || second.Score != 1200 || first.Equipment.Primary.Item != ItemDoubleShot || first.Equipment.Primary.Tier != 1 || second.Equipment.Rear.Item != ItemRearShot {
		t.Fatal("stage reconstruction discarded saved player state")
	}
	if first.Equipment.Shield != 17 || first.Equipment.FireAdvance != 3 || first.Money != 0 || second.Money != 0 || first.AdviceIndex != 0 || first.Level.Terrain.Map[10] != 0 || second.Level.Terrain.Map[10] != 0 || !first.Ready || !second.Ready {
		t.Fatal("fresh level boundaries differ from source")
	}
	first.Level.Terrain.Map[10] = 77
	if second.Level.Terrain.Map[10] != 0 {
		t.Fatal("fresh maps share mutable storage")
	}
}

func TestFifthStageLoopsDifficultyAndAwardsOneCreditPerPlayer(t *testing.T) {
	s, err := NewSession(stageData(t, 5), 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	s.Players[0].Equipment.ApplyItem(ItemLaser)
	s.Players[0].LevelFinished = true
	s.CompleteStage(stageData(t, 1))
	if s.Players[0].ContinueCredits != 3 || s.Players[0].Equipment.Primary.Item != ItemForwardShot || s.Players[0].Equipment.Mounts[0].Item != ItemNone {
		t.Fatal("first victory credit or ordinary weapon cleanup changed")
	}
	s.Players[1].LevelFinished = true
	if _, err = s.CompleteStage(stageData(t, 1)); err != nil {
		t.Fatal(err)
	}
	if s.Difficulty != 2 || s.Players[0].ContinueCredits != 3 || s.Players[1].ContinueCredits != 3 || s.Players[0].Level.Rules.OrdinaryHealthMultiplier != 2 || s.Players[0].Level.Rules.StrongHealthMultiplier != 4 {
		t.Fatal("fifth stage did not start the next original difficulty loop")
	}
}

func TestCompletedPlayerWaitsThroughOtherPlayersNonfinalLoss(t *testing.T) {
	s, err := NewSession(stageData(t, 1), 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	s.Completed[0] = true
	s.Current = 1
	world := s.ActiveWorld()
	world.Ready = false
	world.PlayerAlive = false
	world.MaterializationFrames = 16
	world.deathAnimation = visualassets.NamedActorAnimation{Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Duration: 0, Sprite: "death"}}}}
	world.deathState = NewAnimation(world.deathAnimation.Animation)
	changed, err := s.Advance(Input{})
	if err != nil || changed || s.Current != 1 || world.Equipment.Lives != 2 || !world.Ready {
		t.Fatal("a ship loss reopened the already completed player's stage")
	}
}

// This checks stage admission with complete exported resources. It deliberately
// does not claim a guardian victory or a complete gameplay playthrough.
func TestOriginalResourcesAdmitTwoPlayersAcrossAllFiveStagesOptional(t *testing.T) {
	s, err := NewSession(playableOriginalWorldData(t, 1), 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	s.Players[0].Equipment.ApplyItem(ItemDoubleShot)
	s.Players[0].Equipment.Primary.Tier = 2
	for level := 1; level <= 5; level++ {
		next := playableOriginalWorldData(t, level%5+1)
		current := s.Current
		s.Players[current].LevelFinished = true
		state, err := s.CompleteStage(next)
		if err != nil || state != WaitForOtherPlayer {
			t.Fatalf("level%d first player admission:%v %v", level, state, err)
		}
		s.ActiveWorld().LevelFinished = true
		state, err = s.CompleteStage(next)
		if err != nil || state != LoadedNextStage {
			t.Fatalf("level%d shared admission:%v %v", level, state, err)
		}
		for index, w := range s.Players {
			if w.Level.Number != next.Number || w.Coverage == nil || !w.Ready || w.Equipment.Lives != 3 {
				t.Fatalf("level%d player%d lost source game state", level, index)
			}
			w.Ready = false
			if err := w.Step(Input{}); err != nil {
				t.Fatalf("level%d player%d first pass:%v", next.Number, index, err)
			}
		}
	}
	if s.Difficulty != 2 || s.Players[0].ContinueCredits != 3 || s.Players[1].ContinueCredits != 3 {
		t.Fatal("original difficulty loop or victory credits differ")
	}
}
