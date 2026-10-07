package engine

import "fmt"

// SmallShot is an ordinary point-collision player bullet. Its positioning
// anchor and image dimensions remain independent of its collision point.
type SmallShot struct {
	X, Y       int
	VelocityX  int
	VelocityY  int
	Damage     uint16
	SpriteName string
}

var smallShotImages = [4][3]string{
	{"forward-shot-0", "forward-shot-1", "forward-shot-2"},
	{"left-shot-0", "left-shot-1", "left-shot-2"},
	{"right-shot-0", "right-shot-1", "right-shot-2"},
	{"rear-shot-0", "rear-shot-1", "rear-shot-2"},
}

// AppendSmallWeaponShots emits the original basic, double, rear or side-shot
// pattern into caller-owned storage. Other weapons use their dedicated state
// machines rather than falling back to an ordinary bullet pattern.
func AppendSmallWeaponShots(dst []SmallShot, weapon WeaponSlot, shipX, shipY int) ([]SmallShot, error) {
	if weapon.Tier < 0 || weapon.Tier > 2 {
		return dst, fmt.Errorf("invalid small-weapon power tier")
	}
	appendShot := func(variant, dx, dy, vx, vy int) {
		dst = append(dst, SmallShot{X: shipX + dx, Y: shipY + dy, VelocityX: vx, VelocityY: vy, Damage: uint16(weapon.Tier + 1), SpriteName: smallShotImages[variant][weapon.Tier]})
	}
	switch weapon.Item {
	case ItemForwardShot:
		appendShot(0, 0, -6, 0, -9)
	case ItemDoubleShot:
		appendShot(0, -5, -12, 0, -9)
		appendShot(0, 6, -12, 0, -9)
	case ItemRearShot:
		appendShot(3, 0, 18, 0, 9)
	case ItemSideShot:
		appendShot(1, -2, -3, -9, 0)
		appendShot(2, 2, -3, 9, 0)
	default:
		return dst, fmt.Errorf("item %d needs its dedicated weapon emitter", weapon.Item)
	}
	return dst, nil
}

// Advance preserves whole-pixel movement and the ordinary player-bullet
// clipping bounds. All ordinary player bullets are removed when the ship dies.
func (s *SmallShot) Advance(shipDestroyed bool) bool {
	if shipDestroyed {
		return false
	}
	s.X += s.VelocityX
	s.Y += s.VelocityY
	return s.X >= 0 && s.X < 312 && s.Y >= 0 && s.Y < 192
}
