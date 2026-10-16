package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

type FifthGuardianPartState struct {
	X, Y                                  int
	Clock, MoveRemaining, Health          int
	Heading                               uint8
	FireAccumulator, SecondaryAccumulator uint8
	Sprite                                string
	Active, Destroyed                     bool
	Animation                             AnimationState
	Collision                             CollisionRect
}

type FifthMiddleGuardianState struct {
	Parts    [10]FifthGuardianPartState
	Defeated bool
	Flash    bool
}

type FifthGuardianShot struct {
	InitialClock int
	X, Y, Speed  int
	Heading      uint8
	Animation    string
}

type FifthGuardianLaser struct{ X, Y, Speed int }

type FifthGuardianEvents struct {
	Explosions     int
	Shots          []FifthGuardianShot
	Lasers         []FifthGuardianLaser
	MaximumScroll  int
	Sound2, Sound1 int
	Score          int
	Cash           int
	Defeated       bool
}

func NewFifthMiddleGuardianState(group *visualassets.GuardianGroup, initialY int) (FifthMiddleGuardianState, error) {
	var state FifthMiddleGuardianState
	if group == nil || len(group.Components) != 10 {
		return state, fmt.Errorf("fifth middle guardian requires ten parts")
	}
	for index, part := range group.Components {
		state.Parts[index] = FifthGuardianPartState{X: 112, Y: initialY, MoveRemaining: 4, Health: part.Health, Heading: 4, Sprite: part.Sprite, Active: true, Collision: CollisionRect{Right: -1, Bottom: -1}}
	}
	return state, nil
}

// Advance visits the source tail-inserted controller and nine mounts in order.
func (s *FifthMiddleGuardianState) Advance(group *visualassets.GuardianGroup, scroll, delta, maximum, playerX, playerY int, random *RandomState) FifthGuardianEvents {
	event := FifthGuardianEvents{MaximumScroll: max(maximum, group.MotionParameters["maximum_scroll"]), Sound2: -1, Sound1: -1}
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
		case "middle-body-controller":
			state.Y += delta
			if state.MoveRemaining != 0 {
				event.Sound2 = 22
				if state.MoveRemaining > 0 {
					state.Y++
					state.MoveRemaining--
				} else {
					state.Y--
					state.MoveRemaining++
				}
			} else {
				value := random.Next()
				if uint8(value) < 15 {
					distance := state.Y + scroll - group.MotionParameters["minimum_world_y"]
					positive := value&1 != 0
					if positive {
						distance = group.MotionParameters["maximum_world_y"] - state.Y - scroll
					}
					if distance != 0 {
						amount, _ := random.Below(uint16(distance))
						state.MoveRemaining = int(int16(amount + 1))
						if !positive {
							state.MoveRemaining = int(int16(-int16(state.MoveRemaining)))
						}
					}
				}
			}
			if state.Clock == 0 {
				sum := int(state.FireAccumulator) + group.MotionParameters["body_fire_rate"]
				state.FireAccumulator = uint8(sum)
				if sum >= 256 {
					state.FireAccumulator = uint8(random.Next() & 63)
					state.Clock = 1
				}
			} else {
				state.Clock++
			}
			if state.Clock == 9 {
				state.Clock = 0
			} else if state.Clock == 3 {
				event.Sound2 = 132
				event.Lasers = append(event.Lasers, FifthGuardianLaser{X: 150, Y: state.Y + 48, Speed: group.MotionParameters["body_laser_speed"]})
			}
			sum := int(state.SecondaryAccumulator) + group.MotionParameters["body_laser_rate"]
			state.SecondaryAccumulator = uint8(sum)
			if sum >= 256 {
				state.SecondaryAccumulator = uint8(random.Next() & 63)
				leftClock, rightClock := int(random.Next()&31), int(random.Next()&31)
				event.Shots = append(event.Shots, FifthGuardianShot{InitialClock: leftClock, X: 110, Y: state.Y + 48, Heading: 6, Animation: "middle-side-shot"}, FifthGuardianShot{InitialClock: rightClock, X: 210, Y: state.Y + 48, Heading: 2, Animation: "middle-side-shot"})
			}
		case "follow-middle-body":
			state.Heading = AimDirection(playerX-state.X, playerY-state.Y)
			if len(part.HeadingFrames) > int(state.Heading) {
				state.Sprite = part.HeadingFrames[state.Heading]
			}
			sum := int(state.FireAccumulator) + group.MotionParameters["mount_fire_rate"]
			state.FireAccumulator = uint8(sum)
			if sum >= 256 {
				state.FireAccumulator = uint8(random.Next() & 63)
				event.Shots = append(event.Shots, FifthGuardianShot{X: state.X + 8, Y: state.Y + 8, Speed: group.MotionParameters["mount_shot_speed"], Heading: state.Heading, Animation: "mount-shot"})
			}
		case "middle-central-cannon":
			state.Clock += 256 - state.Health
			if state.Clock >= 1536 {
				state.Clock -= 1536
			}
			selector := (state.Clock >> 8) % 6
			if len(part.HeadingFrames) > selector {
				state.Sprite = part.HeadingFrames[selector]
			}
		case "middle-corner":
			selector := (state.Y + scroll) & 3
			if len(part.HeadingFrames) > selector {
				state.Sprite = part.HeadingFrames[selector]
			}
		}
	}
	return event
}

func (s *FifthMiddleGuardianState) Damage(part int, amount uint16) FifthGuardianEvents {
	event := FifthGuardianEvents{Sound2: -1, Sound1: -1}
	if part < 0 || part >= len(s.Parts) || s.Defeated || s.Parts[part].Destroyed {
		return event
	}
	switch part {
	case 1, 2, 3, 4:
		result := ApplyEnemyDamage(uint16(s.Parts[part].Health), amount)
		s.Parts[part].Health = int(result.Health)
		if result.Destroyed {
			s.Parts[part].Destroyed = true
			event.Score = 200
			event.Explosions = 1
			event.Shots = nil
		}
	case 5:
		s.Flash = true
		result := ApplyEnemyDamage(uint16(s.Parts[part].Health), amount)
		s.Parts[part].Health = int(result.Health)
		if result.Destroyed {
			s.Defeated = true
			event.Defeated = true
			event.Score = 1500
			event.Cash = 5
			event.Explosions = 20
			for index := range s.Parts {
				s.Parts[index].Active = false
			}
		}
	}
	return event
}
