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
)

type Phase string

const (
	LogoDelay    Phase = "logo-delay"
	LogoIn       Phase = "logo-in"
	Credits      Phase = "credits"
	LogoOut      Phase = "logo-out"
	ScoresIn     Phase = "scores-in"
	ScoresHold   Phase = "scores-hold"
	ScoresOut    Phase = "scores-out"
	ReadyMessage Phase = "ready-message"
	ContinueIn   Phase = "continue-in"
	ContinueHold Phase = "continue-hold"
	ContinueOut  Phase = "continue-out"
	Initials     Phase = "initials"
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
type Director struct {
	CreditStage                                 int
	Data                                        *visualassets.Presentation
	Phase                                       Phase
	Tick, Step, CreditPair                      int
	LogoScale                                   int
	Captions                                    []Caption
	Scores                                      [10]Score
	ShowScores                                  bool
	Countdown                                   int
	Accepted                                    bool
	InitialRow, InitialCharacter, InitialLetter int
	result                                      Result
}

func NewDirector(data *visualassets.Presentation) *Director {
	d := &Director{Data: data, Phase: LogoDelay, InitialRow: -1}
	for i := range d.Scores {
		d.Scores[i].Initials = ":::"
	}
	return d
}

func (d *Director) BeginReady(player int) {
	d.Phase = ReadyMessage
	d.Step, d.Tick = 0, 0
	text := d.Data.Ready
	text = strings.Replace(text, "1", fmt.Sprint(player), 1)
	d.Captions = []Caption{{Text: text, Scale: 1, CenterY: 100, Advance: 16}}
	d.LogoScale = 16
	d.ShowScores = false
	d.result = NoResult
}
func (d *Director) BeginContinue() {
	d.Phase = ContinueIn
	d.Step, d.Tick = 0, 0
	d.Countdown = 9
	d.Accepted = false
	d.LogoScale = 0
	d.ShowScores = false
	d.result = NoResult
}

func (d *Director) Advance(input Input) Result {
	if d.result != NoResult {
		result := d.result
		d.result = NoResult
		return result
	}
	switch d.Phase {
	case LogoDelay:
		d.Tick++
		if input.Confirm {
			return ShowMenu
		}
		if d.Tick >= 34 {
			d.Phase = LogoIn
			d.Step = 1
		}
	case LogoIn:
		if input.Confirm {
			return ShowMenu
		}
		d.LogoScale = d.Step
		d.Step++
		if d.Step >= 16 {
			d.Phase = Credits
			d.Step = 0
			d.CreditPair = 0
		}
	case Credits:
		if input.Confirm {
			return ShowMenu
		}
		steps := d.Data.TextZoomSteps
		if d.CreditStage == 1 && len(d.Data.CreditSecondSteps) != 0 {
			steps = d.Data.CreditSecondSteps
		}
		if d.CreditStage == 2 {
			steps = d.Data.CreditOutSteps
			if len(steps) == 0 {
				steps = d.Data.DisappearSteps
			}
		}
		value := steps[d.Step]
		d.Step++
		if value < 0 {
			d.CreditStage++
			d.Step = 0
			if d.CreditStage >= 3 {
				d.CreditStage = 0
				d.CreditPair++
			}
			if d.CreditPair >= 6 {
				d.Phase = LogoOut
				d.Step = 16
				break
			}
			steps = d.Data.TextZoomSteps
			if d.CreditStage == 1 && len(d.Data.CreditSecondSteps) != 0 {
				steps = d.Data.CreditSecondSteps
			}
			if d.CreditStage == 2 {
				steps = d.Data.CreditOutSteps
				if len(steps) == 0 {
					steps = d.Data.DisappearSteps
				}
			}
			value = steps[0]
			d.Step = 1
		}
		d.LogoScale = 16
		pair := d.CreditPair * 2
		switch d.CreditStage {
		case 0:
			d.Captions = []Caption{{Text: d.Data.Credits[pair], Scale: value, CenterY: 120, Advance: 16}}
		case 1:
			d.Captions = []Caption{{Text: d.Data.Credits[pair], Scale: 16, CenterY: 120, Advance: 16}, {Text: d.Data.Credits[pair+1], Scale: value, CenterY: 150, Advance: 16}}
		case 2:
			d.Captions = []Caption{{Text: d.Data.Credits[pair], Scale: value, CenterY: 120, Advance: 16}, {Text: d.Data.Credits[pair+1], Scale: value, CenterY: 150, Advance: 16}}
		}
	case LogoOut:
		if input.Confirm {
			return ShowMenu
		}
		d.Captions = nil
		d.LogoScale = d.Step
		d.Step--
		if d.Step == 0 {
			d.Phase = ScoresIn
			d.Step = 0
		}
	case ScoresIn:
		d.Captions = []Caption{{Text: d.Data.HighScoreHeading, Scale: d.Data.AppearSteps[d.Step], CenterY: 12, Advance: 16}}
		d.Step++
		if d.Step >= len(d.Data.AppearSteps)-1 {
			d.Phase = ScoresHold
			d.Tick = 0
			d.ShowScores = true
		}
	case ScoresHold:
		d.Tick++
		if input.Confirm {
			return ShowMenu
		}
		if d.Tick >= 85 {
			d.Phase = ScoresOut
			d.Step = 0
			d.ShowScores = false
		}
	case ScoresOut:
		d.Captions = []Caption{{Text: d.Data.HighScoreHeading, Scale: d.Data.DisappearSteps[d.Step], CenterY: 12, Advance: 16}}
		d.Step++
		if d.Step >= len(d.Data.DisappearSteps)-1 {
			d.Captions = nil
			d.Phase = LogoDelay
			d.Tick = 0
			return AttractComplete
		}
	case ReadyMessage:
		value := d.Data.MessageSteps[d.Step]
		if value == 17 && !input.Confirm {
			value = 16
		} else {
			d.Step++
		}
		d.Captions[0].Scale = value
		if value < 0 {
			d.Captions = nil
			return ResumeGame
		}
	case ContinueIn:
		d.Captions = []Caption{{Text: d.Data.ContinueHeading, Scale: d.Data.AppearSteps[d.Step], CenterY: 80, Advance: 16}}
		d.Step++
		if d.Step >= len(d.Data.AppearSteps)-1 {
			d.Phase = ContinueHold
			d.Tick = 79
		}
	case ContinueHold:
		d.Countdown = d.Tick / 8
		text := d.Data.ContinueCounter
		text = strings.Replace(text, "9", fmt.Sprint(d.Countdown), 1)
		d.Captions = []Caption{{Text: d.Data.ContinueHeading, Scale: 16, CenterY: 80, Advance: 16}, {Text: text, Scale: 16, CenterY: 120, Advance: 16}}
		if input.Confirm {
			d.Accepted = true
			d.Phase = ContinueOut
			d.Step = 0
		} else {
			d.Tick--
			if d.Tick == 0 {
				d.Phase = ContinueOut
				d.Step = 0
			}
		}
	case ContinueOut:
		d.Captions = []Caption{{Text: d.Data.ContinueHeading, Scale: d.Data.DisappearSteps[d.Step], CenterY: 80, Advance: 16}}
		d.Step++
		if d.Step >= len(d.Data.DisappearSteps)-1 {
			d.Captions = nil
			if d.Accepted {
				return ContinueAccepted
			}
			return ContinueDeclined
		}
	case Initials:
		if input.Horizontal != 0 {
			d.InitialLetter = (d.InitialLetter + input.Horizontal + 38) % 38
		}
		letters := []byte(d.Scores[d.InitialRow].Initials)
		letters[d.InitialCharacter] = d.initialAlphabet()[d.InitialLetter]
		d.Scores[d.InitialRow].Initials = string(letters)
		if input.Confirm {
			d.InitialCharacter++
			d.InitialLetter = 0
			if d.InitialCharacter == 3 {
				d.Phase = ScoresHold
				d.Tick = 68
			}
		}
	}
	return NoResult
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
	d.InitialRow = row
	d.InitialCharacter, d.InitialLetter = 0, 0
	d.Phase = Initials
	d.ShowScores = true
	d.LogoScale = 0
	d.Captions = []Caption{{Text: d.Data.HighScoreHeading, Scale: 16, CenterY: 12, Advance: 16}}
	return true
}

func (d *Director) initialAlphabet() string { return "ABCDEFGHIJKLMNOPQRSTUVWXYZ.:0123456789" }
