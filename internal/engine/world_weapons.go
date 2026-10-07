package engine

import (
	"fmt"

	"xenon2/internal/visualassets"
)

// WeaponContext supplies source-ordered targets, the shared random stream and
// collision callbacks. It contains no renderer or original program state.
type WeaponContext struct {
	Equipment                                         *Equipment
	ShipX, ShipY, PreviousShipX, PreviousShipY        int
	ShipCenterX, ShipCenterY, MaterializationFrames   int
	TrailX, TrailY                                    int
	Motion                                            MotionInput
	Held, Pulse, Diving, Materializing, ShipDestroyed bool
	NextRandom                                        func() uint32
	NextID                                            func() int
	NextActorDriftResidue                             func() uint16
	SkipSmallWeapons                                  bool
	AvailableActorSlots                               int
	Targets                                           []WeaponTarget
	HitPoint                                          func(int, int, uint16) bool
	HitRect                                           func(CollisionRect, uint16, bool) bool
	HitLaser                                          func(CollisionRect, uint16) bool
	Sound                                             func(string)
	SoundVoice                                        func(int, string)
	SoundVoiceIfEmpty                                 func(int, string)
	ImmediateSoundVoice                               func(int, string)
	EffectActive                                      func(int) bool
	StopEffects                                       func()
	ReserveActor                                      func(int16, ActorPoolList, bool) (ActorPoolBinding, error)
	RetireActor                                       func(ActorPoolBinding)
	StoreActorResidue                                 func(ActorPoolBinding)
	ReadActorResidue                                  func(int) (ActorResidue, bool)
}

// WeaponRenderItem is a named attachment or projectile at a display anchor.
// Laser and procedural sparks keep their dedicated geometry fields.
type WeaponRenderItem struct {
	ID                         int
	Kind, Sprite               string
	X, Y, PreviousX, PreviousY float64
	Tier, Length               int
	Active                     bool
}

type runtimeMount struct {
	Binding, SupportBinding                                ActorPoolBinding
	Serial                                                 uint32
	Item                                                   Item
	Animation                                              AnimationState
	AnimationData                                          visualassets.NamedActorAnimation
	Cannon                                                 CannonMountState
	Launcher                                               MissileMountState
	Laser                                                  LaserMountState
	Drone                                                  DroneMountState
	Electro                                                ElectroBallState
	Mine                                                   MineState
	Bomb                                                   BombMountState
	Homing                                                 HomingMountState
	FlamerSound                                            FlamerSoundState
	X, Y                                                   int
	PreviousX, PreviousY                                   int
	RearPending                                            bool
	Visible                                                bool
	SupportAnimation                                       AnimationState
	SupportData                                            visualassets.NamedActorAnimation
	SupportX, SupportY, PreviousSupportX, PreviousSupportY int
	SupportVisible                                         bool
	SupportActive                                          bool
}

type runtimeWeaponProjectile struct {
	Binding       ActorPoolBinding
	Render        WeaponRenderItem
	Animation     AnimationState
	AnimationData visualassets.NamedActorAnimation
	Small         SmallShot
	Cannon        CannonBallMotion
	Laser         LaserBeamState
	Flame         FlameShot
	Spark         SparkShot
	Mine          MineState
	Bomb          BombState
	Homing        HomingMissileState
	Owner         int
	SparkSlot     int
}

// WeaponRuntime owns the original equipment-order state and its projectiles.
// Equipment and projectile phases are separate so the world can retain its
// player/enemy/equipment/projectile ordering.
type WeaponRuntime struct {
	mounts          [7]runtimeMount
	projectiles     []runtimeWeaponProjectile
	animations      map[string]visualassets.NamedActorAnimation
	boxes           map[string]visualassets.CollisionBox
	regions         map[string]visualassets.SpriteRegion
	MineContext     MineContext
	nextID          int
	newID           func() int
	available       int
	limited         bool
	order           [7]int
	context         WeaponContext
	allocationError error
}

