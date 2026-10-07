package engine

import "xenon2/internal/visualassets"

// FixedSpriteState stores scene coordinates and gameplay counters. ScrollDelta
// is applied once per gameplay pass; drawing never advances these counters.
type FixedSpriteState struct {
	X, Y                  int
	VelocityX, VelocityY  int
	Distance, Amplitude   int
	Phase, PhaseDirection int
	FireAccumulator       uint8
	Attacking, Removed    bool
	Variant               int
	Animation             AnimationState
}

type FixedSpriteInputs struct {
	ScrollDelta, ScrollY, MaximumScrollY int
	PlayerX, PlayerY                     int
}

// FixedSpriteEvents separates actor state from effects owned by World. ShotMode
// identifies point shots, the sweeper's turning shot, or an animated actor shot.
type FixedSpriteEvents struct {
	AttackStarted, VariantChanged bool
	MoveToTransientList           bool
	ShotMode, ShotSprite          string
	ShotDirections                [3]int
	ShotCount                     int
	ShotX, ShotY, ShotSpeed       int
	ShotMotionBudget, ShotDelay   int
	ShotAnimation                 visualassets.ActorAnimation
	ContactRectangle              [4]int
	ContactDamage                 int
}

func NewFixedSpriteState(kind visualassets.FixedSpriteKind, variant visualassets.FixedSpriteVariant, event visualassets.FixedEncounter, scrollY int) FixedSpriteState {
	state := FixedSpriteState{X: event.X + variant.OriginOffsetX, Y: event.Y + variant.OriginOffsetY - scrollY, Variant: variant.ID, VelocityX: variant.InitialVelocityX, Animation: NewAnimation(variant.Animation)}
	if kind.Behavior == "scroll-bounce-attack" {
		state.VelocityY = kind.MotionParameters["initial_vertical_velocity"]
		if scale := kind.MotionParameters["amplitude_state1_scale"]; scale != 0 {
			state.Amplitude = event.State1 * scale
		}
		if scale := kind.MotionParameters["amplitude_state2_scale"]; scale != 0 {
			state.Amplitude = event.State2 * scale
		}
		state.Distance = state.Amplitude
	}
	if kind.Behavior == "vertical-oscillator" {
		state.VelocityY = 20
	}
	return state
}

// FixedSpriteAnimation selects the source clip, including attack and reverse
// clips. A horizontal reversal can change the directional variant itself.
func FixedSpriteAnimation(state FixedSpriteState, kind visualassets.FixedSpriteKind) visualassets.ActorAnimation {
	variant, ok := fixedSpriteVariant(kind, state.Variant)
	if !ok {
		return visualassets.ActorAnimation{}
	}
	if state.Attacking && len(variant.AttackAnimation.Frames) > 0 {
		return variant.AttackAnimation
	}
	if kind.Behavior == "vertical-oscillator" && state.VelocityY < 0 {
		return variant.ReverseAnimation
	}
	return variant.Animation
}

func fixedSpriteVariant(kind visualassets.FixedSpriteKind, id int) (visualassets.FixedSpriteVariant, bool) {
	for _, variant := range kind.Variants {
		if variant.ID == id {
			return variant, true
		}
	}
	return visualassets.FixedSpriteVariant{}, false
}

