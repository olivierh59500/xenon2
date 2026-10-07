package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

type FifthFinalGuardianState struct {
	Parts           [22]FifthGuardianPartState
	OuterRemaining  int
	CoreHealth      uint16
	Defeated, Flash bool
}

type FifthMouthCreature struct {
	X, Y, Speed int
	Lower       bool
}

type FifthFinalGuardianEvents struct {
	FifthGuardianEvents
	Creatures         []FifthMouthCreature
	ExtraBackdropStep int
}

func NewFifthFinalGuardianState(group *visualassets.GuardianGroup) (FifthFinalGuardianState, error) {
	var state FifthFinalGuardianState
	if group == nil || len(group.Components) != 22 {
		return state, fmt.Errorf("fifth final guardian requires twenty-two parts")
	}
	state.OuterRemaining = 18
	state.CoreHealth = uint16(group.Components[0].Health)
	for index, part := range group.Components {
		animation := NewAnimation(part.Animation)
		state.Parts[index] = FifthGuardianPartState{X: 48, Y: -320, MoveRemaining: 4, Health: part.Health, Heading: 4, Sprite: animation.Sprite(part.Animation), Animation: animation, Active: true, Collision: CollisionRect{Right: -1, Bottom: -1}}
	}
	return state, nil
}

func (s *FifthFinalGuardianState) Advance(group *visualassets.GuardianGroup, scroll, delta, baseStep, maximum, frame, playerX, playerY int, random *RandomState) FifthFinalGuardianEvents {
	event := FifthFinalGuardianEvents{FifthGuardianEvents: FifthGuardianEvents{MaximumScroll: 416, Sound2: -1, Sound1: -1}}
	if s.Defeated {
		return event
	}
	s.Flash = false
	for index, part := range group.Components {
		state := &s.Parts[index]
		if !state.Active {
			continue
		}
		if index != 0 {
			state.X = s.Parts[0].X + part.OffsetX
			state.Y = s.Parts[0].Y + part.OffsetY
		}
		if state.Destroyed {
			state.Sprite = part.DestroyedSprite
			continue
		}
		switch part.Behavior {
		case "final-body-controller":
			if scroll == 0 {
				event.ExtraBackdropStep = baseStep
			}
			state.Y += delta
			if state.Clock != 0 {
				if state.Clock > 0 {
					state.Clock += 2
				} else {
					state.Clock -= 2
				}
				clock := state.Clock
				if clock < 0 {
					clock = -clock
				}
				if clock == 60 {
					state.Clock = 0
				} else if clock == 20 {
					event.Sound1 = 19
				} else if clock == 40 {
					lower := state.Clock < 0
					y := state.Y + 101
					if lower {
						y = state.Y + 245
					}
					event.Creatures = append(event.Creatures, FifthMouthCreature{X: state.X + 119, Y: y, Speed: group.MotionParameters["mouth_creature_speed"], Lower: lower})
				}
			} else {
				sum := int(state.FireAccumulator) + group.MotionParameters["mouth_fire_rate"]
				state.FireAccumulator = uint8(sum)
				if sum >= 256 {
					state.FireAccumulator = uint8(random.Next() & 63)
					state.Clock = 2
					if frame&1 == 0 {
						state.Clock = -2
					}
				}
			}
		case "final-mount":
			state.Heading = AimDirection(playerX-state.X, playerY-state.Y)
			if len(part.HeadingFrames) > int(state.Heading) {
				state.Sprite = part.HeadingFrames[state.Heading]
			}
			sum := int(state.FireAccumulator) + group.MotionParameters["mount_fire_rate"]
			state.FireAccumulator = uint8(sum)
			if sum >= 256 {
				state.FireAccumulator = uint8(random.Next() & 63)
				event.Shots = append(event.Shots, FifthGuardianShot{X: state.X + 8, Y: state.Y + 8, Heading: state.Heading, Speed: group.MotionParameters["mount_shot_speed"], Animation: "mount-shot"})
			}
		case "final-side-turret":
			state.Animation.Advance(part.Animation)
			state.Sprite = state.Animation.Sprite(part.Animation)
			if state.Animation.Frame == group.MotionParameters["side_laser_frame"] {
				// The source emits the growing column on this penultimate display pass.
				if state.Animation.Remaining == 1 {
					event.Lasers = append(event.Lasers, FifthGuardianLaser{X: state.X - 2, Y: state.Y - 24, Speed: -group.MotionParameters["body_laser_speed"]})
				}
			}
			if state.Animation.Remaining == 0 && state.Y >= 0 {
				sum := int(state.FireAccumulator) + group.MotionParameters["side_fire_rate"]
				state.FireAccumulator = uint8(sum)
				if sum >= 256 {
					state.FireAccumulator = uint8(random.Next() & 63)
					state.Animation.Remaining = 1
				}
			}
		case "follow-final-body":
			state.Animation.Advance(part.Animation)
			state.Sprite = state.Animation.Sprite(part.Animation)
		case "final-weak-point":
			state.Collision = CollisionRect{Right: -1, Bottom: -1}
		case "barrier-band":
			// The source adds the band's height to only the low byte of Y.
			// Retain the upper byte when the bottom edge crosses a byte boundary.
			bottom := state.Y&^0xff | int(uint8(state.Y)+16)
			state.Collision = CollisionRect{Left: state.X, Top: state.Y, Right: state.X + part.Health, Bottom: bottom}
		}
	}
	return event
}

// DamagePart is the original callback after collision eligibility has already
// been checked. The exposed core drains the body's health, not its own record.
func (s *FifthFinalGuardianState) DamagePart(group *visualassets.GuardianGroup, index int, amount uint16) FifthGuardianEvents {
	event := FifthGuardianEvents{Sound2: -1, Sound1: -1}
	if index < 3 || index >= 22 || s.Defeated || s.Parts[index].Destroyed {
		return event
	}
	part := &s.Parts[index]
	if index == 21 {
		s.Flash = true
		result := ApplyEnemyDamage(s.CoreHealth, amount)
		s.CoreHealth = result.Health
		s.Parts[0].Health = int(result.Health)
		if denominator := group.MotionParameters["core_health"]; !result.Destroyed && len(group.Components[index].HeadingFrames) == 12 && denominator != 0 {
			part.Sprite = group.Components[index].HeadingFrames[int(result.Health)*11/denominator]
		}
		if result.Destroyed {
			s.Defeated = true
			event.Defeated = true
			event.Explosions = 40
			event.Cash = 10
			for i := range s.Parts {
				s.Parts[i].Active = false
			}
		}
		return event
	}
	result := ApplyEnemyDamage(uint16(part.Health), amount)
	part.Health = int(result.Health)
	if !result.Destroyed {
		return event
	}
	s.OuterRemaining--
	part.Destroyed = true
	event.Score, event.Explosions = 200, 1
	if group.Components[index].ResourceTag != 0x114 {
		part.Active = false
	}
	return event
}