func equipmentResourceTag(item Item) int16 {
	switch item {
	case ItemForwardShot:
		return 160
	case ItemDoubleShot:
		return 24
	case ItemRearShot:
		return 68
	case ItemSideShot:
		return 152
	case ItemCannon:
		return 52
	case ItemMissileLauncher:
		return 144
	case ItemLaser:
		return 32
	case ItemFlamer:
		return 28
	case ItemDrone:
		return 44
	case ItemElectroBall:
		return 76
	case ItemMineSmall:
		return 56
	case ItemMineLarge:
		return 60
	case ItemBomb:
		return 40
	case ItemHomingMissile:
		return 72
	}
	return 0
}

func weaponProjectileTag(kind string, tier int) int16 {
	switch kind {
	case "small-shot", "launcher-missile":
		return 16
	case "cannon-ball":
		return 48
	case "laser":
		return 36
	case "flame":
		return 180
	case "mine":
		if tier == 0 {
			return 56
		}
		return 60
	case "bomb":
		return 40
	case "homing":
		return 72
	case "explosion":
		return 12
	}
	return 0
}

func weaponSound(c WeaponContext, voice int, effect string) {
	if c.SoundVoice != nil {
		c.SoundVoice(voice, effect)
	} else if c.Sound != nil {
		c.Sound(effect)
	}
}

func NewWeaponRuntime(common *visualassets.SpriteAtlas) (*WeaponRuntime, error) {
	if common == nil {
		return nil, fmt.Errorf("weapon runtime needs common artwork")
	}
	r := &WeaponRuntime{animations: make(map[string]visualassets.NamedActorAnimation), boxes: make(map[string]visualassets.CollisionBox), regions: make(map[string]visualassets.SpriteRegion), MineContext: MineContext{LastX: -100}}
	for _, a := range common.Animations {
		r.animations[a.ID] = a
	}
	for _, s := range common.Sprites {
		r.regions[s.Name] = s
		if s.Collision != nil {
			r.boxes[s.Name] = *s.Collision
		}
	}
	r.projectiles = make([]runtimeWeaponProjectile, 0, 239)
	return r, nil
}

func weaponAnimation(item Item) string {
	switch item {
	case ItemDoubleShot:
		return "double-shot-active"
	case ItemRearShot:
		return "rear-shot-active"
	case ItemCannon:
		return "cannon-idle"
	case ItemMissileLauncher:
		return "launcher-idle"
	case ItemLaser:
		return "laser-active"
	case ItemFlamer:
		return "flamer-active"
	case ItemDrone:
		return "drone-active"
	case ItemElectroBall:
		return "electro-ball-active"
	case ItemMineSmall:
		return "mine-small-active"
	case ItemMineLarge:
		return "mine-large-active"
	}
	return ""
}

func (r *WeaponRuntime) synchronize(c WeaponContext) error {
	slots := c.Equipment.slots()
	order := [7]int{0, 1, 2, 3, 4, 5, 6}
	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && slots[order[j]].Serial < slots[order[j-1]].Serial; j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}
	for _, i := range order {
		slot := slots[i]
		m := &r.mounts[i]
		if m.Serial == slot.Serial && m.Item == slot.Item {
			m.Mine.Tier = slot.Tier
			if slot.Item != ItemNone {
				r.storeMount(c, m, *slot)
			}
			continue
		}
		if c.RetireActor != nil {
			if m.Binding.EntityID != 0 {
				c.RetireActor(m.Binding)
			}
			if m.SupportActive && m.SupportBinding.EntityID != 0 {
				c.RetireActor(m.SupportBinding)
			}
		}
		*m = runtimeMount{Binding: ActorPoolBinding{Slot: NoActorSlot}, SupportBinding: ActorPoolBinding{Slot: NoActorSlot}, Serial: slot.Serial, Item: slot.Item, Laser: NewLaserMountState(), Bomb: NewBombMountState(), Homing: NewHomingMountState(), Mine: MineState{Tier: slot.Tier}}
		if slot.Item != ItemNone && c.ReserveActor != nil {
			binding, err := c.ReserveActor(equipmentResourceTag(slot.Item), ActorPoolEquipment, false)
			if err != nil {
				return err
			}
			m.Binding = binding
			m.FlamerSound.Started = binding.Residue.Counter != 0
			m.FlamerSound.Counter = binding.Residue.Counter
		}
		if a, ok := r.animations[weaponAnimation(slot.Item)]; ok {
			m.AnimationData = a
			m.Animation = NewAnimation(a.Animation)
		}
		if slot.Item == ItemCannon {
			if a, ok := r.animations["cannon-support"]; ok {
				m.SupportData, m.SupportAnimation = a, NewAnimation(a.Animation)
			}
			m.SupportActive = true
			if c.ReserveActor != nil {
				binding, err := c.ReserveActor(64, ActorPoolProjectile, true)
				if err != nil {
					return err
				}
				m.SupportBinding = binding
				m.SupportBinding.Residue.OwnerSlot = m.Binding.Slot
			}
		}
		m.Electro.X, m.Electro.Y = c.ShipX, c.ShipY+25
		m.Mine.X, m.Mine.Y = c.ShipX, c.ShipY+25
		m.X, m.Y = c.ShipX, c.ShipY
		if i >= 1 && i <= 4 {
			m.X += WeaponMountOffset[i-1].X
			m.Y += WeaponMountOffset[i-1].Y
		}
		if slot.Item == ItemDrone || slot.Item == ItemElectroBall || slot.Item == ItemMineSmall || slot.Item == ItemMineLarge {
			m.Y += 25
		}
		m.SupportX, m.SupportY = m.X, m.Y
		m.PreviousX, m.PreviousY, m.PreviousSupportX, m.PreviousSupportY = m.X, m.Y, m.SupportX, m.SupportY
		m.Binding.Residue.EmitterClock = 0xffff
		r.storeMount(c, m, *slot)
	}
	return nil
}

