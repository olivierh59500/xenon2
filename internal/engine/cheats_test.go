package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestTrainerEnergySuppressesDamageButStillAllowsTerrainCrushing(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		w := testWorld(t)
		w.Equipment.Shield = 16
		w.SetCheats(CheatOptions{InfiniteEnergy: enabled})
		w.damagePlayer(16)
		if w.PlayerAlive != enabled || enabled && w.Equipment.Shield != 16 || !enabled && w.Equipment.Shield != 0 {
			t.Fatalf("energy trainer changed the wrong damage boundary: enabled%v alive%v shield%d", enabled, w.PlayerAlive, w.Equipment.Shield)
		}
	}
	w := testWorld(t)
	w.SetCheats(CheatOptions{InfiniteEnergy: true})
	w.Rewind.Timer = -17
	w.Level.PlayerStencil = &visualassets.PlayerTerrainStencil{Width: 1, Height: 1, Rows: []uint32{0x80000000}}
	w.Coverage = &TerrainCoverage{Columns: 20, Rows: 300, Map: make([]uint16, 6000), coverage: make(map[uint16][16]uint16)}
	var rows [16]uint16
	for i := range rows {
		rows[i] = 0xffff
	}
	w.Coverage.coverage[1] = rows
	w.Coverage.Map[((w.ScrollY+w.Player.Y)/16)*20+w.Player.X/16] = 1
	w.Level.Terrain.Map = w.Coverage.Map
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.PlayerAlive || w.Equipment.Shield != 0 {
		t.Fatal("energy trainer incorrectly disabled the original terrain crush")
	}
}

func TestTrainerLivesPreservesShipsButStillAlternatesTurns(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		s, err := NewSession(testWorld(t).Level, 1, NewRandomState())
		if err != nil {
			t.Fatal(err)
		}
		s.SetCheats(CheatOptions{InfiniteLives: enabled})
		w := s.ActiveWorld()
		w.Equipment.Lives = 1
		finishNonfinalShip(t, s)
		if enabled && (w.Equipment.Lives != 1 || w.GameOver || !w.Ready || !w.PlayerAlive) || !enabled && (w.Equipment.Lives != 0 || !w.GameOver) {
			t.Fatalf("ship completion differs from selected trainer: enabled%v ships%d gameover%v ready%v", enabled, w.Equipment.Lives, w.GameOver, w.Ready)
		}
	}
	s, err := NewSession(testWorld(t).Level, 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	s.SetCheats(CheatOptions{InfiniteLives: true})
	for _, w := range s.Players {
		w.Equipment.Lives = 1
	}
	for turn := 0; turn < 8; turn++ {
		outgoing := s.Current
		finishNonfinalShip(t, s)
		if s.Current != outgoing^1 || !s.ActiveWorld().Ready || s.Players[outgoing].Equipment.Lives != 1 || s.Players[outgoing].GameOver {
			t.Fatalf("infinite ships stopped original alternating admission at loss%d", turn)
		}
	}
}

func TestTrainerCreditsPreservesNormalContinueAdmission(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		w := testWorld(t)
		w.SetCheats(CheatOptions{InfiniteCredits: enabled})
		for offer := 0; offer < 5; offer++ {
			w.GameOver, w.PlayerAlive = true, false
			w.Equipment.Lives, w.Score, w.DisplayScore = 0, 500, 500
			accepted := w.AcceptContinue()
			want := enabled || offer < 2
			if accepted != want {
				t.Fatalf("continue%d enabled%v: accepted%v want%v", offer, enabled, accepted, want)
			}
			if accepted && (w.GameOver || !w.Ready || w.Equipment.Lives != 3 || w.Score != 0 || w.DisplayScore != 0) {
				t.Fatal("trainer continue bypassed the normal three-ship/score/checkpoint admission")
			}
		}
		wantCredits := 0
		if enabled {
			wantCredits = 2
		}
		if w.ContinueCredits != wantCredits {
			t.Fatalf("continue debit is%d, want%d", w.ContinueCredits, wantCredits)
		}
	}
}

