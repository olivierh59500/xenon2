package presentation

import (
	"fmt"
	"strings"

	"xenon2/internal/visualassets"
)

type Input struct {
	Confirm    bool
	Horizontal int
}
type Result uint8

const (
	NoResult Result = iota
	ShowMenu
	StartGame
	ResumeGame
	ContinueAccepted
	ContinueDeclined
	AttractComplete
	PlayerGameFinished
)

type Phase string

const (
	LogoDelay       Phase = "logo-delay"
	LogoIn          Phase = "logo-in"
	Credits         Phase = "credits"
	LogoOut         Phase = "logo-out"
	ScoresIn        Phase = "scores-in"
	ScoresHold      Phase = "scores-hold"
	ScoresOut       Phase = "scores-out"
	ReadyMessage    Phase = "ready-message"
	GameOverMessage Phase = "game-over-message"
	ContinueIn      Phase = "continue-in"
	ContinueHold    Phase = "continue-hold"
	ContinueOut     Phase = "continue-out"
	InitialsIn      Phase = "initials-in"
	Initials        Phase = "initials"
	InitialsHold    Phase = "initials-hold"
	InitialsOut     Phase = "initials-out"
	MenuIn          Phase = "menu-in"
	MenuOut         Phase = "menu-out"
)

type Caption struct {
	Text    string
	Scale   int
	CenterY int
	Advance int
}
type Score struct {
	Points   int
	Initials string
}

// Director describes source presentation sequencing independently of graphics.
// Control-table sentinels change state without introducing a display pass.
type Director struct {
	CreditStage                                 int
	Data                                        *visualassets.Presentation
	Phase                                       Phase
	DisplayPhase                                Phase
	Tick, Step, CreditPair                      int
	LogoScale                                   int
	Captions                                    []Caption
	Scores                                      [10]Score
	ShowScores                                  bool
	ShowCredits                                 bool
	Countdown                                   int
	Accepted                                    bool
	InitialRow, InitialCharacter, InitialLetter int
	menuAfterLogo                               bool
	initialDirection, initialRepeat             int
}

func NewDirector(data *visualassets.Presentation) *Director {
	d := &Director{Data: data, Phase: LogoDelay, InitialRow: -1}
	for i := range d.Scores {
		d.Scores[i].Initials = ":::"
	}
	return d
}

func (d *Director) BeginReady(player int) {
	d.beginMessage(ReadyMessage, strings.Replace(d.Data.Ready, "1", fmt.Sprint(player), 1))
}

func (d *Director) BeginGameOver() { d.beginMessage(GameOverMessage, d.Data.GameOver) }

func (d *Director) beginMessage(phase Phase, text string) {
	d.Phase, d.Step, d.Tick = phase, 0, 0
	d.Captions = []Caption{{Text: text, Scale: 0, CenterY: 100, Advance: 16}}
	d.LogoScale, d.ShowScores, d.ShowCredits = 0, false, false
}

func (d *Director) BeginContinue() {
	d.Phase, d.Step, d.Tick = ContinueIn, 0, 0
	d.Countdown, d.Accepted = 9, false
	d.LogoScale, d.ShowScores, d.ShowCredits = 0, false, false
	d.Captions = nil
}

func (d *Director) BeginMenu() {
	d.Phase, d.Step = MenuIn, 0
	d.LogoScale, d.ShowScores, d.ShowCredits = 0, false, false
	d.Captions = nil
}

func (d *Director) BeginStart() {
	d.Phase, d.Step = MenuOut, 0
	d.LogoScale, d.ShowScores, d.ShowCredits = 0, false, false
	d.Captions = nil
}

func (d *Director) caption(text string, scale, targetY int) Caption {
	return Caption{Text: text, Scale: scale, CenterY: 100 + ((targetY - 100) * scale >> 4), Advance: 16}
}

