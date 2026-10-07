package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

// FourthCrawlerState stores world coordinates; only its collision and display
// coordinates subtract the current scroll. A boundary escape resets its nest.
type FourthCrawlerState struct {
	X, Y, HomeX, HomeY                   int
	Variant, Nest, Phase, VerticalChoice int
	Health                               uint16
	Animation                            AnimationState
	Removed, Visible, Flash              bool
	Collision                            CollisionRect
	Fire                                 EnemyFireState
}

type FourthCrawlerInput struct{ ScrollY, MaximumScrollY, PlayerX, PlayerY int }
type FourthCrawlerEvents struct {
	Reset, Destroyed       bool
	Score                  int
	LargeCash              bool
	CashDirection          uint8
	CashX, CashY           int
	ExplosionX, ExplosionY int
}

func NewFourthCrawler(record visualassets.FixedEncounter, art visualassets.FixedSpriteKind) (FourthCrawlerState, error) {
	variant := record.Variant
	if variant < 0 || variant >= len(art.Variants) || record.State1 < 0 || record.State1 >= len(art.MotionTables["cover_x"]) {
		return FourthCrawlerState{}, fmt.Errorf("invalid fourth crawler record")
	}
	v := art.Variants[variant]
	x, y := record.X+v.OriginOffsetX, record.Y+v.OriginOffsetY
	clock := uint16(art.MotionParameters["emitter_clock"])
	return FourthCrawlerState{X: x, Y: y, HomeX: x, HomeY: y, Variant: variant, Nest: record.State1, Health: uint16(art.Health), Animation: NewAnimation(v.Animation), Collision: CollisionRect{Left: 1000, Right: 1000}, Fire: EnemyFireState{Accumulator: uint8(clock >> 8), Rate: uint8(clock)}}, nil
}

func (s *FourthCrawlerState) Sprite(art visualassets.FixedSpriteKind) string {
	return s.Animation.Sprite(art.Variants[s.Variant].Animation)
}

func (s *FourthCrawlerState) Advance(art visualassets.FixedSpriteKind, input FourthCrawlerInput, box func(string) visualassets.CollisionBox, nextRandom func() uint32) (FourthCrawlerEvents, error) {
	var event FourthCrawlerEvents
	if s.Removed {
		return event, nil
	}
	if input.MaximumScrollY+art.MotionParameters["clip_margin"] < s.Y {
		s.Removed = true
		s.Visible = false
		return event, nil
	}
	s.Animation.Advance(art.Variants[s.Variant].Animation)
	s.Visible = s.Y-input.ScrollY <= art.MotionParameters["draw_bottom"]
	s.Flash = false
	if s.Phase == 0 {
		if nextRandom == nil {
			return event, fmt.Errorf("fourth crawler needs the shared random stream")
		}
		if int(uint8(nextRandom())) < art.MotionParameters["activation_chance"] {
			s.Phase = 1
			s.Animation.Remaining = 1
		}
		return event, nil
	}
	speed := art.MotionParameters["speed"]
	if s.Variant == 0 {
		s.X += speed
	} else {
		s.X -= speed
	}
	if s.Phase == 1 {
		if s.Animation.Remaining == 0 {
			s.Phase = 2
			s.VerticalChoice = input.PlayerY - s.Y + input.ScrollY
		}
	} else {
		if s.X >= 40 && s.X <= 280 {
			if s.VerticalChoice < 0 {
				s.Y -= speed
			} else {
				s.Y += speed
			}
		}
		if s.X < -32 || s.X >= 352 {
			s.reset(art)
			event.Reset = true
			return event, nil
		}
	}
	s.Collision = ActorCollisionRect(box(s.Sprite(art)), s.X, s.Y-input.ScrollY)
	return event, nil
}

func (s *FourthCrawlerState) reset(art visualassets.FixedSpriteKind) {
	s.X, s.Y = s.HomeX, s.HomeY
	s.Phase = 0
	s.Health = uint16(art.Health)
	s.Animation = NewAnimation(art.Variants[s.Variant].Animation)
	s.Collision.Left, s.Collision.Right = 1000, 1000
}

func (s *FourthCrawlerState) Strike(art visualassets.FixedSpriteKind, amount uint16, scrollY int, image visualassets.SpriteRegion, nextRandom func() uint32) FourthCrawlerEvents {
	var event FourthCrawlerEvents
	if s.Removed {
		return event
	}
	damage := ApplyEnemyDamage(s.Health, amount)
	s.Health = damage.Health
	if !damage.Destroyed {
		s.Flash = true
		return event
	}
	event.Destroyed, event.Score, event.LargeCash = true, art.Score, true
	event.CashX, event.CashY = s.X, s.Y-scrollY
	event.ExplosionX, event.ExplosionY = s.X-image.AnchorX+image.Width/2, s.Y-image.AnchorY+(image.Height-1)/2-scrollY
	if nextRandom != nil {
		event.CashDirection = uint8(nextRandom()) & 7
	}
	s.reset(art)
	return event
}
