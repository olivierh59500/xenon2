package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

// ThirdGuardianPartState keeps the independent animation and articulated arm
// state. The indexed parts update in their original tail-insertion order.
type ThirdGuardianPartState struct {
	X, Y                          int
	Animation                     AnimationState
	Sprite                        string
	Arc                           GuardianArcState
	Extension, ExtensionDirection int
	FireAccumulator               uint8
	ActiveEye                     bool
	Collision                     CollisionRect
	Collidable                    bool
}

type ThirdGuardianState struct {
	Parts            [17]ThirdGuardianPartState
	Flight           PathMotionState
	EyeHealth        [2]uint16
	EyesRemaining    int
	Defeated, Flash  bool
	ResidualFireRate uint8
}

type ThirdGuardianShot struct {
	X, Y, Speed int
	Direction   uint8
	Animation   string
}
type ThirdGuardianEvents struct {
	Shots        []ThirdGuardianShot
	Sound        string
	ActiveSignal bool
}

func NewThirdGuardianState(group *visualassets.GuardianGroup) (ThirdGuardianState, error) {
	var state ThirdGuardianState
	if group == nil || len(group.Components) != 17 || group.Path == nil {
		return state, fmt.Errorf("third middle guardian resources are incomplete")
	}
	motion, err := NewPathMotion(group.Path, PathMotionConfig{Budget: group.Components[0].PathBudget})
	if err != nil {
		return state, err
	}
	motion.X += int32(group.PathOffsetX) << 16
	motion.Y += int32(group.PathOffsetY) << 16
	state.Flight = motion
	state.EyesRemaining = 2
	for index, part := range group.Components {
		s := &state.Parts[index]
		s.X, s.Y = part.InitialX, part.InitialWorldY
		s.Animation = NewAnimation(part.Animation)
		s.Sprite = s.Animation.Sprite(part.Animation)
		s.Arc = GuardianArcState{X: int32(s.X) << 16, Y: int32(s.Y) << 16, AngularVelocity: part.AngularVelocityFixed, Budget: part.PathBudget}
		s.Collision = CollisionRect{Right: -1, Bottom: -1}
		if part.Behavior == "eye-turret" {
			eye := index - 3
			if eye >= 0 && eye < 2 {
				state.EyeHealth[eye] = uint16(part.Health)
			}
		}
	}
	return state, nil
}