// SynchronizeEquipment applies construction immediately after pickups or shop
// trades, without advancing animation, fire timers or projectile movement.
func (r *WeaponRuntime) SynchronizeEquipment(c WeaponContext) error {
	if c.Equipment == nil {
		return fmt.Errorf("weapon construction needs equipment")
	}
	return r.synchronize(c)
}

func (r *WeaponRuntime) add(kind, animation string, x, y, tier int) *runtimeWeaponProjectile {
	if kind != "spark" && r.limited && r.available <= 0 {
		return nil
	}
	sparkSlot := -1
	if kind == "spark" {
		var used [80]bool
		for _, p := range r.projectiles {
			if p.Render.Active && p.Render.Kind == "spark" {
				used[p.SparkSlot] = true
			}
		}
		for i, occupied := range used {
			if !occupied {
				sparkSlot = i
				break
			}
		}
		if sparkSlot < 0 {
			return nil
		}
	}
	countActors := 0
	for _, existing := range r.projectiles {
		if existing.Render.Active && existing.Render.Kind != "spark" {
			countActors++
		}
	}
	if kind != "spark" && r.context.ReserveActor == nil && countActors >= 159 {
		return nil
	}
	id := -1001 - sparkSlot
	binding := ActorPoolBinding{Slot: NoActorSlot}
	if kind != "spark" {
		if r.context.ReserveActor != nil {
			var err error
			binding, err = r.context.ReserveActor(weaponProjectileTag(kind, tier), ActorPoolProjectile, false)
			if err != nil {
				r.allocationError = err
				return nil
			}
			id = binding.EntityID
		} else {
			r.nextID++
			if r.newID != nil {
				r.nextID = r.newID()
			}
			id = r.nextID
		}
	}
	if r.limited && kind != "spark" {
		r.available--
	}
	p := runtimeWeaponProjectile{Binding: binding, SparkSlot: sparkSlot, Render: WeaponRenderItem{ID: id, Kind: kind, X: float64(x), Y: float64(y), PreviousX: float64(x), PreviousY: float64(y), Tier: tier, Active: true}}
	if a, ok := r.animations[animation]; ok {
		p.AnimationData = a
		p.Animation = NewAnimation(a.Animation)
		p.Render.Sprite = p.Animation.Sprite(a.Animation)
	}
	r.projectiles = append(r.projectiles, p)
	return &r.projectiles[len(r.projectiles)-1]
}

func (r *WeaponRuntime) has(kind string) bool {
	for _, p := range r.projectiles {
		if p.Render.Active && p.Render.Kind == kind {
			return true
		}
	}
	return false
}

