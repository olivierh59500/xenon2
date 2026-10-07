package engine

import "xenon2/internal/visualassets"

type ThirdCannonState struct {
	X, WorldY, Stage, Phase int
	Health                  uint16
	FireAccumulator         uint8
	Removed                 bool
}
type ThirdCannonEvents struct {
	WriteFrame  bool
	Frame       int
	Collision   CollisionRect
	ShotCount   int
	Directions  [8]uint8
	X, Y, Speed int
	SecondStage bool
}

func NewThirdCannon(record visualassets.FixedEncounter, art visualassets.ThirdCompoundCannonArt) ThirdCannonState {
	return ThirdCannonState{X: record.X - 8, WorldY: record.Y - 8, Health: uint16(art.Health[0])}
}

// CollisionAt projects the active source weak point from its fixed world anchor.
// Drawing history is unrelated to this terrain controller's collision geometry.
func (s *ThirdCannonState) CollisionAt(scrollY int) CollisionRect {
	if s.Removed {
		return CollisionRect{Right: -1, Bottom: -1}
	}
	top, height := s.WorldY-scrollY+64, 27
	if s.Stage == 1 {
		top, height = s.WorldY-scrollY+4, 24
	}
	return CollisionRect{Left: s.X + 16, Top: top, Right: s.X + 47, Bottom: top + height - 1}
}

func (s *ThirdCannonState) Advance(scrollY, maximum int, art visualassets.ThirdCompoundCannonArt, random *RandomState) ThirdCannonEvents {
	event := ThirdCannonEvents{Collision: CollisionRect{Right: -1, Bottom: -1}, SecondStage: s.Stage == 1}
	if s.Removed {
		return event
	}
	if maximum+208 < s.WorldY {
		s.Removed = true
		return event
	}
	event.Collision = s.CollisionAt(scrollY)
	if s.Stage == 0 {
		if s.Phase == 0 && (random == nil || int(uint8(random.Next())) >= art.FirstThreshold) {
			return event
		}
		event.WriteFrame, event.Frame = true, s.Phase/2
		s.Phase++
		if s.Phase == 18 {
			s.Phase = 0
			return event
		}
		if s.Phase == 12 {
			if random != nil && int(uint8(random.Next())) < art.RepeatThreshold {
				s.Phase = 4
			}
			return event
		}
		if s.Phase < 4 || s.Phase > 11 {
			return event
		}
		if !thirdCannonCarry(s, art.FirstFireRate, random) {
			return event
		}
		direction := uint8(4)
		if s.Phase&^1 == 6 {
			direction = 3
		}
		if s.Phase&^1 == 8 {
			direction = 5
		}
		event.ShotCount, event.Directions[0], event.X, event.Y, event.Speed = 1, direction, s.X+32, s.WorldY-scrollY+88, art.FirstShotSpeed
	} else {
		if s.Phase == 0 && !thirdCannonCarry(s, art.SecondFireRate, random) {
			return event
		}
		event.WriteFrame, event.Frame = true, (s.Phase&^1)/2
		s.Phase++
		if s.Phase == 10 {
			s.Phase = 0
			return event
		}
		if s.Phase != 4 {
			return event
		}
		event.ShotCount, event.X, event.Y, event.Speed = 8, s.X+32, s.WorldY-scrollY+24, art.SecondShotSpeed
		for i := range 8 {
			event.Directions[i] = uint8(7 - i)
		}
	}
	return event
}
func thirdCannonCarry(s *ThirdCannonState, rate int, random *RandomState) bool {
	sum := int(s.FireAccumulator) + rate
	s.FireAccumulator = uint8(sum)
	if sum < 256 {
		return false
	}
	if random != nil {
		s.FireAccumulator = uint8(random.Next() & 63)
	}
	return true
}
func (s *ThirdCannonState) Strike(amount uint16, art visualassets.ThirdCompoundCannonArt) (transition, removed bool) {
	if s.Removed {
		return false, false
	}
	result := ApplyEnemyDamage(s.Health, amount)
	s.Health = result.Health
	if !result.Destroyed {
		return false, false
	}
	if s.Stage == 0 {
		s.Stage, s.Phase, s.Health = 1, 0, uint16(art.Health[1])
		return true, false
	}
	s.Removed = true
	return false, true
}
