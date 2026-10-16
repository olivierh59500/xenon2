package engine

// SecondDefenseNodeState is the middle arena's one-tile target. Visibility and
// collision depend on the player side and the shared defense-wave flags.
type SecondDefenseNodeState struct {
	Index, TileX, TileY int
	Health              uint16
	Phase               int
	Destroyed           bool
	Collision           CollisionRect
}

type SecondDefenseNodeInput struct {
	Frame                            uint64
	PlayerX, ScrollY, MaximumScrollY int
	Remaining                        int
	DefenseFlags                     uint8
	BackwardScroll                   bool
}

type SecondDefenseNodeEvents struct {
	MaximumScrollY int
	TileFrame      int
	GateChanged    [8]bool
	GateFrame      [8]int
}

func NewSecondDefenseNodeState(index, x, y, health int) SecondDefenseNodeState {
	return SecondDefenseNodeState{Index: index, TileX: x, TileY: y, Health: uint16(health), Collision: CollisionRect{Right: -1, Bottom: -1}}
}

func (s *SecondDefenseNodeState) Advance(input SecondDefenseNodeInput, gateCounters *[8]int) SecondDefenseNodeEvents {
	event := SecondDefenseNodeEvents{MaximumScrollY: input.MaximumScrollY}
	if s.Destroyed {
		return event
	}
	if input.BackwardScroll && event.MaximumScrollY <= 2880 {
		event.MaximumScrollY = 2880
	}
	eligible := false
	switch s.Index {
	case 2:
		eligible = input.Remaining == 1
	case 1:
		eligible = input.PlayerX < 160 && input.DefenseFlags != 3
	case 0:
		eligible = input.PlayerX >= 160 && input.DefenseFlags != 3
	}
	if eligible {
		s.Collision = CollisionRect{Left: s.TileX * 16, Top: s.TileY*16 - input.ScrollY, Right: s.TileX*16 + 15, Bottom: s.TileY*16 - input.ScrollY + 15}
	} else {
		s.Collision = CollisionRect{Right: -1, Bottom: -1}
	}
	if !(eligible && s.Phase == 4 && input.DefenseFlags != 3) && input.Frame&1 == 0 {
		s.Phase = (s.Phase + 1) & 7
	}
	event.TileFrame = s.Phase
	if event.TileFrame > 4 {
		event.TileFrame = 8 - event.TileFrame
	}
	if s.Index == 2 && gateCounters != nil {
		for index := range gateCounters {
			if gateCounters[index] == 0 {
				continue
			}
			gateCounters[index]--
			event.GateChanged[index] = true
			event.GateFrame[index] = (gateCounters[index] >> 1) & 1
		}
	}
	return event
}

type SecondDefenseDamageEvents struct {
	Applied, Destroyed, Flash       bool
	Remaining                       int
	DefenseFlags                    uint8
	ReverseScroll                   bool
	ReleaseMinimum, ClearWaveActors bool
	CashPairs, ExplosionCount       int
}

// Strike blocks damage while both source defense streams remain active. The
// final node's removal releases the middle arena, rather than ending the level.
func (s *SecondDefenseNodeState) Strike(amount uint16, remaining int, flags uint8) SecondDefenseDamageEvents {
	event := SecondDefenseDamageEvents{Remaining: remaining, DefenseFlags: flags}
	if s.Destroyed || flags == 3 {
		return event
	}
	result := ApplyEnemyDamage(s.Health, amount)
	s.Health = result.Health
	event.Applied, event.Flash = true, true
	if !result.Destroyed {
		return event
	}
	s.Destroyed = true
	s.Collision = CollisionRect{Right: -1, Bottom: -1}
	event.Destroyed, event.DefenseFlags, event.Remaining = true, 3, remaining-1
	if event.Remaining == 1 {
		event.ReverseScroll = true
	}
	if event.Remaining == 0 {
		event.ReleaseMinimum, event.ClearWaveActors = true, true
		event.CashPairs, event.ExplosionCount = 5, 40
	}
	return event
}
