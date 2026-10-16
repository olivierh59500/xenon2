package engine

import "xenon2/internal/visualassets"

type FourthFallingActor struct {
	X, Y, Variant, Phase         int
	Health                       uint16
	PrimaryClock, SecondaryClock uint8
	Animation                    AnimationState
	Collision                    CollisionRect
	Removed                      bool
}

type FourthFallingEvents struct {
	SpawnPod                bool
	PodSide, PodX, PodY     int
	Shot                    bool
	ShotX, ShotY, ShotSpeed int
	ShotDirection           uint8
}

func NewFourthFallingActor(variant int, art *visualassets.FourthStageArt, residue ActorResidue, fireRandom uint32) FourthFallingActor {
	v := art.Variants[variant]
	return FourthFallingActor{X: v.X, Y: -32, Variant: variant, Health: uint16(art.Health), PrimaryClock: uint8(fireRandom), SecondaryClock: residue.FireRate(), Animation: NewAnimation(v.Animation)}
}

func (s *FourthFallingActor) Sprite(art *visualassets.FourthStageArt) string {
	return s.Animation.Sprite(art.Variants[s.Variant].Animation)
}

func (s *FourthFallingActor) Advance(art *visualassets.FourthStageArt, scrollDelta int, box func(string) visualassets.CollisionBox, nextRandom func() uint32) FourthFallingEvents {
	var event FourthFallingEvents
	if s.Removed {
		return event
	}
	s.Animation.Advance(art.Variants[s.Variant].Animation)
	if s.Variant&1 != 0 {
		s.Y += scrollDelta + art.GunSpeed
		if s.Y >= 200 {
			s.Removed = true
			return event
		}
		if fourthFire(&s.PrimaryClock, art.GunFireRate, nextRandom) {
			event.Shot = true
			event.ShotX, event.ShotY = s.X+26, s.Y+26
			event.ShotDirection = 10
			if s.Variant == 3 {
				event.ShotX = s.X - 26
				event.ShotDirection = 13
			}
			event.ShotSpeed = art.GunShotSpeed
		}
	} else {
		s.Y += scrollDelta + art.HatchSpeed
		if s.Y >= 200 {
			s.Removed = true
			return event
		}
		s.Phase++
		if s.Phase == 8 && fourthFire(&s.PrimaryClock, art.HatchFireRate, nextRandom) {
			event.SpawnPod = true
			event.PodSide = s.Variant / 2
			event.PodX, event.PodY = art.Variants[s.Variant].PodX, s.Y-30
		}
		if s.Phase == 12 {
			s.Phase = 0
		}
		if fourthFire(&s.SecondaryClock, art.SecondaryFireRate, nextRandom) {
			event.Shot = true
			event.ShotX = s.X + 20
			if s.Variant == 2 {
				event.ShotX = s.X - 16
			}
			event.ShotY, event.ShotDirection, event.ShotSpeed = s.Y, 4, art.ShotSpeed
		}
	}
	s.Collision = ActorCollisionRect(box(s.Sprite(art)), s.X, s.Y)
	return event
}
