package engine

import "xenon2/internal/visualassets"

// CollisionRect uses inclusive pixel edges, as the original actor comparisons
// do. A rectangle with its maximum before its minimum is empty.
type CollisionRect struct {
	Left, Top, Right, Bottom int
}

func (r CollisionRect) Empty() bool {
	return r.Right < r.Left || r.Bottom < r.Top
}

// Intersects includes contact on a shared edge or corner.
func (r CollisionRect) Intersects(other CollisionRect) bool {
	return !r.Empty() && !other.Empty() && r.Right >= other.Left && r.Left <= other.Right && r.Bottom >= other.Top && r.Top <= other.Bottom
}

func (r CollisionRect) Contains(x, y int) bool {
	return !r.Empty() && x >= r.Left && x <= r.Right && y >= r.Top && y <= r.Bottom
}

// ActorCollisionRect positions an exported collision prefix at the actor's
// anchor. Transparent artwork pixels do not change this actor rectangle.
func ActorCollisionRect(box visualassets.CollisionBox, x, y int) CollisionRect {
	left, top := x+box.X, y+box.Y
	return CollisionRect{Left: left, Top: top, Right: left + box.Width - 1, Bottom: top + box.Height - 1}
}

// ShieldDamage reports the damage state after the original shield rule.
type ShieldDamage struct {
	Shield    int
	Lost      int
	Applied   bool
	Destroyed bool
}

// ApplyShieldDamage preserves the original protection and destruction
// thresholds. Callers separately suppress collisions during invulnerability,
// diving or the pending level-exit sequence.
func ApplyShieldDamage(shield, amount int, protected, suppressed bool) ShieldDamage {
	result := ShieldDamage{Shield: shield}
	if suppressed {
		return result
	}
	effective := uint16(amount)
	if protected {
		effective >>= 1
	}
	before := uint16(shield)
	result.Applied = true
	if before <= effective {
		result.Shield = 0
		result.Lost = int(before)
		result.Destroyed = true
	} else {
		result.Shield = int(before - effective)
		result.Lost = int(effective)
	}
	return result
}

// EnemyDamage retains the original unsigned health subtraction, including the
// wrapped value after lethal damage. Removal is handled by the world.
type EnemyDamage struct {
	Health    uint16
	Destroyed bool
}

// ApplyEnemyDamage is the ordinary individual or group-leader damage rule.
// Level-specific guardians can add their own vulnerability conditions.
func ApplyEnemyDamage(health uint16, amount uint16) EnemyDamage {
	return EnemyDamage{Health: health - amount, Destroyed: health <= amount}
}

// ContactDamage distinguishes heavy enemies from ordinary moving enemies.
func ContactDamage(heavy bool) int {
	if heavy {
		return 16
	}
	return 8
}

const EnemyBulletDamage = 4
