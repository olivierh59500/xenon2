package engine

import "xenon2/internal/visualassets"

type ThirdCrawlerState struct {
	X, Y, Direction int
	FireAccumulator uint8
	Animation       AnimationState
	Removed         bool
}
type ThirdCrawlerInput struct {
	ScrollDelta, ScrollY, MaximumScrollY, PlayerX, PlayerY int
	Columns                                                int
	Map                                                    []uint16
}
type ThirdCrawlerEvents struct {
	Shot        bool
	X, Y, Speed int
	Direction   uint8
}

func NewThirdCrawler(record visualassets.FixedEncounter, scrollY int, art visualassets.ThirdCrawlerArt) ThirdCrawlerState {
	direction := record.State2
	return ThirdCrawlerState{X: record.X - 8 + art.InitialDX[direction], Y: record.Y - 8 - scrollY + art.InitialDY[direction], Direction: direction, Animation: NewAnimation(art.Animations[direction])}
}

func (s *ThirdCrawlerState) Advance(input ThirdCrawlerInput, art visualassets.ThirdCrawlerArt, random *RandomState) ThirdCrawlerEvents {
	var event ThirdCrawlerEvents
	if s.Removed {
		return event
	}
	s.Animation.Advance(art.Animations[s.Direction])
	s.Y += input.ScrollDelta
	if input.MaximumScrollY+208-input.ScrollY < s.Y {
		s.Removed = true
		return event
	}
	dx, dy := art.StepDX[s.Direction], art.StepDY[s.Direction]
	s.X += dx
	s.Y += dy
	crossed := false
	switch {
	case dx < 0:
		crossed = s.X&15 == 15
	case dx > 0:
		crossed = s.X&15 == 0
	case dy < 0:
		crossed = (s.Y+input.ScrollY)&15 == 15
	case dy > 0:
		crossed = (s.Y+input.ScrollY)&15 == 0
	}
	if crossed {
		turn := false
		if s.X <= 0 || s.X >= 320 {
			turn = true
		} else {
			cell := (s.Y+input.ScrollY)/16*input.Columns + s.X/16
			occupied := func(index int) bool { return index >= 0 && index < len(input.Map) && input.Map[index] != 0 }
			if occupied(cell) {
				s.X -= dx
				s.Y -= dy
				turn = true
			} else if !occupied(cell + art.ForwardTileOffset[s.Direction]/2) {
				turn = true
			}
		}
		if turn {
			s.X += art.TurnDX[s.Direction]
			s.Y += art.TurnDY[s.Direction]
			s.Direction = art.TurnHeading[s.Direction]
			s.Animation = NewAnimation(art.Animations[s.Direction])
		}
	}
	sum := int(s.FireAccumulator) + art.FireRate
	s.FireAccumulator = uint8(sum)
	if sum >= 256 && random != nil {
		s.FireAccumulator = uint8(random.Next() & 63)
		event = ThirdCrawlerEvents{Shot: true, X: s.X, Y: s.Y, Speed: art.ShotSpeed, Direction: AimDirection(input.PlayerX-s.X, input.PlayerY-s.Y)}
	}
	return event
}
