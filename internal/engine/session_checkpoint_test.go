package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestCheckpointAdmissionNativeStateOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local incoming checkpoint reference not supplied")
	}
	f, err := os.Open(filepath.Join(root, "checkpoint-admission-trace.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows[1:] {
		n := func(i int) int {
			value, err := strconv.Atoi(row[i])
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
		w := testWorld(t)
		w.Checkpoint.PlayerX = n(0)
		w.Checkpoint.ScrollY = 1000
		w.Checkpoint.Money = 222
		w.Player.X, w.Player.Y, w.ScrollY, w.Money = 100, 70, 777, 7777
		w.Equipment.Shield, w.Equipment.FireAdvance, w.InvulnerableFrames, w.Equipment.SuperFrames = n(1), n(2), n(3), n(4)
		w.RestartCheckpoint()
		if w.Player.X != n(5) || w.Player.Y != n(6) || w.ScrollY != n(7) || w.MaximumScrollY != n(8) || w.Money != n(9) || w.Equipment.Shield != n(10) || w.Equipment.FireAdvance != n(11) || w.InvulnerableFrames != n(12) || w.Equipment.SuperFrames != n(13) {
			t.Fatalf("checkpoint admission Go player%+v equipment%+v money%d invuln%d native%v", w.Player, w.Equipment, w.Money, w.InvulnerableFrames, row)
		}
	}
	if len(rows)-1 != 48 {
		t.Fatal("incomplete incoming checkpoint states")
	}
	t.Logf("Compared %d original incoming checkpoint position, wallet and retained timer states.", len(rows)-1)
}

func finishNonfinalShip(t *testing.T, s *Session) {
	t.Helper()
	w := s.ActiveWorld()
	w.Ready = false
	w.PlayerAlive = false
	w.MaterializationFrames = 16
	w.deathAnimation = visualassets.NamedActorAnimation{Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Duration: 0, Sprite: "death"}}}}
	w.deathState = NewAnimation(w.deathAnimation.Animation)
	if _, err := s.Advance(Input{}); err != nil {
		t.Fatal(err)
	}
}

