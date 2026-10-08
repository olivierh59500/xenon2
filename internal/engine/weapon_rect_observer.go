package engine

// WeaponRectImpact describes the rectangle immediately before its native hit
// callback. Observers may inspect the forecast but must never modify it.
// OwnerSlot is a logical equipment position; -1 means attribution is not yet
// supported for that family. Cannon and laser emissions supply it explicitly.
type WeaponRectImpact struct {
	ProjectileID, OwnerSlot int
	Kind                    string
	Area                    CollisionRect
	Damage                  uint16
	AllTargets, Laser       bool
}