// AdvanceEquipment creates source weapon emissions. Projectiles first move in
// the following projectile phase, including emissions created in this pass.
func (r *WeaponRuntime) AdvanceEquipment(c WeaponContext) error {
	if c.Equipment == nil {
		return fmt.Errorf("weapon phase needs equipment")
	}
	r.context, r.allocationError = c, nil
	if err := r.synchronize(c); err != nil {
		return err
	}
	r.newID = c.NextID
	r.limited = c.AvailableActorSlots > 0
	r.available = c.AvailableActorSlots
	r.order = [7]int{0, 1, 2, 3, 4, 5, 6}
	order := &r.order
	slots := c.Equipment.slots()
	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && slots[order[j]].Serial > slots[order[j-1]].Serial; j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}
	for _, index := range order {
		slot, m := slots[index], &r.mounts[index]
		if slot.Item == ItemNone {
			continue
		}
		m.Animation.Advance(m.AnimationData.Animation)
		m.Visible = true
		if m.RearPending {
			m.RearPending = false
			if a, ok := r.animations["rear-shot-fire"]; ok {
				m.AnimationData, m.Animation = a, NewAnimation(a.Animation)
			}
		}
		m.PreviousX, m.PreviousY = m.X, m.Y
		m.X, m.Y = c.ShipX, c.ShipY
		if index >= 1 && index <= 4 {
			offset := WeaponMountOffset[index-1]
			m.X += offset.X
			m.Y += offset.Y
		}
		switch slot.Item {
		case ItemForwardShot, ItemDoubleShot, ItemSideShot, ItemRearShot:
			if c.SkipSmallWeapons || !c.Pulse || c.Diving || c.Materializing && slot.Item != ItemForwardShot {
				r.materializeMount(m, slot.Item, c)
				r.storeMount(c, m, *slot)
				continue
			}
			if slot.Item == ItemRearShot && m.Animation.Remaining == 0 {
				m.RearPending = true
			}
			var storage [2]SmallShot
			shots, err := AppendSmallWeaponShots(storage[:0], *slot, c.ShipX, c.ShipY)
			if err != nil {
				return err
			}
			for _, shot := range shots {
				if p := r.add("small-shot", "", shot.X, shot.Y, slot.Tier); p != nil {
					p.Small = shot
					p.Render.Sprite = shot.SpriteName
					r.storeProjectile(c, p, false)
				}
			}
			if c.SoundVoiceIfEmpty != nil {
				c.SoundVoiceIfEmpty(2, "sampled-effect-09")
			} else {
				weaponSound(c, 2, "sampled-effect-09")
			}
		case ItemCannon:
			m.PreviousSupportX, m.PreviousSupportY = m.SupportX, m.SupportY
			m.SupportX, m.SupportY, m.SupportVisible = m.X, m.Y, m.SupportActive
			m.SupportAnimation.Advance(m.SupportData.Animation)
			if c.MaterializationFrames != 0 {
				if region, ok := r.regions[m.SupportAnimation.Sprite(m.SupportData.Animation)]; ok {
					m.SupportX, m.SupportY, m.SupportVisible = MaterializeAttachment(m.SupportX, m.SupportY, region, c.ShipCenterX, c.ShipCenterY, c.MaterializationFrames)
				}
			}
			fired := m.Cannon.Tick(c.Pulse, c.Materializing)
			if m.Cannon.Phase == 1 {
				if a, ok := r.animations["cannon-fire"]; ok {
					m.AnimationData = a
					m.Animation = NewAnimation(a.Animation)
				}
				weaponSound(c, 1, "sampled-effect-00")
			}
			if fired {
				if p := r.add("cannon-ball", "cannon-ball", m.X, m.Y-11, 0); p != nil {
					p.Cannon = CannonBallMotion{X: m.X, Y: m.Y - 11}
					r.storeProjectile(c, p, false)
				}
			}
		case ItemMissileLauncher:
			previousPhase := m.Launcher.Phase
			fired := m.Launcher.Tick(c.Pulse, c.Materializing)
			if previousPhase == 0 && m.Launcher.Phase == 1 {
				if a, ok := r.animations["launcher-fire"]; ok {
					m.AnimationData, m.Animation = a, NewAnimation(a.Animation)
				}
			}
			if fired {
				var storage [2]SmallShot
				for _, shot := range AppendLauncherMissiles(storage[:0], m.X, m.Y, "") {
					if p := r.add("launcher-missile", "launcher-flight", shot.X, shot.Y, 0); p != nil {
						p.Small = shot
						r.storeProjectile(c, p, false)
					}
				}
			}
		case ItemLaser:
			if m.Laser.Tick(c.Pulse, c.Materializing) {
				weaponSound(c, 1, "synthesized-effect-06")
				if p := r.add("laser", "", m.X, m.Y, slot.Tier); p != nil {
					p.Laser = NewLaserBeamState(slot.Tier)
					p.Owner = index
					p.Binding.Residue.OwnerSlot = m.Binding.Slot
					r.storeProjectile(c, p, false)
				}
			}
		case ItemDrone:
			m.X, m.Y = c.TrailX, c.TrailY+25
			if m.Drone.Tick(c.Held, c.Materializing) {
				var storage [12]SparkShot
				shots, err := AppendDroneSparks(storage[:0], slot.Tier, m.X, m.Y)
				if err != nil {
					return err
				}
				for _, shot := range shots {
					if p := r.add("spark", "", shot.X, shot.Y, slot.Tier); p != nil {
						p.Spark = shot
					}
				}
			}
		case ItemFlamer:
			effectActive := m.FlamerSound.Started
			if c.EffectActive != nil {
				effectActive = c.EffectActive(1)
			}
			start, stop := m.FlamerSound.Advance(c.Held, c.Materializing, effectActive)
			if start {
				weaponSound(c, 1, "synthesized-effect-01")
			}
			if stop && c.StopEffects != nil {
				c.StopEffects()
			}
			if c.Held && !c.Materializing {
				if c.NextRandom == nil {
					return fmt.Errorf("flamer needs the world random stream")
				}
				for particle := 0; particle < 2; particle++ {
					p := r.add("flame", "flamer-particle", c.ShipX, c.ShipY, slot.Tier)
					y := c.ShipY - 16 - particle*6 + int(c.NextRandom()&3)
					drift := int16(int8(uint8(c.NextRandom())))
					residue := uint16(0)
					if p != nil {
						residue = p.Binding.Residue.HorizontalDriftRemainder
					}
					if c.ReserveActor == nil && c.NextActorDriftResidue != nil {
						residue = c.NextActorDriftResidue()
					}
					if p != nil {
						p.Binding.Residue.Direction = drift
						p.Flame = FlameShot{X: int32(c.ShipX)<<16 | 0x8000, Y: y, VelocityX: int32(drift)<<10 + int32(residue>>6), Tier: slot.Tier, Damaging: particle == 0}
						p.Render.X, p.Render.Y = float64(p.Flame.X)/65536, float64(y)
						p.Render.PreviousX, p.Render.PreviousY = p.Render.X, p.Render.Y
						r.storeProjectile(c, p, false)
					}
				}
			}
		case ItemElectroBall:
			if m.Electro.Advance(c.ShipX, c.ShipY, c.PreviousShipX, c.PreviousShipY, c.TrailX, c.TrailY, c.Held, c.Materializing) && c.HitRect != nil {
				if c.HitRect(r.actorRect(m.Animation.Sprite(m.AnimationData.Animation), m.Electro.X, m.Electro.Y), ElectroBallDamage, false) {
					weaponSound(c, 1, "synthesized-effect-21")
				}
			}
			m.X, m.Y = m.Electro.X, m.Electro.Y
		case ItemMineSmall, ItemMineLarge:
			mine, _, _, _ := m.Mine.Advance(c.ShipX, c.ShipY, c.TrailX, c.TrailY, c.Motion, c.Held, c.Materializing, c.ShipDestroyed, &r.MineContext)
			m.X, m.Y = m.Mine.X, m.Mine.Y
			if mine != nil {
				if p := r.add("mine", weaponAnimation(slot.Item), mine.X, mine.Y, slot.Tier); p != nil {
					p.Mine = *mine
					p.AnimationData, p.Animation = m.AnimationData, m.Animation
					if p.Animation.Frame < len(p.AnimationData.Animation.Frames) {
						p.Animation.Remaining = p.AnimationData.Animation.Frames[p.Animation.Frame].Duration
					}
					p.Render.Sprite = p.Animation.Sprite(p.AnimationData.Animation)
					r.storeProjectile(c, p, false)
				}
			}
		case ItemBomb:
			if m.Bomb.Tick(c.Pulse, c.Diving, r.has("bomb")) {
				b := NewBombState(c.ShipX, c.ShipY)
				if p := r.add("bomb", "bomb-flight", b.X, b.Y, 0); p != nil {
					p.Bomb = b
					r.storeProjectile(c, p, false)
				}
			}
		case ItemHomingMissile:
			if m.Homing.Tick(c.Pulse, c.Diving, r.has("homing")) {
				weaponSound(c, 1, "synthesized-effect-09")
				for _, direction := range []uint8{7, 5, 3, 1} {
					if p := r.add("homing", fmt.Sprintf("homing-%d", direction), c.ShipX, c.ShipY, 0); p != nil {
						p.Homing = NewHomingMissileState(c.ShipX, c.ShipY, direction)
						r.storeProjectile(c, p, false)
					}
				}
			}
		default:
			return fmt.Errorf("unsupported installed weapon %d", slot.Item)
		}
		r.materializeMount(m, slot.Item, c)
		r.storeMount(c, m, *slot)
	}
	return r.allocationError
}

