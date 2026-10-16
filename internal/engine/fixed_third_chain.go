package engine

import "xenon2/internal/visualassets"

type ThirdChainPart struct {
	X, Y             int
	Spacing          int
	Animation        AnimationState
	Visible, Removed bool
}
type ThirdChainState struct {
	Variant                           int
	Parts                             [8]ThirdChainPart
	Phase, Speed, AmplitudeOrCooldown int
}
type ThirdChainInput struct{ ScrollDelta, PlayerY int }

func NewThirdChain(record visualassets.FixedEncounter, scrollY int, clip visualassets.ActorAnimation) ThirdChainState {
	state := ThirdChainState{Variant: record.Variant}
	x := record.X - 8
	if record.Variant != 0 {
		x += 32
	}
	for i := range state.Parts {
		spacing := i
		if record.Variant != 0 {
			spacing = -i
		}
		state.Parts[i] = ThirdChainPart{X: x, Y: record.Y - scrollY, Spacing: spacing}
	}
	state.Parts[7].Animation = NewAnimation(clip)
	return state
}
func (s *ThirdChainState) Advance(input ThirdChainInput, art visualassets.ThirdChainArt, random *RandomState) {
	head := &s.Parts[0]
	head.Y += input.ScrollDelta
	if head.Y >= 232 {
		head.Removed = true
	} else {
		head.Visible = false
		if s.Speed == 0 {
			if s.AmplitudeOrCooldown < 0 {
				s.AmplitudeOrCooldown++
			} else {
				delta := head.Y - input.PlayerY
				if delta < 0 {
					delta = -delta
				}
				if delta <= 32 && random != nil {
					s.Speed = 2
					s.AmplitudeOrCooldown = int(random.Next()&6) + 10
				}
			}
		}
		if s.Speed != 0 {
			head.Visible = true
			s.Phase += s.Speed
			if s.Phase == 0 {
				s.Speed, s.AmplitudeOrCooldown = 0, -5
			} else if s.Phase == s.AmplitudeOrCooldown {
				s.Speed = -s.Speed
			}
		}
	}
	for i := 1; i < 8; i++ {
		part := &s.Parts[i]
		part.Y += input.ScrollDelta
		if part.Y >= 232 {
			part.Removed = true
			continue
		}
		part.X = head.X + part.Spacing*s.Phase
		part.Visible = s.Speed != 0
		if i == 7 {
			part.Animation.Advance(art.Tails[s.Variant])
			part.Visible = true
			if s.Speed == 2 && s.Phase == 2 || s.AmplitudeOrCooldown > 0 && s.Phase == s.AmplitudeOrCooldown {
				part.Animation.Remaining = 1
			}
		}
	}
}
