package engine

import "xenon2/internal/visualassets"

// FixedPodState is a world-anchored scenery emitter. It produces one creature
// per opening and retains its last terrain frame after removal.
type FixedPodState struct {
	X, WorldY, Phase, Repeats int
	Large                     bool
	Removed                   bool
}

type FixedPodEvents struct {
	WriteTiles     bool
	Frame          int
	Spawn          bool
	SpawnX, SpawnY int
}

func (s *FixedPodState) Advance(frame uint64, scroll, maximum int) FixedPodEvents {
	var event FixedPodEvents
	if s.Removed {
		return event
	}
	if maximum+208 <= s.WorldY {
		s.Removed = true
		return event
	}
	y := s.WorldY - scroll
	if y >= 180 {
		return event
	}
	if s.Phase >= 5 {
		if s.Repeats == 0 {
			s.Removed = true
			return event
		}
		if s.Phase >= 9 {
			s.Phase = 0
			s.Repeats--
		}
	}
	if s.Phase <= 1 {
		if y >= 100 {
			s.Phase = 2
		} else if !s.Large {
			return event
		} else if frame&3 == 0 {
			s.Phase ^= 1
		}
	}
	event.WriteTiles, event.Frame = true, s.Phase
	if s.Phase >= 2 && frame&3 == 0 {
		s.Phase++
		if s.Phase == 4 {
			event.Spawn = true
			event.SpawnX, event.SpawnY = s.X, s.WorldY-scroll
			if s.Large {
				event.SpawnX += 16
				event.SpawnY += 16
			}
		}
	}
	return event
}

// PodCreatureState uses a named allocation phase for the original per-slot
// wobble. It never stores or dispatches a native address.
type PodCreatureState struct {
	X, Y, Phase, Variant int
	AllocationPhase      int
	Animation            AnimationState
	Clip                 visualassets.ActorAnimation
}

func (s *PodCreatureState) Advance(frame uint64, playerX int, art *visualassets.PodCreatureArtwork) {
	s.Animation.Advance(s.Clip)
	selectClip := func(attack bool) {
		s.Clip = art.Idle[s.Variant]
		if attack {
			s.Clip = art.Attack[s.Variant]
		}
		s.Animation = NewAnimation(s.Clip)
	}
	if s.Phase != 0 {
		s.Y += 12
		if s.Y >= 160 {
			s.Phase = 0
			selectClip(false)
		}
		return
	}
	if s.Y > 50 {
		s.Y -= 5
		s.X += art.Wobble[(((int(frame)+s.AllocationPhase)>>1)&14)/2]
		return
	}
	difference := s.X - playerX
	if difference < 0 {
		difference = -difference
	}
	if difference < 10 {
		s.Phase = 1
		selectClip(true)
		return
	}
	if s.X < playerX {
		s.X += 4
	} else {
		s.X -= 4
	}
	s.Y += art.Wobble[((int(frame)+s.AllocationPhase)&14)/2]
	if s.Y > 50 {
		s.Y = 50
	}
}
