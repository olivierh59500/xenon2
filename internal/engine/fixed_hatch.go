package engine

import "xenon2/internal/visualassets"

type FixedHatchState struct {
	X, WorldY, Phase int
	Removed          bool
}

type FixedHatchEvents struct {
	WriteTiles     bool
	Frame          int
	Spawn          bool
	SpawnX, SpawnY int
}

func (s *FixedHatchState) Advance(scroll, threshold int) FixedHatchEvents {
	var event FixedHatchEvents
	if s.Removed {
		return event
	}
	if s.Phase == 0 {
		if s.WorldY-scroll >= threshold {
			s.Phase = 1
		}
		return event
	}
	event.WriteTiles, event.Frame = true, s.Phase
	s.Phase++
	if s.Phase >= 21 {
		s.Removed = true
		event.Spawn = true
		event.SpawnX, event.SpawnY = s.X+16, s.WorldY-scroll+16
	}
	return event
}

// HatchCreatureState retains its finite escape timer and periodically chooses
// the source's inverted eight-direction aim away from the ship.
type HatchCreatureState struct {
	X, Y, Timer int
	Direction   uint8
	Animation   AnimationState
	Clip        visualassets.ActorAnimation
	Removed     bool
}

type HatchCreatureEvents struct {
	PlayerDamage int
	Explosion    bool
}

func (s *HatchCreatureState) Advance(art *visualassets.HatchCreatureArtwork, input FixedProjectileInputs, box func(string) visualassets.CollisionBox) HatchCreatureEvents {
	var event HatchCreatureEvents
	if s.Removed {
		return event
	}
	s.Animation.Advance(s.Clip)
	s.Timer++
	if s.Timer >= art.Lifetime {
		s.Removed = true
		event.Explosion = true
		return event
	}
	if s.Timer&15 == 0 {
		s.Direction = ^AimDirection(input.PlayerX-s.X, input.PlayerY-s.Y) & 7
		s.Clip = art.HeadingAnimations[s.Direction]
		s.Animation = NewAnimation(s.Clip)
	}
	x := s.X + art.StepX[s.Direction]
	if x < 0 || x >= 320 {
		s.Removed = true
		return event
	}
	s.X = x
	y := s.Y + art.StepY[s.Direction] + input.ScrollDelta
	if y < 0 || y >= 192 {
		s.Removed = true
		return event
	}
	s.Y = y
	rect := ActorCollisionRect(box(s.Animation.Sprite(s.Clip)), s.X, s.Y)
	if input.CanHitPlayer && rect.Intersects(input.PlayerBounds) {
		s.Removed = true
		if input.Invulnerable {
			event.Explosion = true
		} else {
			event.PlayerDamage = 4
		}
	}
	return event
}