func TestTrainerMoneyUsesOriginalEntryGrantAndPaidTransactions(t *testing.T) {
	w := testWorld(t)
	w.Money = 1234
	rules := ShopRules{Level: 1, StockLimit: 600}
	random := w.RandomState()
	if ordinary := w.PrepareShop(rules); ordinary != rules || w.Money != 1234 {
		t.Fatal("disabled trainer changed the real stock or wallet")
	}
	w.SetCheats(CheatOptions{InfiniteMoney: true})
	if w.Money != 1234 {
		t.Fatal("money trainer granted cash outside its original shop entry")
	}
	trained := w.PrepareShop(rules)
	if rules.StockLimit != 600 || trained.StockLimit != 30000 || trained.Level != 1 || w.Money != 5000000 {
		t.Fatal("money trainer lost its original wallet/stock grant")
	}
	if paid, err := trained.Buy(&w.Equipment, &w.Money, ItemProtection); err != nil || paid != 6000 || w.Money != 4994000 || !w.Equipment.Protection {
		t.Fatalf("trainer purchase became free or changed original price: paid%d cash%d error%v", paid, w.Money, err)
	}
	for range 4 {
		w.Equipment.ApplyItem(ItemCannon)
	}
	before, cash := w.Equipment, w.Money
	if _, err := trained.Buy(&w.Equipment, &w.Money, ItemLaser); err == nil || w.Equipment != before || w.Money != cash {
		t.Fatal("money trainer bypassed normal equipment compatibility")
	}
	w.PrepareShop(rules)
	if w.Money != 5000000 || w.RandomState() != random {
		t.Fatal("later merchant entry failed its fixed grant or consumed gameplay randomness")
	}
}

func TestTrainerOptionsSurviveAlternatingPlayersAndStageReplacement(t *testing.T) {
	s, err := NewSession(stageData(t, 1), 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	options := CheatOptions{InfiniteLives: true, InfiniteCredits: true, InfiniteMoney: true, InfiniteEnergy: true, KeyFunctions: true}
	s.SetCheats(options)
	for _, w := range s.Players {
		if w.Cheats != options {
			t.Fatal("trainer did not reach both saved games")
		}
	}
	// This is stage-admission coverage, not a guardian-victory playthrough.
	s.Completed = [2]bool{true, true}
	s.ActiveWorld().LevelFinished = true
	if transition, err := s.CompleteStage(stageData(t, 2)); err != nil || transition != LoadedNextStage {
		t.Fatalf("stage replacement failed: transition%d error%v", transition, err)
	}
	for _, w := range s.Players {
		if w.Cheats != options || w.Level.Number != 2 {
			t.Fatal("stage replacement lost the selected trainer options")
		}
	}
	s.SetCheats(CheatOptions{})
	for _, w := range s.Players {
		if w.Cheats.Enabled() {
			t.Fatal("explicitly disabled trainer remained active")
		}
	}
}

func TestOptionalKeyEquipmentFunctionsAreGatedAndKeepTheWallet(t *testing.T) {
	w := testWorld(t)
	w.Money = 4321
	before := w.Equipment
	if w.ApplyCheatItem(ItemExtraLife) || w.Equipment != before {
		t.Fatal("disabled key functions granted a ship")
	}
	w.SetCheats(CheatOptions{KeyFunctions: true})
	if !w.ApplyCheatItem(ItemExtraLife) || w.Equipment.Lives != 4 || !w.ApplyCheatItem(ItemBitmapShades) || w.Equipment.ShadesFrames != 220 {
		t.Fatal("selected keyboard equipment initializers differ from the original functions")
	}
	if !w.ApplyCheatItem(ItemSuperNashwan) || w.Equipment.SuperFrames != 170 || !w.Equipment.SuperLoadoutActive || w.Equipment.Primary.Item != ItemDoubleShot || w.Equipment.Primary.Tier != 2 {
		t.Fatal("keyboard Nashwan did not install its immediate temporary suite")
	}
	if w.Money != 4321 || w.ApplyCheatItem(ItemAdvice) || w.ApplyCheatItem(Item(255)) {
		t.Fatal("keyboard functions spent cash or admitted a nonexistent initializer")
	}
}
