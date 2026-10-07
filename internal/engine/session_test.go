package engine

import "testing"

func TestSessionChangesTurnsWithoutSharingWeaponsOrWallet(t *testing.T) {
	level := testWorld(t).Level
	s, err := NewSession(level, 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	first := s.ActiveWorld()
	first.Equipment.ApplyItem(ItemDoubleShot)
	first.Money = 1200
	first.NextUIRandom()
	wantRandom := first.RandomState()
	if !s.switchTurn() || s.Current != 1 {
		t.Fatal("second player was not admitted")
	}
	second := s.ActiveWorld()
	if second.Equipment.Primary.Item != ItemForwardShot || second.Money != 0 || second.RandomState() != wantRandom || !second.Ready {
		t.Fatal("second player's inventory or shared RNG differs")
	}
	second.NextUIRandom()
	if !s.switchTurn() || first.RandomState() != second.RandomState() || first.Equipment.Primary.Item != ItemDoubleShot || first.Money != 1200 {
		t.Fatal("returning turn lost independent state or random continuity")
	}
}

func TestSessionFinalLossWaitsForContinueDecision(t *testing.T) {
	s, err := NewSession(testWorld(t).Level, 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	s.Players[0].GameOver = true
	s.Players[0].Equipment.Lives = 0
	if changed, err := s.Advance(Input{}); err != nil || changed || s.Current != 0 {
		t.Fatal("score/continue director must run before changing final-loss turns")
	}
	if !s.DeclineContinue() || s.Current != 1 {
		t.Fatal("declining continue did not admit the remaining player")
	}
}
