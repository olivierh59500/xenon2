package engine

// CannonMountState retains the held-ready and four recoil frames. Animation
// advances before the materialization guard, matching the weapon's own update.
type CannonMountState struct {
	Phase int
}

// Tick returns true on the designated firing frame. A pulse starts recoil
// immediately, while the actual cannonball is created on the following pass.
func (s *CannonMountState) Tick(pulse, materializing bool) bool {
	if s.Phase != 0 {
		s.Phase = (s.Phase + 1) % 5
	}
	if materializing {
		return false
	}
	if s.Phase == 2 {
		return true
	}
	if s.Phase == 0 && pulse {
		s.Phase = 1
	}
	return false
}

// MissileMountState retains the held-ready frame and six launch frames. The
// first pulse releases the held frame; two missiles appear on the next pass.
type MissileMountState struct {
	Phase        int
	PendingStart bool
}

func (s *MissileMountState) Tick(pulse, materializing bool) bool {
	if s.PendingStart {
		s.Phase = 1
		s.PendingStart = false
	} else if s.Phase != 0 {
		s.Phase = (s.Phase + 1) % 7
	}
	if materializing {
		return false
	}
	if s.Phase == 1 {
		return true
	}
	if s.Phase == 0 && pulse {
		s.PendingStart = true
	}
	return false
}

// CannonBallMotion is the whole-pixel movement of an animated, rectangle-hit
// cannonball. Its own collision prefix supplies the rectangle after movement.
type CannonBallMotion struct {
	X, Y int
}

func (s *CannonBallMotion) Advance(shipDestroyed bool) bool {
	if shipDestroyed {
		return false
	}
	s.Y -= 10
	return s.Y >= 0
}

const CannonBallDamage = 2

// AppendLauncherMissiles emits the original pair around an additional mount.
// These missiles use animated artwork and ordinary point-hit bullet movement.
func AppendLauncherMissiles(dst []SmallShot, mountX, mountY int, spriteName string) []SmallShot {
	return append(dst,
		SmallShot{X: mountX - 3, Y: mountY - 16, VelocityY: -9, Damage: 1, SpriteName: spriteName},
		SmallShot{X: mountX + 5, Y: mountY - 16, VelocityY: -9, Damage: 1, SpriteName: spriteName},
	)
}
