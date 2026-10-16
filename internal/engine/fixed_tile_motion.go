package engine

import "xenon2/internal/visualassets"

// FixedTileState owns mutable terrain animation separately from sprite actors.
// Coordinates remain anchored to the level map, including during reverse scroll.
type FixedTileState struct {
	Kind, Variant int
	X, WorldY     int
	Phase         int
	Accumulator   uint8
	Removed       bool
}

type FixedTileEvents struct {
	Frame                                int
	Collision                            CollisionRect
	Shot                                 bool
	ShotX, ShotY, ShotSpeed, ShotVariant int
	ShotDirection                        uint8
	WriteTiles                           bool
}

func StepFirstTileCannon(s *FixedTileState, kind visualassets.FixedTileKind, scroll, maximum int, random *RandomState) FixedTileEvents {
	event := FixedTileEvents{Collision: CollisionRect{Right: -1, Bottom: -1}}
	if s.Removed {
		return event
	}
	if maximum+208 < s.WorldY {
		s.Removed = true
		return event
	}
	y := s.WorldY - scroll
	event.WriteTiles = true
	switch s.Kind {
	case 1:
		event.Collision = CollisionRect{Left: s.X + 10, Top: y + 4, Right: s.X + 28, Bottom: y + 28}
		event.Frame = s.Phase >> 2
		s.Phase = (s.Phase + 1) & 15
	case 2:
		event.Collision = CollisionRect{Left: s.X + 4, Top: y + 4, Right: s.X + 28, Bottom: y + 28}
		event.Frame = s.Phase
		s.Phase = (s.Phase + 1) & 3
	case 3:
		event.Collision = CollisionRect{Left: s.X + 8, Top: y + 8, Right: s.X + 24, Bottom: y + 24}
		event.Frame = s.Phase >> 1
	}
	if s.Kind != 3 || s.Phase == 0 {
		sum := int(s.Accumulator) + kind.FireRate
		s.Accumulator = uint8(sum)
		if sum < 256 {
			return event
		}
		s.Accumulator = uint8(random.Next() & 63)
	}
	if s.Kind == 3 {
		s.Phase++
		if s.Phase == 40 {
			s.Phase = 0
			return event
		}
		if s.Phase != 24 {
			return event
		}
	}
	event.Shot = true
	event.ShotDirection = 6
	offsetX, offsetY := 0, 14
	if s.Kind == 3 {
		offsetY = 2
	}
	// All three constructors use record variant zero for the right-facing
	// cannon. Their resource tags use different numeric orders.
	right := s.Variant == 0
	if right {
		event.ShotDirection = 2
		switch s.Kind {
		case 1:
			offsetX = 32
		case 2:
			offsetX = 16
		case 3:
			offsetX = 26
		}
	}
	event.ShotX, event.ShotY = s.X+offsetX, y+offsetY
	event.ShotSpeed, event.ShotVariant = kind.ShotSpeed, kind.ShotVariant
	return event
}

// StepTileFireCycle preserves idle trigger draws and the different pre-step or
// post-step frame selection used by the finite terrain cannon animations.
func StepTileFireCycle(s *FixedTileState, kind visualassets.FixedTileKind, scroll, maximum int, random *RandomState) FixedTileEvents {
	event := FixedTileEvents{Collision: CollisionRect{Right: -1, Bottom: -1}}
	if s.Removed {
		return event
	}
	if maximum+208 < s.WorldY {
		s.Removed = true
		return event
	}
	event.Collision = ActorCollisionRect(kind.Collision, s.X, s.WorldY-scroll)
	if s.Phase == 0 {
		if kind.IdleRandomThreshold {
			if uint8(random.Next()) < uint8(kind.FireRate) {
				s.Phase = 1
			}
			return event
		}
		sum := int(s.Accumulator) + kind.FireRate
		s.Accumulator = uint8(sum)
		if sum < 256 {
			return event
		}
		s.Accumulator = uint8(random.Next() & 63)
		if kind.FrameBeforeStep {
			s.Phase = 1
			return event
		}
	}
	if kind.FrameBeforeStep {
		event.Frame = s.Phase / kind.FrameDuration
		s.Phase++
	} else {
		s.Phase++
		event.Frame = s.Phase / kind.FrameDuration
	}
	event.WriteTiles = true
	if s.Phase == kind.PhaseLength {
		s.Phase = 0
		return event
	}
	if s.Phase != kind.ShotPhase {
		return event
	}
	event.Shot = true
	event.ShotX = s.X + kind.ShotOffsetX[s.Variant]
	event.ShotY = s.WorldY - scroll + kind.ShotOffsetY[s.Variant]
	event.ShotDirection = kind.ShotDirections[s.Variant]
	event.ShotVariant, event.ShotSpeed = kind.ShotVariant, kind.ShotSpeed
	return event
}
