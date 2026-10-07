package presentation

import (
	"testing"
	"xenon2/internal/visualassets"
)

func directorData() *visualassets.Presentation {
	p := &visualassets.Presentation{Ready: " GET READY PLAYER 1 ", GameOver: "     GAME OVER      ", HighScoreHeading: "     HIGH SCORE     ", ContinueHeading: "   CONTINUE GAME    ", ContinueCounter: "         9          ", TextZoomSteps: []int{1, 2, 16, 16, -1}, AppearSteps: []int{1, 2, 16, -1}, DisappearSteps: []int{16, 8, 0, -1}, MessageSteps: []int{1, 16, 17, 16, 0, -1}}
	for i := 0; i < 12; i++ {
		p.Credits = append(p.Credits, "CREDIT")
	}
	return p
}

func TestReadyDirectorWaitsForFireAtOriginalSentinel(t *testing.T) {
	d := NewDirector(directorData())
	d.BeginReady(2)
	d.Advance(Input{})
	d.Advance(Input{})
	for i := 0; i < 50; i++ {
		if result := d.Advance(Input{}); result != NoResult {
			t.Fatal("ready screen advanced without fire")
		}
	}
	if d.Step != 2 || d.Captions[0].Text != " GET READY PLAYER 2 " {
		t.Fatal("ready sentinel or player number changed")
	}
	d.Advance(Input{Confirm: true})
	for i := 0; i < 4; i++ {
		if d.Advance(Input{}) == ResumeGame {
			return
		}
	}
	t.Fatal("confirmed ready screen did not finish its exit")
}

func TestContinueDirectorOriginalCountdownAndExit(t *testing.T) {
	d := NewDirector(directorData())
	d.BeginContinue()
	for d.Phase != ContinueHold {
		d.Advance(Input{})
	}
	if d.Countdown != 9 {
		t.Fatal("continue does not start at nine")
	}
	for i := 0; i < 7; i++ {
		d.Advance(Input{})
	}
	if d.Countdown != 9 {
		t.Fatal("countdown does not retain eight passes per digit")
	}
	d.Advance(Input{})
	if d.Countdown != 8 {
		t.Fatal("countdown does not progress after eight passes")
	}
	d.Advance(Input{Confirm: true})
	for i := 0; i < 5; i++ {
		if d.Advance(Input{}) == ContinueAccepted {
			return
		}
	}
	t.Fatal("continue confirmation did not produce accepted result")
}

func TestHighScoreInsertionKeepsTenRowsAndThreeInitials(t *testing.T) {
	d := NewDirector(directorData())
	if !d.InsertScore(500) {
		t.Fatal("positive score rejected")
	}
	for d.Phase != Initials {
		d.Advance(Input{})
	}
	for i := 0; i < 3; i++ {
		d.Advance(Input{Horizontal: 1})
		d.Advance(Input{Confirm: true})
	}
	if d.Scores[0].Points != 500 || d.Scores[0].Initials != "BBB" || d.Phase != InitialsHold {
		t.Fatalf("wrong inserted score %+v", d.Scores[0])
	}
	if !d.InsertScore(1000) || d.Scores[1].Points != 500 {
		t.Fatal("score ordering not preserved")
	}
}

func TestAttractDirectorRunsAllCreditPairsBeforeScores(t *testing.T) {
	d := NewDirector(directorData())
	seen := [6]bool{}
	for frame := 0; frame < 1000; frame++ {
		if d.Phase == Credits {
			seen[d.CreditPair] = true
		}
		if result := d.Advance(Input{}); result == AttractComplete {
			for pair, visible := range seen {
				if !visible {
					t.Fatalf("credit pair %d omitted", pair)
				}
			}
			if d.Phase != LogoDelay {
				t.Fatal("attract does not return to logo delay")
			}
			return
		}
	}
	t.Fatal("attract loop did not finish")
}
