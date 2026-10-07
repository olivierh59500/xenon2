package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

type FixedProjectileInputs struct {
	ScrollDelta, PlayerX, PlayerY int
	PlayerBounds                  CollisionRect
	CanHitPlayer, Invulnerable    bool
}

// FixedProjectileEvents keeps contact damage and invulnerable-hit explosions
// separate from movement, so World can preserve its original update phases.
type FixedProjectileEvents struct {
	PlayerDamage           int
	Explosion              bool
	ExplosionX, ExplosionY int
	Collision              CollisionRect
}

// TurningFixedProjectile preserves fractional travel and the source's two
// horizontal turning thresholds. The collision image precedes the turn.
type TurningFixedProjectile struct {
	Motion  DirectionalProjectile
	Sprite  string
	Removed bool
}

func NewTurningFixedProjectile(event FixedSpriteEvents, art *visualassets.FixedProjectileArtwork, fractionX, fractionY uint16) (TurningFixedProjectile, error) {
	if art == nil || art.Kind != "turning-projectile" || event.ShotCount != 1 {
		return TurningFixedProjectile{}, fmt.Errorf("invalid turning projectile event")
	}
	if event.ShotDirections[0] != 3 && event.ShotDirections[0] != 5 {
		return TurningFixedProjectile{}, fmt.Errorf("invalid turning projectile heading")
	}
	return TurningFixedProjectile{Motion: DirectionalProjectile{X: int32(event.ShotX)<<16 | int32(fractionX), Y: int32(event.ShotY)<<16 | int32(fractionY), Direction: uint8(event.ShotDirections[0]), Speed: event.ShotMotionBudget}, Sprite: event.ShotSprite}, nil
}

func (s *TurningFixedProjectile) Advance(art *visualassets.FixedProjectileArtwork, input FixedProjectileInputs, box visualassets.CollisionBox) (FixedProjectileEvents, error) {
	result := FixedProjectileEvents{Collision: CollisionRect{Right: -1, Bottom: -1}}
	if s.Removed {
		return result, nil
	}
	alive, err := s.Motion.Advance(input.ScrollDelta)
	if err != nil {
		return result, err
	}
	if !alive {
		s.Removed = true
		return result, nil
	}
	x, y := int(s.Motion.X>>16), int(s.Motion.Y>>16)
	result.Collision = ActorCollisionRect(box, x, y)
	if input.CanHitPlayer && result.Collision.Intersects(input.PlayerBounds) {
		s.Removed = true
		if !input.Invulnerable {
			result.PlayerDamage = art.ContactDamage
		}
		return result, nil
	}
	if x < 128 {
		s.Motion.Direction, s.Sprite = 3, art.TurningSprites[0]
	}
	if x >= 192 {
		s.Motion.Direction, s.Sprite = 5, art.TurningSprites[1]
	}
	return result, nil
}

// AnimatedAimingFixedProjectile is a damageable moving actor. Its lifetime
// counter limits it independently of screen bounds, and every eighth pass
// turns one octant toward the current ship, resolving opposite turns by X parity.
type AnimatedAimingFixedProjectile struct {
	X, Y, Timer int
	Direction   uint8
	Animation   AnimationState
	Removed     bool
}

func NewAnimatedAimingFixedProjectile(event FixedSpriteEvents, art *visualassets.FixedProjectileArtwork) (AnimatedAimingFixedProjectile, error) {
	if art == nil || art.Kind != "animated-aiming-projectile" || len(art.HeadingAnimations) != 8 || event.ShotCount != 1 {
		return AnimatedAimingFixedProjectile{}, fmt.Errorf("invalid animated aiming projectile event")
	}
	direction := uint8(event.ShotDirections[0])
	if direction >= 8 {
		return AnimatedAimingFixedProjectile{}, fmt.Errorf("invalid animated projectile heading")
	}
	return AnimatedAimingFixedProjectile{X: event.ShotX, Y: event.ShotY, Timer: event.ShotDelay, Direction: direction, Animation: NewAnimation(art.HeadingAnimations[direction])}, nil
}

func (s AnimatedAimingFixedProjectile) Sprite(art *visualassets.FixedProjectileArtwork) string {
	return s.Animation.Sprite(art.HeadingAnimations[s.Direction])
}

func (s *AnimatedAimingFixedProjectile) Advance(art *visualassets.FixedProjectileArtwork, input FixedProjectileInputs, box func(string) visualassets.CollisionBox, image func(string) visualassets.SpriteRegion) FixedProjectileEvents {
	result := FixedProjectileEvents{Collision: CollisionRect{Right: -1, Bottom: -1}}
	if s.Removed {
		return result
	}
	s.Animation.Advance(art.HeadingAnimations[s.Direction])
	s.Timer = int(int16(s.Timer + 1))
	if s.Timer >= art.Lifetime {
		s.Removed = true
		s.setExplosion(&result, image(s.Sprite(art)))
		return result
	}
	if s.Timer&7 == 0 {
		wanted := AimDirection(input.PlayerX-s.X, input.PlayerY-s.Y)
		delta := (int(wanted) - int(s.Direction) + 8) & 7
		if delta != 0 {
			if delta > 4 || delta == 4 && s.X&1 == 0 {
				s.Direction = (s.Direction - 1) & 7
			} else {
				s.Direction = (s.Direction + 1) & 7
			}
			s.Animation = NewAnimation(art.HeadingAnimations[s.Direction])
		}
	}
	dx, dy := homingX[s.Direction], homingY[s.Direction]+input.ScrollDelta
	s.X = int(int16(s.X + dx))
	s.Y = int(int16(s.Y + dy))
	result.Collision = ActorCollisionRect(box(s.Sprite(art)), s.X, s.Y)
	// The source point probe receives the still-live displacement registers.
	// Its separately updated actor rectangle remains available to player shots.
	probe := CollisionRect{Left: dx, Top: dy, Right: dx, Bottom: dy}
	if input.CanHitPlayer && probe.Intersects(input.PlayerBounds) {
		s.Removed = true
		if input.Invulnerable {
			s.setExplosion(&result, image(s.Sprite(art)))
		} else {
			result.PlayerDamage = art.ContactDamage
		}
	}
	return result
}

func (s AnimatedAimingFixedProjectile) setExplosion(result *FixedProjectileEvents, region visualassets.SpriteRegion) {
	result.Explosion = true
	result.ExplosionX = s.X - region.AnchorX + region.Width/2
	result.ExplosionY = s.Y - region.AnchorY + (region.Height-1)/2
}