func (r *WeaponRuntime) materializeMount(m *runtimeMount, item Item, c WeaponContext) {
	if c.MaterializationFrames == 0 {
		return
	}
	if sprite := m.Animation.Sprite(m.AnimationData.Animation); sprite != "" {
		if region, ok := r.regions[sprite]; ok {
			m.X, m.Y, m.Visible = MaterializeAttachment(m.X, m.Y, region, c.ShipCenterX, c.ShipCenterY, c.MaterializationFrames)
			if item == ItemElectroBall {
				m.Electro.X, m.Electro.Y = m.X, m.Y
			}
			if item == ItemMineSmall || item == ItemMineLarge {
				m.Mine.X, m.Mine.Y = m.X, m.Y
			}
		}
	}
}

func (r *WeaponRuntime) actorRect(sprite string, x, y int) CollisionRect {
	box, ok := r.boxes[sprite]
	if !ok {
		return CollisionRect{Right: -1, Bottom: -1}
	}
	return ActorCollisionRect(box, x, y)
}

// AdvanceProjectiles queries current enemy bounds in newest-first order. The
// callbacks preserve the world's individual and linked-group damage behavior.
func (r *WeaponRuntime) AdvanceProjectiles(c WeaponContext) error {
	return r.advanceProjectiles(c, 0)
}