// Advance performs one original presentation pass, not one display refresh.
func (d *Director) Advance(input Input) Result {
	result := NoResult
	for {
		d.DisplayPhase = d.Phase
		switch d.Phase {
		case LogoDelay:
			d.LogoScale, d.ShowScores, d.ShowCredits, d.Captions = 0, false, false, nil
			if input.Confirm {
				d.BeginMenu()
				continue
			}
			d.Tick++
			if d.Tick == 34 {
				d.Phase, d.Step = LogoIn, 1
			}
		case LogoIn:
			if input.Confirm {
				d.BeginMenu()
				continue
			}
			d.LogoScale = d.Step
			d.Step++
			if d.Step == 16 {
				d.Phase, d.Step, d.CreditPair, d.CreditStage = Credits, 0, 0, 0
			}
		case Credits:
			if input.Confirm {
				d.menuAfterLogo = true
				d.Phase, d.Step = LogoOut, 16
				input.Confirm = false
				continue
			}
			d.Captions = d.Captions[:0]
			pair := d.CreditPair * 2
			// The preceding phase draws the first line before reading its next
			// table entry, including the single overlapping shrink pass.
			if d.CreditStage == 1 {
				d.Captions = append(d.Captions, Caption{Text: d.Data.Credits[pair], Scale: 16, CenterY: 120, Advance: 16})
			}
			value := d.creditSteps()[d.Step]
			d.Step++
			for value < 0 {
				d.CreditStage++
				if d.CreditStage == 1 {
					d.Captions = append(d.Captions, Caption{Text: d.Data.Credits[pair], Scale: 16, CenterY: 120, Advance: 16})
				}
				if d.CreditStage == 3 {
					d.CreditStage = 0
					d.CreditPair++
					pair = d.CreditPair * 2
				}
				if d.CreditPair == 6 {
					d.Phase, d.Step = LogoOut, 16
					break
				}
				d.Step = 1
				value = d.creditSteps()[0]
			}
			if d.Phase == LogoOut {
				continue
			}
			d.LogoScale = 16
			if value > 0 {
				line, center := pair, 120
				if d.CreditStage == 1 {
					line++
					center = 150
				}
				d.Captions = append(d.Captions, Caption{Text: d.Data.Credits[line], Scale: value, CenterY: center, Advance: 16})
			}
		case LogoOut:
			if input.Confirm {
				d.BeginMenu()
				continue
			}
			d.Captions = nil
			d.LogoScale = d.Step
			d.Step--
			if d.Step == 0 {
				if d.menuAfterLogo {
					d.menuAfterLogo = false
					d.Phase, d.Step = MenuIn, 0
				} else {
					d.Phase, d.Step = ScoresIn, 0
				}
			}
		case ScoresIn, InitialsIn, ContinueIn, MenuIn:
			d.LogoScale = 0
			phase := d.Phase
			value := d.Data.AppearSteps[d.Step]
			if value < 0 {
				d.Step = 0
				switch phase {
				case ScoresIn:
					d.Phase, d.Tick = ScoresHold, 0
				case InitialsIn:
					d.Phase = Initials
				case ContinueIn:
					d.Phase, d.Tick = ContinueHold, 79
				case MenuIn:
					d.Captions = nil
					return ShowMenu
				}
				continue
			}
			d.ShowScores = false
			d.Step++
			text, target := d.Data.HighScoreHeading, 12
			if phase == ContinueIn {
				text, target = d.Data.ContinueHeading, 80
			}
			if phase == MenuIn {
				text = d.Data.MenuHeading
			}
			d.Captions = []Caption{d.caption(text, value, target)}
		case ScoresHold, InitialsHold:
			d.ShowScores = true
			d.Captions = []Caption{d.caption(d.Data.HighScoreHeading, 16, 12)}
			if input.Confirm && d.Phase == ScoresHold {
				d.BeginMenu()
				continue
			}
			d.Tick++
			limit := 85
			if d.Phase == InitialsHold {
				limit = 17
			}
			if d.Tick == limit {
				if d.Phase == InitialsHold {
					d.Phase = InitialsOut
				} else {
					d.Phase = ScoresOut
				}
				d.Step = 0
			}
		case ScoresOut, InitialsOut, ContinueOut, MenuOut:
			phase := d.Phase
			value := d.Data.DisappearSteps[d.Step]
			if value < 0 {
				d.Captions = nil
				switch phase {
				case ContinueOut:
					if d.Accepted {
						return ContinueAccepted
					}
					return ContinueDeclined
				case MenuOut:
					return StartGame
				case InitialsOut:
					d.Phase, d.Tick = LogoDelay, 0
					return AttractComplete
				default:
					d.Phase, d.Tick = LogoDelay, 0
					result = AttractComplete
					continue
				}
			}
			d.ShowScores, d.ShowCredits = false, false
			d.Step++
			text, target := d.Data.HighScoreHeading, 12
			if phase == ContinueOut {
				text, target = d.Data.ContinueHeading, 80
			}
			if phase == MenuOut {
				text = d.Data.MenuHeading
			}
			d.Captions = nil
			if value > 0 {
				d.Captions = []Caption{d.caption(text, value, target)}
			}
		case ReadyMessage, GameOverMessage:
			value := d.Data.MessageSteps[d.Step]
			if value < 0 {
				d.Captions = nil
				if d.Phase == ReadyMessage {
					return ResumeGame
				}
				return PlayerGameFinished
			}
			if value != 17 || d.Phase == GameOverMessage || input.Confirm {
				d.Step++
			}
			d.Captions[0].Scale = value
		case ContinueHold:
			d.Countdown = d.Tick / 8
			text := strings.Replace(d.Data.ContinueCounter, "9", fmt.Sprint(d.Countdown), 1)
			d.Captions = []Caption{d.caption(d.Data.ContinueHeading, 16, 80), d.caption(text, 16, 120)}
			d.ShowCredits = true
			d.Tick--
			if input.Confirm || d.Tick == 0 {
				d.Accepted = input.Confirm
				d.Phase, d.Step = ContinueOut, 0
			}
		case Initials:
			d.ShowScores = true
			d.Captions = []Caption{d.caption(d.Data.HighScoreHeading, 16, 12)}
			if input.Horizontal != d.initialDirection {
				d.initialDirection, d.initialRepeat = input.Horizontal, 0
			}
			if input.Horizontal != 0 {
				if d.initialRepeat == 0 {
					d.InitialLetter = (d.InitialLetter + input.Horizontal + 38) % 38
					d.initialRepeat = 4
				}
				d.initialRepeat--
			}
			letters := []byte(d.Scores[d.InitialRow].Initials)
			letters[d.InitialCharacter] = d.initialAlphabet()[d.InitialLetter]
			d.Scores[d.InitialRow].Initials = string(letters)
			if input.Confirm {
				d.InitialCharacter++
				d.InitialLetter = 0
				d.initialDirection, d.initialRepeat = 0, 0
				if d.InitialCharacter == 3 {
					d.Phase, d.Tick = InitialsHold, 1
				}
			}
		}
		return result
	}
}