// StepFixedSpriteMotion preserves the different callback orders of the five
// original families, including byte overflow and random draws for attack rates.
func StepFixedSpriteMotion(state *FixedSpriteState, kind visualassets.FixedSpriteKind, input FixedSpriteInputs, random *RandomState) FixedSpriteEvents {
	var result FixedSpriteEvents
	if state.Removed {
		return result
	}
	variant, ok := fixedSpriteVariant(kind, state.Variant)
	if !ok {
		state.Removed = true
		return result
	}
	if kind.Behavior == "scroll-bounce-attack" {
		if input.MaximumScrollY+kind.MotionParameters["clip_margin"]-input.ScrollY < state.Y {
			state.Removed = true
			return result
		}
		advanceFixedAnimation(state, FixedSpriteAnimation(*state, kind))
		state.Y += input.ScrollDelta
	} else {
		state.Y += input.ScrollDelta
		switch kind.Behavior {
		case "horizontal-sweeper":
			state.Removed = input.MaximumScrollY+208-input.ScrollY < state.Y
		case "extending-beam":
			state.Removed = state.Y > kind.MotionParameters["clip_bottom"]
		case "vertical-oscillator":
			state.Removed = state.Y >= kind.MotionParameters["clip_bottom"]
		}
		if state.Removed {
			return result
		}
		advanceFixedAnimation(state, FixedSpriteAnimation(*state, kind))
	}
	switch kind.Behavior {
	case "scroll-bounce-attack":
		if state.VelocityY == 0 {
			break
		}
		delta := state.Y - input.PlayerY
		if delta >= kind.MotionParameters["attack_y_minimum"] && delta <= kind.MotionParameters["attack_y_maximum"] && random != nil && int(byte(random.Next())) < kind.MotionParameters["attack_random_threshold"] {
			state.VelocityY, state.Attacking = 0, true
			state.Animation = NewAnimation(variant.AttackAnimation)
			result.AttackStarted, result.MoveToTransientList = true, true
			setFixedShots(&result, *state, variant)
			break
		}
		state.Y += state.VelocityY
		state.Distance += state.VelocityY
		if state.Distance == 0 || state.Distance == state.Amplitude {
			state.VelocityY = -state.VelocityY
		}
	case "horizontal-sweeper":
		if state.Phase == 0 {
			state.X += state.VelocityX
			switch {
			case state.X <= kind.MotionParameters["left_edge"]:
				state.X = kind.MotionParameters["left_reset"]
				resetSweeperDirection(state, kind, 0)
				result.VariantChanged = true
			case state.X >= kind.MotionParameters["right_edge"]:
				state.X = kind.MotionParameters["right_reset"]
				resetSweeperDirection(state, kind, 1)
				result.VariantChanged = true
			case state.X == kind.MotionParameters["left_attack_x"] || state.X == kind.MotionParameters["right_attack_x"]:
				state.Phase, state.Attacking = 1, true
				state.Animation = NewAnimation(variant.AttackAnimation)
				result.AttackStarted = true
			}
		} else if state.Animation.Remaining == 0 {
			state.Phase, state.Attacking = 0, false
			if state.X <= 160 {
				state.X = 104
				resetSweeperDirection(state, kind, 1)
			} else {
				state.X = 216
				resetSweeperDirection(state, kind, 0)
			}
			result.VariantChanged = true
		} else {
			state.Phase++
			if state.Phase == kind.MotionParameters["attack_shot_tick"] {
				setFixedShots(&result, *state, variant)
			}
		}
	case "vertical-oscillator":
		if state.VelocityY < 0 {
			state.Y--
		} else {
			state.Y++
		}
		worldY := state.Y + input.ScrollY
		if state.VelocityY >= 0 && worldY >= kind.MotionParameters["maximum_world_y"] {
			state.VelocityY = -1
			state.Animation = NewAnimation(variant.ReverseAnimation)
		} else if state.VelocityY < 0 && worldY <= kind.MotionParameters["minimum_world_y"] {
			state.VelocityY = 1
			state.Animation = NewAnimation(variant.Animation)
		}
		if fixedFireOverflow(state, kind.MotionParameters["fire_rate"], random) {
			setFixedShots(&result, *state, variant)
			// The projectile constructor consumes a second random number.
			if random != nil {
				result.ShotDelay = int(random.Next() & 31)
			}
		}
	case "extending-beam":
		advance := state.Phase != 0
		if state.Phase == 0 {
			state.PhaseDirection = 1
			advance = fixedFireOverflow(state, kind.MotionParameters["fire_rate"], random)
		}
		if advance {
			state.Phase += state.PhaseDirection
		}
		if state.Phase == kind.MotionParameters["maximum_phase"] {
			state.PhaseDirection = -state.PhaseDirection
		}
		spacing := kind.MotionParameters["phase_spacing"]
		left, right := state.X, state.X+state.Phase*spacing+16
		if variant.ID == 1 {
			left, right = state.X-state.Phase*spacing, state.X+16
		}
		result.ContactRectangle = [4]int{left, state.Y + 8, right, state.Y + 24}
		result.ContactDamage = kind.ContactDamage
	}
	return result
}

func advanceFixedAnimation(state *FixedSpriteState, animation visualassets.ActorAnimation) {
	if animation.Ending == "remove" && state.Animation.Frame == len(animation.Frames)-1 && state.Animation.Remaining == 1 {
		state.Animation.Remaining, state.Removed = 0, true
		return
	}
	state.Animation.Advance(animation)
}

func resetSweeperDirection(state *FixedSpriteState, kind visualassets.FixedSpriteKind, variantID int) {
	state.VelocityX = -state.VelocityX
	state.Variant = variantID
	state.Animation = NewAnimation(FixedSpriteAnimation(*state, kind))
}

func fixedFireOverflow(state *FixedSpriteState, rate int, random *RandomState) bool {
	sum := int(state.FireAccumulator) + rate
	state.FireAccumulator = uint8(sum)
	if sum < 256 {
		return false
	}
	if random != nil {
		state.FireAccumulator = uint8(random.Next() & 63)
	}
	return true
}

func setFixedShots(result *FixedSpriteEvents, state FixedSpriteState, variant visualassets.FixedSpriteVariant) {
	result.ShotMode, result.ShotSprite = variant.ShotMode, variant.ShotSprite
	result.ShotX, result.ShotY = state.X+variant.ShotOffsetX, state.Y+variant.ShotOffsetY
	result.ShotSpeed, result.ShotMotionBudget = variant.ShotSpeed, variant.ShotMotionBudget
	result.ShotAnimation = variant.ShotAnimation
	for _, direction := range variant.ShotDirections {
		if result.ShotCount < len(result.ShotDirections) {
			result.ShotDirections[result.ShotCount] = direction
			result.ShotCount++
		}
	}
}