func (r *WeaponRuntime) advanceProjectiles(c WeaponContext, onlyID int) error {
	r.context, r.allocationError = c, nil
	for i := len(r.projectiles) - 1; i >= 0; i-- {
		if onlyID != 0 && r.projectiles[i].Render.ID != onlyID {
			continue
		}
		p := &r.projectiles[i]
		if !p.Render.Active {
			continue
		}
		if onlyID == 0 && p.Render.Kind == "spark" {
			continue
		}
		p.Render.PreviousX, p.Render.PreviousY = p.Render.X, p.Render.Y
		if len(p.AnimationData.Animation.Frames) != 0 && !p.Animation.AdvanceNamed(p.AnimationData) {
			p.Render.Active = false
			r.storeProjectile(c, p, true)
			continue
		}
		if len(p.AnimationData.Animation.Frames) != 0 {
			p.Render.Sprite = p.Animation.Sprite(p.AnimationData.Animation)
		}
		var area CollisionRect
		var damage uint16
		explosion := ""
		queryRect, queryPoint := false, false
		switch p.Render.Kind {
		case "small-shot", "launcher-missile":
			p.Render.Active = p.Small.Advance(c.ShipDestroyed)
			p.Render.X, p.Render.Y = float64(p.Small.X), float64(p.Small.Y)
			damage = p.Small.Damage
			queryPoint = p.Render.Active
		case "cannon-ball":
			p.Render.Active = p.Cannon.Advance(c.ShipDestroyed)
			p.Render.X, p.Render.Y = float64(p.Cannon.X), float64(p.Cannon.Y)
			area = r.actorRect(p.Render.Sprite, p.Cannon.X, p.Cannon.Y)
			damage = CannonBallDamage
			queryRect = p.Render.Active
		case "laser":
			m := &r.mounts[p.Owner]
			x, y := m.X, m.Y
			if c.ReadActorResidue != nil && p.Binding.Residue.OwnerSlot != NoActorSlot {
				if owner, ok := c.ReadActorResidue(p.Binding.Residue.OwnerSlot); ok {
					x, y = int(owner.X), int(owner.Y)
				}
			}
			p.Render.Active, area, damage = p.Laser.Advance(x, y, c.ShipDestroyed)
			p.Render.X, p.Render.Y = float64(p.Laser.X), float64(p.Laser.Y)
			p.Render.Length = p.Laser.Length
			queryRect = p.Render.Active
		case "spark":
			p.Render.Active = !c.ShipDestroyed && p.Spark.Advance()
			p.Render.X, p.Render.Y = float64(p.Spark.X), float64(p.Spark.Y)
			damage = p.Spark.Damage
			queryPoint = p.Render.Active
		case "flame":
			p.Render.Active = p.Flame.Advance(c.ShipDestroyed)
			p.Render.X, p.Render.Y = float64(p.Flame.X)/65536, float64(p.Flame.Y)
			damage = p.Flame.Damage()
			area = r.actorRect(p.Render.Sprite, int(p.Flame.X>>16), p.Flame.Y)
			queryRect = p.Render.Active && damage != 0
		case "mine":
			_, area, damage, p.Render.Active = p.Mine.Advance(0, 0, 0, 0, MotionInput{}, c.Held, false, c.ShipDestroyed, &r.MineContext)
			queryRect = !area.Empty()
			if queryRect {
				explosion = "mine-small"
				if p.Mine.Tier != 0 {
					explosion = "mine-large"
				}
			}
		case "bomb":
			p.Render.Active, area, damage = p.Bomb.Advance(c.ShipDestroyed)
			p.Render.X, p.Render.Y = float64(p.Bomb.X), float64(p.Bomb.Y)
			queryRect = !area.Empty()
			if queryRect {
				explosion = "bomb"
			}
		case "homing":
			previousDirection := p.Homing.Direction
			p.Render.Active = p.Homing.Advance(c.Targets, c.NextRandom, c.ShipDestroyed)
			if p.Homing.ExpiredWithoutTarget {
				explosion = "homing-expiry"
			}
			if p.Homing.Direction != previousDirection {
				if a, ok := r.animations[fmt.Sprintf("homing-%d", p.Homing.Direction)]; ok {
					p.AnimationData = a
					p.Animation = NewAnimation(a.Animation)
					p.Render.Sprite = p.Animation.Sprite(a.Animation)
				}
			}
			p.Render.X, p.Render.Y = float64(p.Homing.X), float64(p.Homing.Y)
			damage = 1
			queryPoint = p.Render.Active
		case "explosion":
			// The finite animation advances above; explosion art has no motion or hit query.
		default:
			return fmt.Errorf("unknown runtime projectile %q", p.Render.Kind)
		}
		if explosion != "" {
			var effects [4]WeaponExplosion
			items, err := AppendWeaponExplosions(effects[:0], explosion, int(p.Render.X), int(p.Render.Y), c.NextRandom)
			if err != nil {
				return err
			}
			for _, effect := range items {
				r.add("explosion", effect.Animation, effect.X, effect.Y, 0)
			}
			if explosion == "homing-expiry" {
				weaponSound(c, 2, "sampled-effect-05")
			} else {
				weaponSound(c, 1, "sampled-effect-03")
				weaponSound(c, 2, "sampled-effect-03")
				if c.ImmediateSoundVoice != nil {
					c.ImmediateSoundVoice(0, "sampled-effect-03")
				} else {
					weaponSound(c, 0, "sampled-effect-03")
				}
			}
			// Appending effects can grow storage; reacquire the current projectile.
			p = &r.projectiles[i]
		}
		if queryPoint && c.HitPoint != nil && c.HitPoint(int(p.Render.X), int(p.Render.Y), damage) {
			p.Render.Active = false
		}
		if queryRect && (c.HitRect != nil || p.Render.Kind == "laser" && c.HitLaser != nil) && !area.Empty() {
			if p.Render.Kind == "laser" && c.HitLaser != nil {
				if c.HitLaser(area, damage) {
					p.Render.Active = false
				}
			} else {
				hit := c.HitRect(area, damage, p.Render.Kind == "laser" || p.Render.Kind == "mine" || p.Render.Kind == "bomb")
				if hit && (p.Render.Kind == "cannon-ball" || p.Render.Kind == "flame") {
					p.Render.Active = false
				}
			}
		}
		r.storeProjectile(c, p, true)
	}
	if onlyID != 0 {
		return r.allocationError
	}
	kept := r.projectiles[:0]
	for _, p := range r.projectiles {
		if p.Render.Active {
			kept = append(kept, p)
		}
	}
	r.projectiles = kept
	return r.allocationError
}

