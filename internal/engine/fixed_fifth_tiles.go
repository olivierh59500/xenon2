package engine

import "xenon2/internal/visualassets"

// FifthTileState retains a map-anchored barrier part or aiming turret. Its
// animation clock is independent of the render rate and reverse scrolling.
type FifthTileState struct {
	X, WorldY, Part int
	Phase           int
	Heading         uint8
	Accumulator     uint8
	Removed         bool
}

func (s *FifthTileState) AdvanceBarrier(kind visualassets.FixedTileKind, scroll int) FixedTileEvents {
	part := kind.Parts[s.Part]
	if s.Part == 1 {
		s.Phase = (s.Phase + 4) & 15
	} else {
		s.Phase = (s.Phase + 1) & 3
	}
	frame := s.Phase
	if s.Part == 1 {
		frame /= 4
	}
	return FixedTileEvents{Frame: frame, WriteTiles: true, Collision: ActorCollisionRect(part.Collision, s.X, s.WorldY-scroll)}
}

// AdvanceAimingTurret keeps the initial rotation, the one-step shortest turn
// (including its four-step tie), and the carry-triggered shared random draw.
func (s *FifthTileState) AdvanceAimingTurret(kind visualassets.FixedTileKind, scroll, maximum, playerX, playerY int, random *RandomState) FixedTileEvents {
	e := FixedTileEvents{Collision: CollisionRect{Right: -1, Bottom: -1}}
	if s.Removed {
		return e
	}
	if int(int16(maximum+208)) < s.WorldY {
		s.Removed = true
		return e
	}
	e.Collision = ActorCollisionRect(kind.Collision, s.X, s.WorldY-scroll)
	e.WriteTiles = true
	if s.Phase == 0 && s.WorldY-scroll < -16 {
		e.Frame = int(s.Heading)
		return e
	}
	if s.Phase != 8 {
		s.Phase++
		s.Heading = (s.Heading + 1) & 7
	} else {
		target := AimDirection(int(int16(playerX-s.X-16)), int(int16(playerY-s.WorldY+scroll-16)))
		delta := (target - s.Heading) & 7
		if delta != 0 {
			if delta < 4 {
				s.Heading++
			} else {
				s.Heading--
			}
			s.Heading &= 7
		}
		sum := int(s.Accumulator) + kind.FireRate
		s.Accumulator = uint8(sum)
		if sum >= 256 {
			s.Accumulator = uint8(random.Next() & 63)
			e.Shot = true
			offset := kind.DirectionalOffsets[s.Heading]
			e.ShotX, e.ShotY = s.X+16+offset[0], s.WorldY-scroll+16+offset[1]
			e.ShotDirection, e.ShotSpeed = s.Heading, kind.ShotSpeed
		}
	}
	e.Frame = int(s.Heading)
	return e
}

func (s *FifthTileState) AdvanceRadialTurret(kind visualassets.FixedTileKind, scroll, maximum int, random *RandomState) FixedTileEvents {
	e := FixedTileEvents{Collision: CollisionRect{Right: -1, Bottom: -1}}
	if s.Removed {
		return e
	}
	if int(int16(maximum+208)) < s.WorldY {
		s.Removed = true
		return e
	}
	s.Phase = (s.Phase + 1) & 15
	e.Frame, e.WriteTiles = s.Phase/2, true
	e.Collision = ActorCollisionRect(kind.Collision, s.X, s.WorldY-scroll)
	sum := int(s.Accumulator) + kind.FireRate
	s.Accumulator = uint8(sum)
	if sum >= 256 {
		s.Accumulator = uint8(random.Next() & 63)
		e.Shot = true
		e.ShotX, e.ShotY = s.X+16, s.WorldY-scroll+16
		e.ShotSpeed = kind.ShotSpeed
	}
	return e
}
