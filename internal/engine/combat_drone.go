package engine

import "fmt"

// DroneMountState retains the separate six-pass radial-shot cooldown.
type DroneMountState struct {
	Cooldown int
}

func (s *DroneMountState) Tick(held, materializing bool) bool {
	if materializing {
		return false
	}
	if s.Cooldown != 0 {
		s.Cooldown--
	}
	if s.Cooldown == 0 && held {
		s.Cooldown = 6
		return true
	}
	return false
}

// SparkShot is a three-pixel procedural projectile from the drone. Its collision
// is a point and its own palette-bit drawing is independent of actor artwork.
type SparkShot struct {
	X, Y, VelocityX, VelocityY int
	Damage                     uint16
}

var dronePatterns = [][][4]int{
	{{4, 4, 6, 6}, {-8, 4, -6, 6}, {-8, -8, -6, -6}, {4, -8, 6, -6}},
	{{6, -2, 8, 0}, {4, 4, 6, 6}, {-2, 6, 0, 8}, {-8, 4, -6, 6}, {-10, -2, -8, 0}, {-8, -8, -6, -6}, {-2, -10, 0, -8}, {4, -8, 6, -6}},
	{{6, -2, 8, 0}, {5, 2, 7, 4}, {2, 5, 4, 7}, {-2, 6, 0, 8}, {-6, 5, -4, 7}, {-9, 2, -7, 4}, {-10, -2, -8, 0}, {-9, -6, -7, -4}, {-6, -9, -4, -7}, {-2, -10, 0, -8}, {2, -9, 4, -7}, {5, -6, 7, -4}},
}

// AppendDroneSparks preserves the original offsets, integer velocities and
// creation order. The world retains the original shared eighty-spark capacity.
func AppendDroneSparks(dst []SparkShot, tier, droneX, droneY int) ([]SparkShot, error) {
	if tier < 0 || tier >= len(dronePatterns) {
		return dst, fmt.Errorf("invalid drone power tier")
	}
	for _, p := range dronePatterns[tier] {
		dst = append(dst, SparkShot{X: droneX + p[0], Y: droneY + p[1], VelocityX: p[2], VelocityY: p[3], Damage: 1})
	}
	return dst, nil
}

func (s *SparkShot) Advance() bool {
	s.X += s.VelocityX
	s.Y += s.VelocityY
	return s.X >= 0 && s.X < 316 && s.Y >= 0 && s.Y < 188
}
