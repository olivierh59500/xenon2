package engine

func (r *WeaponRuntime) storeMount(c WeaponContext, m *runtimeMount, slot WeaponSlot) {
	m.Binding.Residue.X, m.Binding.Residue.Y = int16(m.X), int16(m.Y)
	m.Binding.Residue.Health, m.Binding.Residue.PowerOrScore, m.Binding.Residue.WaveBonusToken = uint16(slot.MaxTier), uint16(slot.Tier), uint16(slot.Item)
	switch slot.Item {
	case ItemDrone:
		m.Binding.Residue.Counter = int16(m.Drone.Cooldown)
	case ItemElectroBall:
		m.Binding.Residue.Counter = int16(m.Electro.Mode)
	case ItemMineSmall, ItemMineLarge:
		m.Binding.Residue.Counter = int16(m.Mine.Mode)
	case ItemFlamer:
		m.Binding.Residue.Counter = m.FlamerSound.Counter
	case ItemLaser:
		m.Binding.Residue.EmitterClock = uint16(m.Laser.Cooldown)
	case ItemBomb:
		m.Binding.Residue.EmitterClock = uint16(m.Bomb.Cooldown)
	case ItemHomingMissile:
		m.Binding.Residue.EmitterClock = uint16(m.Homing.Cooldown)
	}
	if c.StoreActorResidue != nil && m.Binding.EntityID != 0 {
		c.StoreActorResidue(m.Binding)
	}
	if m.SupportActive && m.SupportBinding.EntityID != 0 {
		m.SupportBinding.Residue.X, m.SupportBinding.Residue.Y = int16(m.SupportX), int16(m.SupportY)
		if c.StoreActorResidue != nil {
			c.StoreActorResidue(m.SupportBinding)
		}
	}
}

func (r *WeaponRuntime) storeProjectile(c WeaponContext, p *runtimeWeaponProjectile, retire bool) {
	if p.Render.Kind == "spark" || p.Binding.EntityID == 0 {
		return
	}
	residue := &p.Binding.Residue
	residue.X, residue.Y = int16(p.Render.X), int16(p.Render.Y)
	switch p.Render.Kind {
	case "small-shot", "launcher-missile":
		residue.Direction, residue.VerticalVelocity = int16(p.Small.VelocityX), int16(p.Small.VelocityY)
		residue.Counter, residue.PowerOrScore = 0, uint16(p.Render.Tier)
	case "cannon-ball":
		residue.Counter = 0
	case "laser":
		residue.Counter, residue.PowerOrScore = int16(p.Laser.Length), uint16(p.Laser.Tier)
	case "flame":
		residue.X = int16(p.Flame.X >> 16)
		residue.XFraction = uint16(p.Flame.X)
		residue.PowerOrScore = uint16(p.Flame.Tier)
		residue.VerticalFraction = 0
		if p.Flame.Damaging {
			residue.VerticalFraction = 1
		}
	case "mine":
		residue.Counter, residue.PowerOrScore = int16(p.Mine.Mode), uint16(p.Mine.Tier)
	case "bomb":
		residue.Counter = int16(p.Bomb.Timer)
	case "homing":
		residue.Counter, residue.Direction = int16(p.Homing.TurnTimer), int16(p.Homing.Direction)
	}
	if c.StoreActorResidue != nil {
		c.StoreActorResidue(p.Binding)
	}
	if retire && !p.Render.Active && c.RetireActor != nil {
		c.RetireActor(p.Binding)
		p.Binding.EntityID = 0
	}
}

// DropActor removes only the displaced presentation/state. Allocation itself
// does not run damage, reward, score or other gameplay death callbacks.
func (r *WeaponRuntime) DropActor(entityID int) {
	for i := range r.projectiles {
		p := &r.projectiles[i]
		if p.Render.ID == entityID {
			p.Render.Active, p.Binding.EntityID = false, 0
		}
	}
	for _, mounts := range []*[7]runtimeMount{&r.mounts, &r.savedMounts} {
		for i := range mounts {
			m := &mounts[i]
			if m.SupportBinding.EntityID == entityID {
				m.SupportActive, m.SupportVisible, m.SupportBinding.EntityID = false, false, 0
			}
		}
	}
}