func (s *ThirdGuardianState) Advance(group *visualassets.GuardianGroup, sine *[256]int8, frame uint64, playerX, playerY int, random *RandomState) (ThirdGuardianEvents, error) {
	var event ThirdGuardianEvents
	if s.Defeated {
		return event, nil
	}
	s.Flash = false
	event.ActiveSignal = true
	for index, part := range group.Components {
		state := &s.Parts[index]
		state.Collidable = true
		switch part.Behavior {
		case "head-flight-fire":
			state.Animation.Advance(part.Animation)
			var next func() uint16
			if random != nil {
				next = func() uint16 { return uint16(random.Next()) }
			}
			if err := s.Flight.Advance(group.Path, sine, next); err != nil {
				return event, err
			}
			state.X, state.Y = int(s.Flight.X>>16), int(s.Flight.Y>>16)
			if s.ResidualFireRate != 0 {
				fire := EnemyFireState{Rate: s.ResidualFireRate, Accumulator: state.FireAccumulator}
				var next32 func() uint32
				if random != nil {
					next32 = random.Next
				}
				shot, fired, err := fire.Tick(next32, playerX-state.X, playerY-state.Y)
				if err != nil {
					return event, err
				}
				state.FireAccumulator = fire.Accumulator
				if fired {
					event.Shots = append(event.Shots, ThirdGuardianShot{X: state.X, Y: state.Y, Speed: shot.Speed, Direction: shot.Direction})
				}
			}
			if thirdFireCarry(state, group.MotionParameters["head_fire_rate"], random) {
				event.Shots = append(event.Shots, ThirdGuardianShot{X: state.X + 16, Y: state.Y + 60, Speed: group.MotionParameters["head_shot_speed"], Direction: AimDirection(playerX-state.X, playerY-state.Y), Animation: "middle-head-shot"})
			}
		case "follow-head", "eye-turret", "arm-base":
			animation := part.Animation
			if part.Behavior == "eye-turret" && state.ActiveEye {
				animation = part.ActiveAnimation
			}
			state.Animation.Advance(animation)
			if part.Behavior == "eye-turret" && state.ActiveEye && state.Animation.Remaining == 0 {
				state.ActiveEye = false
				state.Animation = NewAnimation(part.Animation)
			}
			state.X, state.Y = s.Parts[0].X+part.OffsetX, s.Parts[0].Y+part.OffsetY
			state.Arc.X, state.Arc.Y = int32(state.X)<<16, int32(state.Y)<<16
			if part.Behavior == "eye-turret" {
				eye := index - 3
				if int16(s.EyeHealth[eye]) <= 0 {
					state.Sprite = part.DestroyedSprite
					state.Animation.Remaining = 0
					state.Collision = CollisionRect{Right: -1, Bottom: -1}
					state.Collidable = false
					continue
				}
				if state.Animation.Remaining == 0 {
					if thirdFireCarry(state, group.MotionParameters["eye_fire_rate"], random) {
						state.Animation = NewAnimation(part.ActiveAnimation)
						state.ActiveEye = true
						state.Sprite = state.Animation.Sprite(part.ActiveAnimation)
					} else {
						state.Sprite = state.Animation.Sprite(part.Animation)
					}
					state.Collision = CollisionRect{Right: -1, Bottom: -1}
					state.Collidable = false
					continue
				}
				state.Sprite = state.Animation.Sprite(part.ActiveAnimation)
			}
			if part.Behavior == "arm-base" {
				next := &s.Parts[index+1]
				if state.Extension == 0 {
					if random != nil && uint8(random.Next()) < 3 {
						event.Sound = "synthesized-effect-15"
						state.Extension, state.ExtensionDirection = 1, 1
						heading := uint8(AimDirection(playerX-state.X, playerY-state.Y)*32 - 96)
						velocity := int32((random.Next()&31)+32) << 10
						if index == 5 {
							heading += 64
							velocity = -velocity
						}
						next.Arc.AngleFixed = int32(heading) << 16
						next.Arc.AngularVelocity = velocity
						next.Arc.AngularAcceleration = 0
						next.Arc.Budget = 1
					} else {
						next.Arc.Budget = 0
						next.Arc.AngleFixed = (next.Arc.AngleFixed & 65535) | 64<<16
					}
				} else {
					state.Extension += state.ExtensionDirection
					if state.Extension == 14 {
						state.ExtensionDirection = -state.ExtensionDirection
					}
					next.Arc.Budget = state.Extension
				}
			}
		case "arm-link", "arm-tip":
			parent := s.Parts[part.ParentIndex]
			state.Arc.X, state.Arc.Y = parent.Arc.X, parent.Arc.Y
			angle, velocity, acceleration := state.Arc.AngleFixed, state.Arc.AngularVelocity, state.Arc.AngularAcceleration
			state.Arc.Advance(sine)
			state.X, state.Y = int(state.Arc.X>>16), int(state.Arc.Y>>16)
			if part.Behavior == "arm-link" {
				next := &s.Parts[index+1]
				next.Arc.AngleFixed, next.Arc.AngularVelocity, next.Arc.AngularAcceleration, next.Arc.Budget = state.Arc.AngleFixed, state.Arc.AngularVelocity, state.Arc.AngularAcceleration, state.Arc.Budget
				state.Arc.AngleFixed, state.Arc.AngularVelocity, state.Arc.AngularAcceleration = angle, velocity, acceleration
			} else {
				heading := int((uint32(state.Arc.AngleFixed)>>16)>>5) & 7
				state.Sprite = part.HeadingFrames[heading*2+int((frame&4)>>2)]
				state.Animation.Remaining = 0
				if state.Arc.Budget == 14 {
					event.Sound = "synthesized-effect-12"
				}
			}
		default:
			return event, fmt.Errorf("unknown third guardian behavior %q", part.Behavior)
		}
		if part.Behavior != "eye-turret" && part.Behavior != "arm-tip" {
			state.Sprite = state.Animation.Sprite(part.Animation)
		}
	}
	return event, nil
}

func thirdFireCarry(state *ThirdGuardianPartState, rate int, random *RandomState) bool {
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

type ThirdGuardianDamageEvents struct {
	Applied, EyeDestroyed, Defeated bool
	CashPairs                       int
}

func (s *ThirdGuardianState) StrikeEye(eye int, amount uint16) ThirdGuardianDamageEvents {
	var event ThirdGuardianDamageEvents
	if eye < 0 || eye >= 2 || int16(s.EyeHealth[eye]) <= 0 || s.Defeated {
		return event
	}
	s.Flash, event.Applied = true, true
	before := s.EyeHealth[eye]
	s.EyeHealth[eye] -= amount
	if before <= amount {
		s.EyesRemaining--
		event.EyeDestroyed = true
		if s.EyesRemaining == 0 {
			s.Defeated = true
			event.Defeated, event.CashPairs = true, 5
		}
	}
	return event
}