func TestAlternatingShipLossRestoresIncomingTurnExactlyOnce(t *testing.T) {
	s, err := NewSession(testWorld(t).Level, 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	for i, w := range s.Players {
		w.Checkpoint.PlayerX = 100 + i*40
		w.Checkpoint.ScrollY = 3000 + i*100
		w.Checkpoint.Money = 200 + i*100
		w.Equipment.ApplyItem(ItemCannon)
		w.Checkpoint.Loadout = w.Equipment.WeaponLoadout
	}
	for turn := 0; turn < 4; turn++ {
		outgoing := s.Current
		w := s.ActiveWorld()
		beforeSerial := w.Equipment.NextWeaponSerial
		w.Player.X = 200
		w.ScrollY = 2800
		w.Money = 9000
		finishNonfinalShip(t, s)
		if s.Current != outgoing^1 {
			t.Fatal("nonfinal loss did not alternate")
		}
		if w.Player.X != 200 || w.ScrollY == w.Checkpoint.ScrollY || w.Money != 9000 || w.Equipment.NextWeaponSerial != beforeSerial {
			t.Fatal("outgoing turn restored its ship, cash or guns before admission")
		}
		incoming := s.ActiveWorld()
		if incoming.Player.X != incoming.Checkpoint.PlayerX || incoming.ScrollY != incoming.Checkpoint.ScrollY || incoming.Money != incoming.Checkpoint.Money || !incoming.Ready || !incoming.PlayerAlive {
			t.Fatal("returning player did not restore its saved checkpoint")
		}
	}
}

func TestShipLossWithNoAdmissibleOpponentRestartsCurrentTurn(t *testing.T) {
	s, err := NewSession(testWorld(t).Level, 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	s.Players[1].GameOver = true
	s.Players[1].Equipment.Lives = 0
	w := s.ActiveWorld()
	w.Checkpoint.PlayerX = 120
	w.Checkpoint.ScrollY = 3000
	w.Player.X = 200
	w.ScrollY = 2800
	finishNonfinalShip(t, s)
	if s.Current != 0 || w.Player.X != 120 || w.ScrollY != 3000 || !w.Ready || !w.PlayerAlive {
		t.Fatal("sole surviving player did not restart its own checkpoint")
	}
}

func TestCompletedPlayerAdmissionRestoresAliveCheckpointProperties(t *testing.T) {
	s, err := NewSession(testWorld(t).Level, 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := s.Players[0]
	s.Completed[0] = true
	w.LevelFinished = true
	w.Checkpoint.PlayerX = 140
	w.Checkpoint.ScrollY = 1000
	w.Checkpoint.Money = 222
	w.Player.X = 200
	w.ScrollY = 416
	w.Money = 7777
	w.Equipment.Shield = 17
	w.Equipment.FireAdvance = 3
	s.Current = 1
	s.ActiveWorld().GameOver = true
	s.ActiveWorld().Equipment.Lives = 0
	if !s.DeclineContinue() || s.Current != 0 || !w.LevelFinished || !w.Ready || w.Player.X != 140 || w.ScrollY != 1000 || w.Money != 222 || w.Equipment.Shield != 17 || w.Equipment.FireAdvance != 3 {
		t.Fatal("completed survivor lost incoming checkpoint or alive equipment properties")
	}
}

func TestAlternatingCheckpointRetainsGuardianSlotsAndSingleCameraShiftOptional(t *testing.T) {
	s, err := NewSession(originalWorldData(t, 5), 2, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := s.ActiveWorld()
	if err := w.activateFifthGuardian(visualassets.FixedEncounter{Y: 2336}, false); err != nil {
		t.Fatal(err)
	}
	w.Checkpoint.ScrollY = 4600
	beforeY := w.FifthMiddle.Parts[0].Y
	bodySlot := w.fifthMiddleActors[0].Binding.Slot
	w.damageFifthGuardian(w.fifthMiddleActors[1], 40)
	w.beginInvulnerability()
	remaining := w.InvulnerableFrames
	if !s.switchTurn() {
		t.Fatal("second player unavailable")
	}
	if w.ScrollY != 4608 || w.FifthMiddle.Parts[0].Y != beforeY+8 || w.fifthMiddleActors[0].Binding.Slot != bodySlot {
		t.Fatal("outgoing cleanup reset the camera or lost the guardian binding")
	}
	for _, actor := range w.Actors {
		if actor.invulnerability != nil {
			t.Fatal("source turn cleanup retained its aura actor")
		}
	}
	if w.InvulnerableFrames != remaining {
		t.Fatal("source turn cleanup changed the retained invulnerability timer")
	}
	if !s.switchTurn() {
		t.Fatal("first player unavailable")
	}
	if w.ScrollY != 4600 || w.FifthMiddle.Parts[0].Y != beforeY+8 || w.fifthMiddleActors[0].Binding.Slot != bodySlot || !w.FifthMiddle.Parts[1].Destroyed {
		t.Fatal("incoming restore applied a second guardian camera shift or reset damage")
	}
}

func TestSinglePlayerContinueRestoresOnceAndKeepsStageData(t *testing.T) {
	s, err := NewSession(testWorld(t).Level, 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := s.ActiveWorld()
	w.Equipment.ApplyItem(ItemCannon)
	w.Checkpoint.Loadout = w.Equipment.WeaponLoadout
	w.Checkpoint.PlayerX = 120
	w.Checkpoint.ScrollY = 1500
	w.Checkpoint.Money = 200
	w.Level.Terrain.Map[100] = 77
	w.GameOver = true
	w.Equipment.Lives = 0
	w.Score = 9000
	w.DisplayScore = 9000
	serial := w.Equipment.NextWeaponSerial
	if !s.AcceptContinue() || s.Current != 0 || !w.Ready || !w.PlayerAlive || w.Equipment.Lives != 3 || w.Score != 0 || w.DisplayScore != 0 || w.ContinueCredits != 1 {
		t.Fatal("single-player continue did not restore its source game state")
	}
	if w.Player.X != 120 || w.ScrollY != 1500 || w.Money != 200 || w.Level.Terrain.Map[100] != 77 || w.Equipment.NextWeaponSerial != serial+2 {
		t.Fatal("continue lost its map/checkpoint or initialized guns more than once")
	}
}
