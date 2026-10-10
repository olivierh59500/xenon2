package engine

import "fmt"

// Session keeps one independent saved game per alternating player while
// carrying the shared random stream across turns and shop interactions.
type Session struct {
	Completed        [2]bool
	Difficulty       int
	Players          [2]*World
	PlayerCount      int
	Current          int
	hasPlayed        [2]bool
	completedReentry bool
}

func NewSession(level LevelData, players int, random RandomState) (*Session, error) {
	if players != 1 && players != 2 {
		return nil, fmt.Errorf("a session needs one or two alternating players")
	}
	s := &Session{PlayerCount: players, Difficulty: 1}
	for i := range players {
		data := level
		data.InitialRandom = &random
		world, err := NewWorld(data)
		if err != nil {
			return nil, err
		}
		world.SetRandomState(random)
		world.Ready = true
		s.Players[i] = world
	}
	s.hasPlayed[0] = true
	return s, nil
}

func (s *Session) ActiveWorld() *World { return s.Players[s.Current] }

// Advance changes turns after a nonfinal ship loss. Final losses remain on the
// current player until the director finishes score entry and the continue offer.
func (s *Session) Advance(input Input) (turnChanged bool, err error) {
	world := s.ActiveWorld()
	other := s.Current ^ 1
	world.deferCheckpointRestart = s.PlayerCount == 2 && !s.Completed[other] && !s.Players[other].GameOver && s.Players[other].Equipment.Lives > 0
	err = world.Step(input)
	world.deferCheckpointRestart = false
	if err != nil {
		return false, err
	}
	if world.shipLossCompleted && !world.GameOver {
		if s.Completed[s.Current^1] {
			return false, nil
		}
		return s.switchTurn(), nil
	}
	return false, nil
}

// DeclineContinue is called after the original director's offer expires or is
// refused. It admits the other player if that game still has ships remaining.
func (s *Session) DeclineContinue() (turnChanged bool) {
	if !s.ActiveWorld().GameOver {
		return false
	}
	changed := s.switchTurn()
	s.completedReentry = changed && s.Completed[s.Current]
	return changed
}

// AcceptContinue records the new ships on the accepting player's saved game,
// then follows ordinary turn admission. A live incomplete opponent plays next.
func (s *Session) AcceptContinue() bool {
	w := s.ActiveWorld()
	next := s.Current ^ 1
	change := s.PlayerCount == 2 && !s.Completed[next] && !s.Players[next].GameOver && s.Players[next].Equipment.Lives > 0
	w.deferCheckpointRestart = change
	accepted := w.AcceptContinue()
	w.deferCheckpointRestart = false
	if accepted && change {
		s.switchTurn()
	}
	return accepted
}

func (s *Session) switchTurn() bool {
	if s.PlayerCount != 2 {
		return false
	}
	next := s.Current ^ 1
	if s.Players[next].GameOver || s.Players[next].Equipment.Lives == 0 {
		return false
	}
	random := s.ActiveWorld().RandomState()
	background := s.ActiveWorld().BackgroundY
	s.ActiveWorld().suspendTurn()
	s.Current = next
	world := s.ActiveWorld()
	world.SetRandomState(random)
	// The backdrop offset is shared display state outside the saved players.
	world.BackgroundY, world.PreviousBackgroundY = background, background
	world.RestartCheckpoint()
	world.LevelFinished = s.Completed[next]
	s.hasPlayed[next] = true
	world.Ready = true
	return true
}
