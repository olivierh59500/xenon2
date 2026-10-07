package engine

import "fmt"

// WeaponExplosion is a finite visual effect at an integer display anchor.
type WeaponExplosion struct {
	X, Y      int
	Animation string
}

// AppendWeaponExplosions preserves the original alternating X/Y random draws.
// The animation's rectangle is independent of the weapon's damage rectangle.
func AppendWeaponExplosions(dst []WeaponExplosion, kind string, x, y int, nextRandom func() uint32) ([]WeaponExplosion, error) {
	count, left, top, width, height, animation := 0, x, y, 0, 0, ""
	switch kind {
	case "mine-small":
		count, left, top, width, height, animation = 3, x-8, y-8, 16, 16, "explosion-small"
	case "mine-large":
		count, left, top, width, height, animation = 3, x-18, y-18, 20, 20, "explosion-large"
	case "bomb":
		count, left, top, width, height, animation = 4, x-20, y-20, 40, 40, "explosion-large"
	case "homing-expiry":
		return append(dst, WeaponExplosion{X: x, Y: y, Animation: "explosion-small"}), nil
	default:
		return dst, fmt.Errorf("unknown weapon explosion %q", kind)
	}
	if nextRandom == nil {
		return dst, fmt.Errorf("weapon explosion needs the shared random stream")
	}
	for range count {
		dx := int((nextRandom()>>2)&0xffff) % width
		dy := int((nextRandom()>>2)&0xffff) % height
		dst = append(dst, WeaponExplosion{X: left + dx, Y: top + dy, Animation: animation})
	}
	return dst, nil
}