// RenderState appends active named weapon state to caller-owned storage.
func (r *WeaponRuntime) RenderState(dst []WeaponRenderItem) []WeaponRenderItem {
	for _, i := range r.order {
		m := r.mounts[i]
		if m.Item == ItemNone || !m.Visible {
			continue
		}
		sprite := m.Animation.Sprite(m.AnimationData.Animation)
		if sprite != "" {
			id := m.Binding.EntityID
			if id == 0 {
				id = -i - 1
			}
			dst = append(dst, WeaponRenderItem{ID: id, Kind: "attachment", Sprite: sprite, X: float64(m.X), Y: float64(m.Y), PreviousX: float64(m.PreviousX), PreviousY: float64(m.PreviousY), Active: true})
		}
	}
	for _, p := range r.projectiles {
		dst = append(dst, p.Render)
	}
	for i, m := range r.mounts {
		if m.Item == ItemCannon && m.SupportActive && m.SupportVisible {
			if sprite := m.SupportAnimation.Sprite(m.SupportData.Animation); sprite != "" {
				id := m.SupportBinding.EntityID
				if id == 0 {
					id = -100 - i
				}
				dst = append(dst, WeaponRenderItem{ID: id, Kind: "cannon-support", Sprite: sprite, X: float64(m.SupportX), Y: float64(m.SupportY), PreviousX: float64(m.PreviousSupportX), PreviousY: float64(m.PreviousSupportY), Active: true})
			}
		}
	}
	return dst
}