func (d *Director) creditSteps() []int {
	switch d.CreditStage {
	case 1:
		if len(d.Data.CreditSecondSteps) > 0 {
			return d.Data.CreditSecondSteps
		}
	case 2:
		if len(d.Data.CreditOutSteps) > 0 {
			return d.Data.CreditOutSteps
		}
		return d.Data.DisappearSteps
	}
	return d.Data.TextZoomSteps
}

func (d *Director) InsertScore(points int) bool {
	row := -1
	for i, score := range d.Scores {
		if points > score.Points {
			row = i
			break
		}
	}
	if row < 0 {
		return false
	}
	copy(d.Scores[row+1:], d.Scores[row:9])
	d.Scores[row] = Score{Points: points, Initials: ":::"}
	d.InitialRow, d.InitialCharacter, d.InitialLetter = row, 0, 0
	d.Phase, d.Step = InitialsIn, 0
	d.ShowScores, d.ShowCredits, d.LogoScale = false, false, 0
	d.Captions = nil
	return true
}

func (d *Director) initialAlphabet() string { return "ABCDEFGHIJKLMNOPQRSTUVWXYZ.:0123456789" }

// AttractMusic identifies the presentation passes that retain the intro score.
func (d *Director) AttractMusic() bool {
	if d.menuAfterLogo {
		return false
	}
	switch d.Phase {
	case LogoDelay, LogoIn, Credits, LogoOut, ScoresIn, ScoresHold, ScoresOut:
		return true
	}
	return false
}
