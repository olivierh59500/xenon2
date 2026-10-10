package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestTurnBackgroundMatchesOriginalSwapOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local original turn-background reference not supplied")
	}
	file, err := os.Open(filepath.Join(root, "turn-background.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(file).ReadAll()
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 81 {
		t.Fatalf("incomplete original turn reference: %d cases", len(rows)-1)
	}
	for _, row := range rows[1:] {
		if len(row) != 8 {
			t.Fatal("invalid original turn row")
		}
		var values [8]int
		for i, field := range row {
			values[i], err = strconv.Atoi(field)
			if err != nil {
				t.Fatal(err)
			}
		}
		s, err := NewSession(stageData(t, values[0]), 2, NewRandomState())
		if err != nil {
			t.Fatal(err)
		}
		s.Current = values[1]
		outgoing, incoming := s.ActiveWorld(), s.Players[s.Current^1]
		outgoing.BackgroundY, outgoing.Frame = values[2], uint64(values[3])
		incoming.BackgroundY, incoming.PreviousBackgroundY, incoming.Frame = (values[2]+97)%192, (values[2]+113)%192, uint64(values[4])
		if !s.switchTurn() || s.Current != values[5] || incoming.BackgroundY != values[6] || incoming.PreviousBackgroundY != values[6] || incoming.Frame != uint64(values[7]) {
			t.Fatalf("original turn%v: current%d background%d previous%d frame%d", values, s.Current, incoming.BackgroundY, incoming.PreviousBackgroundY, incoming.Frame)
		}
	}
	t.Logf("Matched shared background and independent player frame counters in %d original swaps across all five levels", len(rows)-1)
}

func TestSessionBackgroundContinuesAcrossAllTurnAdmissionRoutes(t *testing.T) {
	for _, route := range []string{"ship-loss", "accepted-continue", "declined-continue", "completed-stage"} {
		t.Run(route, func(t *testing.T) {
			s, err := NewSession(stageData(t, 1), 2, NewRandomState())
			if err != nil {
				t.Fatal(err)
			}
			outgoing, incoming := s.Players[0], s.Players[1]
			outgoing.BackgroundY = 137
			incoming.BackgroundY, incoming.PreviousBackgroundY, incoming.Frame = 23, 22, 14
			switch route {
			case "ship-loss":
				finishNonfinalShip(t, s)
			case "accepted-continue":
				outgoing.GameOver, outgoing.Equipment.Lives, outgoing.ContinueCredits = true, 0, 1
				if !s.AcceptContinue() {
					t.Fatal("ordinary continue was refused")
				}
			case "declined-continue":
				outgoing.GameOver, outgoing.Equipment.Lives = true, 0
				if !s.DeclineContinue() {
					t.Fatal("ordinary decline did not admit the opponent")
				}
			case "completed-stage":
				outgoing.LevelFinished = true
				transition, err := s.CompleteStage(stageData(t, 2))
				if err != nil || transition != WaitForOtherPlayer {
					t.Fatalf("shared stage gate: transition%d error%v", transition, err)
				}
			}
			phase := outgoing.BackgroundY
			if s.Current != 1 || s.ActiveWorld() != incoming || !incoming.Ready || incoming.BackgroundY != phase || incoming.PreviousBackgroundY != phase || incoming.Frame != 14 {
				t.Fatalf("turn admission resumed stale background or player frame: current%d phase%d/%d previous%d frame%d", s.Current, incoming.BackgroundY, phase, incoming.PreviousBackgroundY, incoming.Frame)
			}
			if _, err := s.Advance(Input{Fire: true}); err != nil {
				t.Fatal(err)
			}
			if incoming.Ready || incoming.BackgroundY != phase || incoming.Frame != 14 {
				t.Fatal("READY acknowledgement advanced the background or frame counter")
			}
			if _, err := s.Advance(Input{}); err != nil {
				t.Fatal(err)
			}
			if incoming.BackgroundY != (phase+191)%192 || incoming.PreviousBackgroundY != phase || incoming.Frame != 15 {
				t.Fatal("first incoming pass did not continue the shared phase with its saved frame parity")
			}
		})
	}
}