// ProjectileIDs appends newest-first creation IDs for the world's merged phase.
func (r *WeaponRuntime) ProjectileIDs(dst []int) []int {
	for i := len(r.projectiles) - 1; i >= 0; i-- {
		if r.projectiles[i].Render.Active && r.projectiles[i].Render.Kind != "spark" {
			dst = append(dst, r.projectiles[i].Render.ID)
		}
	}
	return dst
}

func (r *WeaponRuntime) AdvanceProjectile(id int, c WeaponContext) error {
	return r.advanceProjectiles(c, id)
}

// AdvanceSparks visits the dedicated eighty-slot buffer in its original slot
// order. It runs after fixed enemies and timers, separately from actor bullets.
func (r *WeaponRuntime) AdvanceSparks(c WeaponContext) error {
	var active [80]int
	for i := range active {
		active[i] = -1
	}
	for i := range r.projectiles {
		p := &r.projectiles[i]
		if p.Render.Active && p.Render.Kind == "spark" {
			active[p.SparkSlot] = i
		}
	}
	for slot := range 80 {
		if i := active[slot]; i >= 0 {
			if err := r.AdvanceProjectile(r.projectiles[i].Render.ID, c); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *WeaponRuntime) Compact() {
	kept := r.projectiles[:0]
	for _, p := range r.projectiles {
		if p.Render.Active {
			kept = append(kept, p)
		}
	}
	r.projectiles = kept
}

func (r *WeaponRuntime) ResetProjectiles() {
	r.projectiles = r.projectiles[:0]
	r.MineContext = MineContext{LastX: -100}
}
