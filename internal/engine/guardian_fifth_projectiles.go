package engine

import "xenon2/internal/visualassets"

type FifthLaserColumnState struct {
	X, Y, Length, Speed int
	Active              bool
	Collision           CollisionRect
}

// Advance preserves the separate growing and full-length column branches.
// A full column travels independently of scenery scrolling.
func (s *FifthLaserColumnState) Advance(scrollDelta, growth int) {
	if !s.Active {
		return
	}
	s.Collision.Left, s.Collision.Right = s.X, s.X+11
	if s.Length != 48 {
		s.Length = min(48, s.Length+growth)
		s.Y += scrollDelta
		if s.Speed < 0 {
			s.Y -= growth
		}
	} else {
		s.Y += s.Speed
		if s.Speed >= 0 && s.Y >= 192 || s.Speed < 0 && s.Y+s.Length < 0 {
			s.Active = false
			return
		}
	}
	s.Collision.Top, s.Collision.Bottom = s.Y, s.Y+s.Length
}

type FifthSeekingState struct {
	X, Y, Clock int
	Heading     uint8
	Animation   AnimationState
	Sprite      string
	Active      bool
}

// AdvanceMouth follows the source creature's eight-pass steering clock. It
// doubles its table movement after the guardian's eighteen outer parts fall.
func (s *FifthSeekingState) AdvanceMouth(group *visualassets.GuardianGroup, playerX, playerY, outerRemaining int) {
	if !s.Active {
		return
	}
	clips := group.Components[0].HeadingAnimations
	if len(clips) != 8 {
		return
	}
	s.Animation.Advance(clips[s.Heading])
	s.Sprite = s.Animation.Sprite(clips[s.Heading])
	s.Clock++
	if s.Clock&7 == 0 {
		target := AimDirection(playerX-s.X, playerY-s.Y)
		difference := (int(target) - int(s.Heading) + 8) & 7
		if difference != 0 {
			turn := 1
			if difference >= 4 {
				turn = -1
			}
			s.Heading = uint8((int(s.Heading) + turn + 8) & 7)
			s.Animation = NewAnimation(clips[s.Heading])
			s.Sprite = s.Animation.Sprite(clips[s.Heading])
		}
	}
	scale := 1
	if outerRemaining == 0 {
		scale = 2
	}
	s.X += group.MotionTables["seeking_x"][s.Heading] * scale
	s.Y += group.MotionTables["seeking_y"][s.Heading] * scale
}

// AdvanceSide retains the source lifetime, scroll adjustment and exact
// opposite-heading tie, which depends on the current horizontal coordinate.
func (s *FifthSeekingState) AdvanceSide(group *visualassets.GuardianGroup, playerX, playerY, scrollDelta int) {
	if !s.Active {
		return
	}
	clips := group.Components[0].HeadingAnimations
	if len(clips) != 8 {
		return
	}
	s.Animation.Advance(clips[s.Heading])
	s.Sprite = s.Animation.Sprite(clips[s.Heading])
	s.Clock++
	if s.Clock >= group.MotionParameters["side_lifetime"] {
		s.Active = false
		return
	}
	if s.Clock&7 == 0 {
		target := AimDirection(playerX-s.X, playerY-s.Y)
		difference := (int(target) - int(s.Heading) + 8) & 7
		if difference != 0 {
			turn := 1
			if difference > 4 || difference == 4 && s.X&1 == 0 {
				turn = -1
			}
			s.Heading = uint8((int(s.Heading) + turn + 8) & 7)
			s.Animation = NewAnimation(clips[s.Heading])
			s.Sprite = s.Animation.Sprite(clips[s.Heading])
		}
	}
	s.X += group.MotionTables["seeking_x"][s.Heading]
	s.Y += group.MotionTables["seeking_y"][s.Heading] + scrollDelta
}
